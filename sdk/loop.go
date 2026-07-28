package sdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/duynguyendang/manglekit/core"
	engine_memory "github.com/duynguyendang/manglekit/internal/engine/memory"
)

// Define constants or config struct
const (
	DefaultMaxSteps   = 10
	DefaultMaxRetries = 3
	BackoffBase       = 100 * time.Millisecond
)

// WithMaxSteps sets the maximum number of loop iterations allowed.
// Zero uses the default (DefaultMaxSteps = 10).
func WithMaxSteps(n int) ExecuteOption {
	return func(p *ExecutionParams) {
		if n > 0 {
			p.MaxSteps = n
		}
	}
}

// ExecuteByName executes a registered action by its name, handling the Semantic State Machine loop.
func (c *Client) ExecuteByName(ctx context.Context, actionName string, input any, opts ...ExecuteOption) (core.Envelope, error) {
	params := ExecutionParams{
		MemoryMode: core.MemoryModeNone,
	}
	for _, opt := range opts {
		opt(&params)
	}
	return c.runLoopInternal(ctx, actionName, input, params)
}

// runLoopInternal implements the core Semantic State Machine loop.
// It iterates through steps, managing memory storage, handling decisions (Retry/Route),
// and enforcing the execution limits (max steps, timeouts).
func (c *Client) runLoopInternal(ctx context.Context, startAction string, payload any, params ExecutionParams) (core.Envelope, error) {
	ctx = core.ContextWithLogger(ctx, c.logger)

	// 1. Determine Store Strategy
	switch params.MemoryMode {
	case core.MemoryModePersist:
		params.Store = c.agentMemory
		if params.SessionID != "" {
			var err error
			// Load initial history
			params.CurrentHistory, err = params.Store.Read(ctx, params.SessionID)
			if err != nil && c.logger != nil {
				c.logger.Warn("RunLoop failed to hydrate history", "error", err)
			}
		}
	case core.MemoryModeTransient:
		params.Store = &engine_memory.VolatileStore{}
	default:
		params.Store = &core.NopStore{}
	}

	// 2. Hydrate from durable state if available
	// This restores the full execution context including logical facts
	if c.stateManager != nil && params.SessionID != "" {
		state, err := c.stateManager.Hydrate(ctx, params.SessionID)
		if err == nil && state != nil {
			// Restore execution context
			params.CurrentHistory = state.ExecutionCtx.CurrentHistory
			params.FeedbackHistory = state.ExecutionCtx.FeedbackHistory
			params.RetryCount = state.ExecutionCtx.RetryCount

			// Restore envelope as starting payload
			// The envelope contains the last successful state
			payload = state.ActiveEnvelope.Payload

			c.logger.Info("Hydrated session state",
				"session_id", params.SessionID,
				"retry_count", params.RetryCount,
				"history_length", len(params.CurrentHistory))
		} else if err != nil {
			c.logger.Warn("Failed to hydrate durable state", "error", err)
		}
	}

	currentAction := startAction
	currentPayload := payload

	maxSteps := params.MaxSteps
	if maxSteps <= 0 {
		maxSteps = c.maxSteps
	}
	if maxSteps <= 0 {
		maxSteps = DefaultMaxSteps
	}

	for step := 0; step < maxSteps; step++ {
		// Check context before starting new step
		if err := ctx.Err(); err != nil {
			return core.Envelope{}, err
		}

		c.logger.Info("RunLoop step", "step", step, "action", currentAction)

		result, err := c.ExecuteSingleStep(ctx, currentAction, currentPayload, &params)
		if err != nil {
			return core.Envelope{}, err
		}

		// Capture audit record for this step — enriched with rule/tier/latency data.
		outcome := ""
		if d, ok := result.Metadata[core.KeyDecision].(string); ok {
			outcome = d
		}
		if outcome == "" {
			outcome = core.DecisionProceed
		}
		var record core.AuditRecord
		if trail, ok := result.Metadata["manglekit.audit_trail"].(*core.AuditTrail); ok && trail != nil {
			record = core.NewAuditRecordFromTrail(trail, step, outcome)
		} else {
			record = core.AuditRecord{Step: step, Outcome: outcome}
		}
		params.AuditRecords = append(params.AuditRecords, record)

		decision := result.Metadata[core.KeyDecision]
		if decision == core.DecisionRoute {
			// Update flow for next loop
			next, ok := result.Metadata[core.KeyNextStep].(string)
			if !ok || next == "" {
				return core.Envelope{}, fmt.Errorf("route decision missing next_step")
			}

			// Validate/Log Payload Handover
			c.logger.Info("RunLoop: Routing to next action", "from", currentAction, "to", next, "payload_type", fmt.Sprintf("%T", result.Payload))

			currentAction = next
			currentPayload = result.Payload
			continue
		}

		if decision == core.DecisionRetry {
			// Params internal state (RetryCount, Feedback) already updated by ExecuteSingleStep
			// Validation (MaxRetries) and Backoff (Sleep) also handled by ExecuteSingleStep
			// Just loop again with same action/payload
			continue
		}

		// ALLOW or empty -> Done
		return result, nil
	}
	return core.Envelope{}, fmt.Errorf("max steps exceeded")
}

