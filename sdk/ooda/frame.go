package ooda

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

// Memory defines the interface for storing and retrieving context.
// It represents the "Memory" component of the OODA loop.
type Memory interface {
	// Recall retrieves relevant context from memory based on the input.
	// This is called during the Orient phase to hydrate the frame with facts.
	Recall(ctx context.Context, input string) ([]Atom, error)

	// Commit stores the results of an action back into memory.
	// This is called after the Act phase to persist the outcome.
	Commit(ctx context.Context, frame *CognitiveFrame) error

	// Store stores a single atom into memory.
	Store(ctx context.Context, atom Atom) error

	// Query retrieves atoms matching a predicate pattern.
	Query(ctx context.Context, predicate string) ([]Atom, error)
}

// Brain defines the interface for policy evaluation and decision making.
// It wraps the Manglekit PolicyEngine.
type Brain interface {
	// Evaluate makes a decision based on the input and context.
	// Returns a Decision with AuditTrail explaining which rules were matched.
	Evaluate(ctx context.Context, frame *CognitiveFrame) (*core.Decision, error)

	// Verify validates an action result against policy requirements.
	Verify(ctx context.Context, frame *CognitiveFrame) (*core.AuditTrail, error)

	// LoadPolicy loads additional policy rules.
	LoadPolicy(ctx context.Context, rules string) error
}

// Executor defines the interface for executing actions.
// It handles the actual work of the chosen action.
type Executor interface {
	// Execute performs the action selected by the Brain.
	// Returns the result of the execution.
	Execute(ctx context.Context, frame *CognitiveFrame, decision *core.Decision) (any, error)

	// Rollback reverses a previous execution if needed.
	Rollback(ctx context.Context, frame *CognitiveFrame, result any) error
}

// Run executes the full OODA loop:
// 1. Observe - Capture raw input
// 2. Orient - Hydrate context from Memory
// 3. Decide - Evaluate with Brain (returns Decision + AuditTrail)
// 4. Act - Execute the action
// 5. Verify - Post-act validation
func Run(ctx context.Context, frame *CognitiveFrame) (*CognitiveFrame, error) {
	ctx, cancel := context.WithTimeout(ctx, frame.Timeout)
	defer cancel()

	// Initialize phase durations
	if frame.PhaseDurations == nil {
		frame.PhaseDurations = make(map[Phase]time.Duration)
	}

	// Phase 1: Observe
	if err := observe(ctx, frame); err != nil {
		return frame, fmt.Errorf("observe phase failed: %w", err)
	}

	// Phase 2: Orient
	if err := orient(ctx, frame); err != nil {
		return frame, fmt.Errorf("orient phase failed: %w", err)
	}

	// Phase 3: Decide
	if err := decide(ctx, frame); err != nil {
		return frame, fmt.Errorf("decide phase failed: %w", err)
	}

	// Phase 4: Act (with retry)
	if err := actWithRetry(ctx, frame); err != nil {
		return frame, fmt.Errorf("act phase failed after %d retries: %w", frame.RetryCount, err)
	}

	// Phase 5: Verify
	if err := verify(ctx, frame); err != nil {
		return frame, fmt.Errorf("verify phase failed: %w", err)
	}

	// Commit to memory
	if frame.Memory != nil {
		if err := frame.Memory.Commit(ctx, frame); err != nil {
			return frame, fmt.Errorf("failed to commit to memory: %w", err)
		}
	}

	return frame, nil
}

// observe captures raw input and environment state
func observe(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseObserve

	// Default observation: just capture input as-is
	// Apps can customize this by implementing custom logic
	if frame.Input != "" {
		frame.Context = append(frame.Context, Atom{
			Predicate: "raw_input",
			Subject:   frame.ID.String(),
			Object:    frame.Input,
			Weight:    1.0,
		})
	}

	frame.PhaseDurations[PhaseObserve] = time.Since(start)
	return nil
}

