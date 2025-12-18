package sdk

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/core"
	engine_memory "github.com/duynguyendang/manglekit/internal/engine/memory"
)

// Define constants or config struct
const (
	DefaultMaxSteps   = 10
	DefaultMaxRetries = 3
	BackoffBase       = 100 * time.Millisecond
)

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
	if c.logger != nil {
		ctx = core.ContextWithLogger(ctx, c.logger)
	}

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

	currentAction := startAction
	currentPayload := payload

	for step := 0; step < DefaultMaxSteps; step++ {
		// Check context before starting new step
		if err := ctx.Err(); err != nil {
			return core.Envelope{}, err
		}

		if c.logger != nil {
			c.logger.Info("RunLoop step", "step", step, "action", currentAction)
		}

		result, err := c.ExecuteSingleStep(ctx, currentAction, currentPayload, &params)
		if err != nil {
			return core.Envelope{}, err
		}

		decision := result.Metadata[core.KeyDecision]
		if decision == core.DecisionRoute {
			// Update flow for next loop
			next, ok := result.Metadata[core.KeyNextStep].(string)
			if !ok || next == "" {
				return core.Envelope{}, fmt.Errorf("route decision missing next_step")
			}

			// Validate/Log Payload Handover
			if c.logger != nil {
				c.logger.Info("RunLoop: Routing to next action", "from", currentAction, "to", next, "payload_type", fmt.Sprintf("%T", result.Payload))
			}

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
// It handles: Action Execution, History Persistence, Blueprint Alignment Backoff, and Steering Logic (Retry/Route updates).
func (c *Client) ExecuteSingleStep(ctx context.Context, actionName string, payload any, params *ExecutionParams) (core.Envelope, error) {
	// 1. Resolve Action
	action, ok := c.registry[actionName]
	if !ok {
		return core.Envelope{}, fmt.Errorf("action not found: %s", actionName)
	}

	env := core.NewEnvelope(payload)
	env.ContentType = action.Metadata().InputContentType

	// --- Phase 1: Context Injection ---
	// 1.1 Inject Feedback (History & Last Hint)
	if len(params.FeedbackHistory) > 0 {
		env.Metadata[core.KeyPrevFeedback] = strings.Join(params.FeedbackHistory, "; ")
	}
	if params.LastFeedback != "" {
		env.SetFeedback(params.LastFeedback)
		env.Metadata["mangle_feedback"] = params.LastFeedback
	}

	// 1.2 Inject Chat History
	if len(params.CurrentHistory) > 0 && params.MemoryMode != core.MemoryModeNone {
		env.SetHistory(params.CurrentHistory)
	}

	// 1.3 Inject Semantic Memory (RAG)
	c.recallContext(ctx, payload, &env)

	// 1.3 Inject Explicit Metadata
	if params.Metadata != nil {
		for k, v := range params.Metadata {
			env.Metadata[k] = v
		}
	}

	// 1.4 Inject Context Facts (e.g. User Role, Budget from sdk.WithFact)
	// This ensures facts propagate to the Engine for Blueprint Checks.
	if facts := core.ContextFacts(ctx); facts != nil {
		for k, v := range facts {
			env.Metadata[k] = v
		}
	}

	// --- Phase 2: Blueprint Check (Pre-check) & Phase 3: Execution (Intuition) ---
	// The Action.Execute wrapper handles the Pre-check (Blueprint) and the actual Genkit Execution.
	result, err := action.Execute(ctx, env)
	if err != nil {
		var pve *core.AlignmentError
		if errors.As(err, &pve) {
			if params.RetryCount >= DefaultMaxRetries {
				return core.Envelope{}, fmt.Errorf("max retries exceeded: %w", err)
			}
			params.RetryCount++
			params.LastFeedback = pve.Message

			if c.logger != nil {
				c.logger.Warn("RunLoop: Blueprint Alignment Issue", "feedback", params.LastFeedback, "attempt", params.RetryCount)
			}

			// Context-aware Backoff
			if err := c.backoff(ctx, params.RetryCount); err != nil {
				return core.Envelope{}, err
			}

			// Return a mock result to signal RETRY to caller
			res := core.NewEnvelope(payload)
			res.Metadata[core.KeyDecision] = core.DecisionRetry
			return res, nil
		}
		return core.Envelope{}, err
	}

	params.LastFeedback = ""

	// 4. Update History
	if params.MemoryMode != core.MemoryModeNone {
		userContent := safelyStringify(payload)
		assistContent := safelyStringify(result.Payload)

		newExchange := []core.Message{
			{Role: "user", Content: userContent},
			{Role: "assistant", Content: assistContent},
		}
		params.CurrentHistory = append(params.CurrentHistory, newExchange...)
	}

	// 5. Persist
	if params.Store != nil && params.SessionID != "" && params.MemoryMode == core.MemoryModePersist {
		if err := params.Store.Append(ctx, params.SessionID, params.CurrentHistory); err != nil && c.logger != nil {
			c.logger.Warn("RunLoop failed to persist history", "error", err)
		}
	}

	// --- Phase 4: Evaluation & Correction (Post-check) ---
	// The Engine evaluates the result against the Blueprint and decides next steps (Retry/Route/Allow).
	decision := result.Metadata[core.KeyDecision]

	// --- HOOK: MEMORIZE ---
	// Only learn if no error and Decision is PROCEED (or empty/legacy)
	if err == nil && (decision == "" || decision == core.DecisionProceed) {
		c.asyncMemorize(payload, result.Payload)
	}

	if c.logger != nil {
		c.logger.Debug("RunLoop decision", "decision", decision, "action", actionName)
	}

	switch decision {
	case core.DecisionRetry:
		if params.RetryCount >= DefaultMaxRetries {
			return core.Envelope{}, fmt.Errorf("max retries exceeded for action %s", actionName)
		}
		params.RetryCount++
		hint := result.GetFeedback()
		params.LastFeedback = hint
		params.FeedbackHistory = append(params.FeedbackHistory, hint)

		if c.logger != nil {
			c.logger.Warn("RunLoop: RETRY triggered", "feedback", hint)
		}

		// Context-aware Backoff
		if err := c.backoff(ctx, params.RetryCount); err != nil {
			return core.Envelope{}, err
		}
		// Return result so caller loops
		return result, nil

	case core.DecisionRoute:
		// Reset retry count for new action
		params.RetryCount = 0
		params.FeedbackHistory = nil

		if c.logger != nil {
			c.logger.Info("RunLoop: Feedback history cleared for new action route")
		}
		// Return result so caller loops
		return result, nil

	case core.DecisionProceed, "":
		return result, nil

	case core.DecisionHalt:
		reason := result.Metadata["reason"]
		if reason == "" {
			reason = result.Metadata["violation_msg"]
		}
		if reason == "" {
			reason = "blueprint violation"
		}
		return core.Envelope{}, fmt.Errorf("action halted by blueprint: %s", reason)
	}

	// Should not reach here for standard decisions
	return result, nil
}

// backoff handles the sleep and context cancellation check
func (c *Client) backoff(ctx context.Context, retryCount int) error {
	sleepDuration := time.Duration(retryCount) * BackoffBase
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(sleepDuration):
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
			if c.logger != nil {
				c.logger.Warn("Engine check failed, skipping memory", "err", err)
			}
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
	contextData, err := c.agentMemory.Recall(ctx, inputStr)
	if err != nil {
		if c.logger != nil {
			c.logger.Warn("Memory Recall failed", "error", err)
		}
		if span != nil {
			span.RecordError(err)
		}
		return
	}

	// Inject if found
	if contextData != "" {
		env.SetMeta(core.KeyContext, contextData)
		if c.logger != nil {
			c.logger.Debug("Injected memory context", "len", len(contextData))
		}
	}
}

// asyncMemorize handles the Fire-and-Forget storage logic.
func (c *Client) asyncMemorize(input any, output any) {
	if c.agentMemory == nil {
		return
	}

	inputStr := safelyStringify(input)
	outputStr := safelyStringify(output)

	// Launch Goroutine to not block response latency
	go func(q, a string) {
		// Create a detached context with timeout to prevent leaks
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := c.agentMemory.Memorize(ctx, q, a); err != nil {
			if c.logger != nil {
				c.logger.Warn("Memory Memorize failed", "error", err)
			}
		}
	}(inputStr, outputStr)
}
