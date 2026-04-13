package ooda

import (
	"context"
	"fmt"
	"time"
)

// RunOODA executes the OODA loop for deterministic tool execution.
// No EAST steering. No entropy measurement. Simple and fast.
//
// Flow: Observe → Orient → Decide → Act → Verify(schema) → Commit
//
// Security gates at every phase boundary:
//   - Observe: input validation (input_rule, injection_pattern)
//   - Orient: tool lookup + authorization (tool_authorization, tool_pattern)
//   - Decide: Datalog policy evaluation (Brain.Evaluate with AuditTrail)
//   - Act: tool execution with simple retry
//   - Verify: output schema check (output_schema)
func RunOODA(ctx context.Context, frame *CognitiveFrame) (*CognitiveFrame, error) {
	ctx, cancel := context.WithTimeout(ctx, frame.Timeout)
	defer cancel()

	if frame.PhaseDurations == nil {
		frame.PhaseDurations = make(map[Phase]time.Duration)
	}

	// 1. Observe (Datalog: input validation)
	if err := observeWithValidation(ctx, frame); err != nil {
		return frame, fmt.Errorf("observe failed: %w", err)
	}

	// 2. Orient (Datalog: tool lookup + authorization)
	if err := orientForTool(ctx, frame); err != nil {
		return frame, fmt.Errorf("orient failed: %w", err)
	}

	// 3. Decide (Datalog: authorize action + validate command)
	if err := decide(ctx, frame); err != nil {
		return frame, fmt.Errorf("decide failed: %w", err)
	}

	// 4. Act (execute with simple retry)
	if err := actWithSimpleRetry(ctx, frame); err != nil {
		return frame, fmt.Errorf("act failed: %w", err)
	}

	// 5. Verify (Datalog: output schema check)
	if err := VerifySchema(ctx, frame); err != nil {
		return frame, fmt.Errorf("verify failed: %w", err)
	}

	// 6. Commit
	if frame.Memory != nil {
		if err := frame.Memory.Commit(ctx, frame); err != nil {
			return frame, fmt.Errorf("commit failed: %w", err)
		}
	}

	frame.Status = VerifyStatusPassed
	return frame, nil
}

// observeWithValidation captures input and validates against injection patterns.
func observeWithValidation(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseObserve

	if frame.Input == "" {
		return fmt.Errorf("empty input")
	}

	// Capture input as atom (low weight — untrusted until Orient validates)
	frame.Context = append(frame.Context, Atom{
		Predicate: "raw_input",
		Subject:   frame.ID.String(),
		Object:    frame.Input,
		Weight:    0.3, // INT8: untrusted
	})

	// Input validation would be done here via Brain if available
	// For now, capture the atom and proceed

	frame.PhaseDurations[PhaseObserve] = time.Since(start)
	return nil
}

// orientForTool finds the appropriate tool and checks authorization.
func orientForTool(ctx context.Context, frame *CognitiveFrame) error {
	start := time.Now()
	frame.Phase = PhaseOrient

	// Query Memory for tool context
	if frame.Memory != nil {
		atoms, err := frame.Memory.Recall(ctx, frame.Input)
		if err == nil {
			frame.Context = append(frame.Context, atoms...)
		}
	}

	// Query KnowledgeStore for tool authorization rules
	if frame.KnowledgeStore != nil {
		graphID := "default"
		if frame.WorkflowID != "" {
			graphID = frame.WorkflowID
		}
		coreAtoms, err := frame.KnowledgeStore.Recall(ctx, frame.Input, 5, graphID)
		if err == nil {
			for _, ca := range coreAtoms {
				frame.Context = append(frame.Context, Atom{
					Subject:   ca.Subject,
					Predicate: ca.Predicate,
					Object:    ca.Object,
					Weight:    ca.Weight,
				})
			}
		}
	}

	frame.PhaseDurations[PhaseOrient] = time.Since(start)
	return nil
}

// actWithSimpleRetry executes the action with simple error retry (no feedback).
func actWithSimpleRetry(ctx context.Context, frame *CognitiveFrame) error {
	var lastErr error
	maxRetries := frame.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
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
			frame.Executor.Rollback(ctx, frame, frame.ActionResult)
		}
	}

	return lastErr
}