// orient retrieves relevant context from Memory (MEB) and Session Store
func orient(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseOrient

	// === Dual Memory Architecture ===
	// 1. Query KnowledgeStore (MEB) for long-term knowledge (playbooks, rules)
	if frame.KnowledgeStore != nil {
		graphID := "default"
		if frame.WorkflowID != "" {
			graphID = frame.WorkflowID // Use workflow as graph scope
		}

		// Recall top-K facts from MEB based on input
		coreAtoms, err := frame.KnowledgeStore.Recall(ctx, frame.Input, 10, graphID)
		if err != nil {
			return fmt.Errorf("failed to recall from knowledge store: %w", err)
		}
		// Convert core.Atom to ooda.Atom
		for _, ca := range coreAtoms {
			frame.Context = append(frame.Context, Atom{
				Subject:   ca.Subject,
				Predicate: ca.Predicate,
				Object:    ca.Object,
				Weight:    ca.Weight,
			})
		}
	}

	// 2. Query TransientStore (Session) for short-term coordination facts (current_node, agent_status)
	if frame.TransientStore != nil && frame.SessionID != "" {
		coreAtoms, err := frame.TransientStore.ToAtoms(ctx, frame.SessionID)
		if err != nil {
			return fmt.Errorf("failed to recall from transient store: %w", err)
		}
		// Convert core.Atom to ooda.Atom
		for _, ca := range coreAtoms {
			frame.Context = append(frame.Context, Atom{
				Subject:   ca.Subject,
				Predicate: ca.Predicate,
				Object:    ca.Object,
			})
		}
	}

	// Legacy: Fallback to Memory interface if available
	if frame.Memory != nil {
		atoms, err := frame.Memory.Recall(ctx, frame.Input)
		if err != nil {
			return fmt.Errorf("failed to recall from memory: %w", err)
		}
		frame.Context = append(frame.Context, atoms...)
	}

	frame.PhaseDurations[PhaseOrient] = time.Since(start)
	return nil
}

// decide evaluates the context using Brain (Manglekit)
func decide(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseDecide

	if frame.Brain == nil {
		return nil // No brain, skip decision
	}

	// Evaluate with Brain - returns Decision with AuditTrail
	decision, err := frame.Brain.Evaluate(ctx, frame)
	if err != nil {
		return fmt.Errorf("brain evaluation failed: %w", err)
	}

	frame.Decision = decision

	// Extract audit trail if available
	if decision != nil {
		frame.AuditTrail = decision.AuditTrail
	}

	frame.PhaseDurations[PhaseDecide] = time.Since(start)
	return nil
}

// act executes the chosen action
func act(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseAct

	if frame.Decision == nil {
		return nil // No decision, nothing to execute
	}

	// Try Dispatcher first (new way), then fall back to Executor (legacy)
	result, err := executeAction(ctx, frame)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	frame.ActionResult = result

	// Commit result to TransientStore (Session Memory) if available
	if frame.TransientStore != nil && frame.SessionID != "" && result != nil {
		sessionID := frame.SessionID
		if frame.Decision.Action != nil && frame.Decision.Action.SessionID != "" {
			sessionID = frame.Decision.Action.SessionID
		}

		// Store the action result in transient store
		resultKey := "action_result_" + frame.Decision.Action.Name
		resultStr := fmt.Sprintf("%v", result)
		err := frame.TransientStore.Put(ctx, sessionID, resultKey, &ports.TransientFact{
			Subject:   "action",
			Predicate: frame.Decision.Action.Name,
			Object:    resultStr,
			Graph:     "session",
		})
		if err != nil {
			// Log but don't fail the action
			fmt.Printf("Warning: failed to commit action result to transient store: %v\n", err)
		}
	}

	frame.PhaseDurations[PhaseAct] = time.Since(start)
	return nil
}

// executeAction attempts to execute the action using Dispatcher or Executor.
func executeAction(ctx context.Context, frame *CognitiveFrame) (any, error) {
	decision := frame.Decision

	// Priority 1: Use Dispatcher if available and Decision has Action
	if frame.Dispatcher != nil && decision.Action != nil {
		return executeWithDispatcher(ctx, frame, decision)
	}

	// Priority 2: Use legacy Executor if available
	if frame.Executor != nil {
		return frame.Executor.Execute(ctx, frame, decision)
	}

	// Priority 3: If Decision has Action but no Dispatcher, try to dispatch
	if decision.Action != nil {
		return nil, fmt.Errorf("dispatcher not configured but action '%s' specified in decision", decision.Action.Name)
	}

	// No execution path available
	return nil, nil
}