// ExecuteSingleStep runs one step of the action and returns the decision.
// It orchestrates: Context Injection → Execution → History → Decision Handling.
func (c *Client) ExecuteSingleStep(ctx context.Context, actionName string, payload any, params *ExecutionParams) (core.Envelope, error) {
	// 1. Resolve Action
	c.registryLock.RLock()
	action, ok := c.registry[actionName]
	c.registryLock.RUnlock()
	if !ok {
		return core.Envelope{}, fmt.Errorf("action not found: %s", actionName)
	}

	// 2. Create envelope and inject context
	env := core.NewEnvelope(payload)
	env.ContentType = action.Metadata().InputContentType
	c.injectContext(ctx, &env, payload, params)

	// 3. Execute action (includes Blueprint pre-check)
	// The supervisor chain calls AssessPlan → Assess → evaluateGateWithTrail,
	// which builds the audit trail. The trail travels with the result envelope
	// via metadata — no shared engine state needed.
	result, err := action.Execute(ctx, env)
	if err != nil {
		return c.handleExecutionError(ctx, err, payload, params)
	}
	params.LastFeedback = ""

	// 3b. Evaluate post-execution steering for route/retry decisions.
	// Gated by steeringEnabled — when false, prompts pass through unmodified.
	if c.steeringEnabled {
		if result.Metadata == nil {
			result.Metadata = make(map[string]any)
		}
		if _, exists := result.Metadata[core.KeyDecision]; !exists || result.Metadata[core.KeyDecision] == "" {
			if steeringDec, steeringMeta, err := c.engine.EvaluateSteering(ctx, result); err == nil && steeringDec != "" {
				result.Metadata[core.KeyDecision] = steeringDec
				for k, v := range steeringMeta {
					result.Metadata[k] = v
				}
			}
		}
	}

	// 4. Update and persist history
	c.updateHistory(ctx, payload, result, params)

	// 5. Checkpoint state after successful execution (Atomic Checkpoint)
	// Only checkpoint if decision is PROCEED to prevent persisting invalid states
	if c.stateManager != nil && params.SessionID != "" {
		decision := result.Metadata[core.KeyDecision]
		if decision == "" || decision == core.DecisionProceed {
			// Extract current facts from the result envelope
			facts, err := c.stateManager.ExtractFacts(ctx, result)
			if err != nil {
				c.logger.Warn("Failed to extract facts for checkpoint", "error", err)
				facts = []string{} // Continue with empty facts
			}

			// Create session state snapshot
			state := &core.SessionState{
				SessionID:      params.SessionID,
				ActiveEnvelope: result,
				ExecutionCtx: core.ExecutionContext{
					RetryCount:      params.RetryCount,
					FeedbackHistory: params.FeedbackHistory,
					CurrentHistory:  params.CurrentHistory,
				},
				LogicalFacts: facts,
				AuditRecords: params.AuditRecords,
			}

			// Persist the checkpoint
			if err := c.stateManager.Checkpoint(ctx, state); err != nil {
				c.logger.Warn("Failed to checkpoint state", "error", err)
				// Don't fail the execution, just log the warning
			}
		}
	}

	// 6. Handle decision (Retry/Route/Proceed/Halt)
	return c.handleDecision(ctx, actionName, result, payload, params)
}

// injectContext populates the envelope with feedback, history, RAG context, metadata, and facts.
func (c *Client) injectContext(ctx context.Context, env *core.Envelope, payload any, params *ExecutionParams) {
	// Inject feedback history
	if len(params.FeedbackHistory) > 0 {
		env.Metadata[core.KeyPrevFeedback] = strings.Join(params.FeedbackHistory, "; ")
	}
	if params.LastFeedback != "" {
		env.SetFeedback(params.LastFeedback)
		env.Metadata["mangle_feedback"] = params.LastFeedback
	}

	// Inject chat history
	if len(params.CurrentHistory) > 0 && params.MemoryMode != core.MemoryModeNone {
		if err := env.SetHistory(params.CurrentHistory); err != nil {
			c.logger.Warn("failed to set history on envelope", "error", err)
		}
	}

	// Inject semantic memory (RAG)
	c.recallContext(ctx, payload, env)

	// Inject explicit metadata
	for k, v := range params.Metadata {
		env.Metadata[k] = v
	}

	// Inject context facts
	for k, v := range core.ContextFacts(ctx) {
		env.Metadata[k] = v
	}
}

// handleExecutionError processes errors from action execution, handling alignment errors with retry logic.
func (c *Client) handleExecutionError(ctx context.Context, err error, payload any, params *ExecutionParams) (core.Envelope, error) {
	var alignErr *core.AlignmentError
	if !errors.As(err, &alignErr) {
		return core.Envelope{}, err
	}

	if params.RetryCount >= DefaultMaxRetries {
		return core.Envelope{}, fmt.Errorf("max retries exceeded: %w", err)
	}

	params.RetryCount++
	params.LastFeedback = alignErr.Message

	c.logger.Warn("RunLoop: Blueprint Alignment Issue", "feedback", params.LastFeedback, "attempt", params.RetryCount)

	if err := c.backoff(ctx, params.RetryCount); err != nil {
		return core.Envelope{}, err
	}

	// Signal RETRY to caller
	res := core.NewEnvelope(payload)
	res.Metadata[core.KeyDecision] = core.DecisionRetry
	return res, nil
}

// updateHistory appends the exchange to history and persists if configured.
func (c *Client) updateHistory(ctx context.Context, payload any, result core.Envelope, params *ExecutionParams) {
	if params.MemoryMode == core.MemoryModeNone {
		return
	}

	newExchange := []core.Message{
		{Role: "user", Content: safelyStringify(payload)},
		{Role: "assistant", Content: safelyStringify(result.Payload)},
	}
	params.CurrentHistory = append(params.CurrentHistory, newExchange...)

	// Persist if configured
	if params.Store != nil && params.SessionID != "" && params.MemoryMode == core.MemoryModePersist {
		if err := params.Store.Append(ctx, params.SessionID, params.CurrentHistory); err != nil && c.logger != nil {
			c.logger.Warn("RunLoop failed to persist history", "error", err)
		}
	}
}

// handleDecision processes the steering decision and returns the appropriate result.
func (c *Client) handleDecision(ctx context.Context, actionName string, result core.Envelope, payload any, params *ExecutionParams) (core.Envelope, error) {
	decision := result.Metadata[core.KeyDecision]

	// Memorize on success
	if decision == "" || decision == core.DecisionProceed {
		c.asyncMemorize(payload, result.Payload)
	}

	c.logger.Debug("RunLoop decision", "decision", decision, "action", actionName)

	switch decision {
	case core.DecisionRetry:
		return c.handleRetryDecision(ctx, actionName, result, params)

	case core.DecisionRoute:
		params.RetryCount = 0
		params.FeedbackHistory = nil
		c.logger.Info("RunLoop: Feedback history cleared for new action route")
		return result, nil

	case core.DecisionProceed, "":
		return result, nil

	case core.DecisionHalt:
		return core.Envelope{}, c.buildHaltError(result)
	}

	return result, nil
}

// handleRetryDecision processes a RETRY decision with backoff.
func (c *Client) handleRetryDecision(ctx context.Context, actionName string, result core.Envelope, params *ExecutionParams) (core.Envelope, error) {
	if params.RetryCount >= DefaultMaxRetries {
		return core.Envelope{}, fmt.Errorf("max retries exceeded for action %s", actionName)
	}

	hint := result.GetFeedback()

	// Semantic Thrashing Detection
	for _, prevFeedback := range params.FeedbackHistory {
		if isSemanticallySimilar(prevFeedback, hint) {
			c.logger.Warn("Semantic Thrashing detected", "new_feedback", hint, "prev_feedback", prevFeedback)
			return core.Envelope{}, fmt.Errorf("semantic thrashing detected: feedback loop on %q", hint)
		}
	}

	params.RetryCount++
	params.LastFeedback = hint
	params.FeedbackHistory = append(params.FeedbackHistory, hint)

	c.logger.Warn("RunLoop: RETRY triggered", "feedback", hint)

	if err := c.backoff(ctx, params.RetryCount); err != nil {
		return core.Envelope{}, err
	}

	return result, nil
}

// buildHaltError extracts the reason from metadata and builds the halt error.
func (c *Client) buildHaltError(result core.Envelope) error {
	reason := result.Metadata["reason"]
	if reason == "" {
		reason = result.Metadata["violation_msg"]
	}
	if reason == "" {
		reason = "blueprint violation"
	}
	return fmt.Errorf("action halted by blueprint: %s", reason)
}