// executeWithDispatcher uses the action registry to execute the action.
func executeWithDispatcher(ctx context.Context, frame *CognitiveFrame, decision *core.Decision) (any, error) {
	action := decision.Action

	// Execute via dispatcher
	result, err := frame.Dispatcher.Dispatch(ctx, action.Name, action.Arguments)
	if err != nil {
		// Log the sovereign violation
		fmt.Printf("SOVEREIGN VIOLATION: %v\n", err)

		// Add violation to audit trail if available
		if decision.AuditTrail != nil {
			decision.AuditTrail.MatchedRules = append(decision.AuditTrail.MatchedRules, core.RuleInference{
				RuleName:   "SOVEREIGN_VIOLATION",
				Tier:       core.TierUnknown,
				Definition: fmt.Sprintf("Unknown action: %s", action.Name),
				Predicate:  "dispatch",
			})
		}

		return nil, err
	}

	return result, nil
}

// actWithRetry executes the action with retry logic
func actWithRetry(ctx context.Context, frame *CognitiveFrame) error {
	var lastErr error

	for attempt := 0; attempt <= frame.MaxRetries; attempt++ {
		frame.RetryCount = attempt

		if attempt > 0 {
			timer := time.NewTimer(time.Duration(attempt) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
				timer.Stop()
			}
		}

		err := act(ctx, frame)
		if err == nil {
			return nil
		}

		lastErr = err

		if frame.Executor != nil && frame.ActionResult != nil {
			rollbackErr := frame.Executor.Rollback(ctx, frame, frame.ActionResult)
			if rollbackErr != nil {
				fmt.Printf("Warning: rollback failed: %v\n", rollbackErr)
			}
		}
	}

	return lastErr
}

// verify performs post-act validation by running a mini Decide phase
func verify(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseVerify

	// If we have an action result, verify it against the TransientStore
	if frame.TransientStore != nil && frame.SessionID != "" && frame.ActionResult != nil && frame.Decision != nil && frame.Decision.Action != nil {
		actionName := frame.Decision.Action.Name
		resultStr := fmt.Sprintf("%v", frame.ActionResult)

		// Check if the action result is considered successful via Datalog
		// Query pattern: is_action_successful(ActionName, Result)
		// For now, we check if result contains "error" or "failed" (simple heuristic)
		verifyStatus := "success"
		if strings.Contains(strings.ToLower(resultStr), "error") || strings.Contains(strings.ToLower(resultStr), "failed") {
			verifyStatus = "failed"
		}

		// Store verification status in TransientStore
		err := frame.TransientStore.Put(ctx, frame.SessionID, "verify_status_"+actionName, &ports.TransientFact{
			Subject:   "verify",
			Predicate: actionName,
			Object:    verifyStatus,
			Graph:     "session",
		})
		if err != nil {
			fmt.Printf("Warning: failed to store verify status: %v\n", err)
		}

		// If verification failed, flag for retry
		if verifyStatus == "failed" {
			// Store retry flag
			frame.TransientStore.Put(ctx, frame.SessionID, "needs_retry", &ports.TransientFact{
				Subject:   "workflow",
				Predicate: "retry",
				Object:    "true",
				Graph:     "session",
			})
		}
	}

	// Run Brain.Verify if available for policy-based verification
	if frame.Brain != nil {
		// Verify the action result against policy
		auditTrail, err := frame.Brain.Verify(ctx, frame)
		if err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}

		// Store verification result
		frame.AuditTrail = auditTrail
	}

	frame.PhaseDurations[PhaseVerify] = time.Since(start)
	return nil
}

// GetAuditSummary returns a human-readable summary of the audit trail
func (f *CognitiveFrame) GetAuditSummary() string {
	if f.AuditTrail == nil {
		return "No audit trail available"
	}
	return f.AuditTrail.Summary()
}

// GetPhaseDurations returns a map of phase to duration
func (f *CognitiveFrame) GetPhaseDurations() map[Phase]time.Duration {
	return f.PhaseDurations
}

// TotalDuration returns the total execution time
func (f *CognitiveFrame) TotalDuration() time.Duration {
	var total time.Duration
	for _, d := range f.PhaseDurations {
		total += d
	}
	return total
}