// backoff handles the sleep and context cancellation check
func (c *Client) backoff(ctx context.Context, retryCount int) error {
	sleepDuration := time.Duration(retryCount) * BackoffBase
	timer := time.NewTimer(sleepDuration)
	select {
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	case <-timer.C:
		timer.Stop()
		return nil
	}
}

// ---------------------------------------------------------
// PRIVATE HELPERS (Memory & Context)
// ---------------------------------------------------------

// safelyStringify converts any payload to string for embedding.
func safelyStringify(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	// Fallback to generic formatting
	return fmt.Sprintf("%v", v)
}

// recallContext handles the RAG lookup logic.
// It fails silently (logs only) to not break the main flow.
func (c *Client) recallContext(ctx context.Context, payload any, env *core.Envelope) {
	if c.agentMemory == nil {
		return
	}

	// [GATE] Check if memory is required
	if c.engine != nil {
		// Ask: requires(Req, "memory") ?
		needed, err := c.engine.CheckRequirement(ctx, *env, "memory")
		if err != nil {
			c.logger.Warn("Engine check failed, skipping memory", "err", err)
			return
		}
		if !needed {
			return // Skip RAG if not required
		}
	}

	// Start Span for Observability
	var span core.Span
	if c.tracer != nil {
		ctx, span = c.tracer.Start(ctx, core.SpanMemory)
		defer span.End()
	}

	inputStr := safelyStringify(payload)

	// Call Memory Provider
	var contextData string
	var err error

	// Extended Interface Check
	if memWithFacts, ok := c.agentMemory.(core.AgentMemoryWithFacts); ok {
		var facts map[string]any
		contextData, facts, err = memWithFacts.RecallWithFacts(ctx, inputStr)
		if err == nil && len(facts) > 0 {
			// Merge facts into metadata
			for k, v := range facts {
				env.Metadata[k] = v
			}
		}
	} else {
		contextData, err = c.agentMemory.Recall(ctx, inputStr)
	}

	if err != nil {
		c.logger.Warn("Memory Recall failed", "error", err)
		if span != nil {
			span.RecordError(err)
		}
		return
	}

	// Inject if found
	if contextData != "" {
		env.SetMeta(core.KeyContext, contextData)
		c.logger.Debug("Injected memory context", "len", len(contextData))
	}
}

// asyncMemorize handles the Fire-and-Forget storage logic.
// After Shutdown is called, this is a no-op; in-flight operations
// from before Shutdown are awaited by Shutdown.
func (c *Client) asyncMemorize(input any, output any) {
	if c.agentMemory == nil {
		return
	}

	inputStr := safelyStringify(input)
	outputStr := safelyStringify(output)

	// Acquire the mutex to either Add(1) and proceed, or observe
	// shuttingDown and bail out. This guarantees that any Add happens
	// before Shutdown's Wait, satisfying sync.WaitGroup's contract.
	c.memMu.Lock()
	if c.shuttingDown {
		c.memMu.Unlock()
		return
	}
	c.memWg.Add(1)
	c.memMu.Unlock()

	go func(q, a string) {
		defer c.memWg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.agentMemory.Memorize(ctx, q, a); err != nil {
			c.logger.Warn("Memory Memorize failed", "error", err)
		}
	}(inputStr, outputStr)
}

// normalizeString lowercases, removes punctuation, and collapses spaces.
func normalizeString(s string) string {
	// Fast path for empty strings
	if s == "" {
		return ""
	}

	// 1. Lowercase and remove punctuation efficiently
	builder := strings.Builder{}
	builder.Grow(len(s))

	for _, r := range s {
		if unicode.IsPunct(r) {
			continue // Skip punctuation
		}
		builder.WriteRune(unicode.ToLower(r))
	}

	// 2. Collapse spaces (using strings.Fields is efficient enough for short error msgs)
	return strings.Join(strings.Fields(builder.String()), " ")
}

// isSemanticallySimilar checks if two strings are semantically equivalent.
func isSemanticallySimilar(s1, s2 string) bool {
	if s1 == s2 {
		return true
	}
	n1 := normalizeString(s1)
	n2 := normalizeString(s2)
	if n1 == n2 {
		return true
	}
	// Guard against empty strings matching everything via Contains
	if n1 == "" || n2 == "" {
		return false
	}
	// Check containment for verbosity
	return strings.Contains(n1, n2) || strings.Contains(n2, n1)
}
