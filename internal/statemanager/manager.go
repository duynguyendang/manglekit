package statemanager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// DurableStateManager orchestrates state persistence and recovery for Manglekit sessions.
// It wraps the core.StateProvider interface and adds semantic recovery capabilities.
type DurableStateManager struct {
	provider core.StateProvider
	engine   core.Evaluator
	logger   core.Logger
}

// New creates a new DurableStateManager.
func New(provider core.StateProvider, engine core.Evaluator, logger core.Logger) *DurableStateManager {
	if logger == nil {
		logger = core.NopLogger{}
	}

	return &DurableStateManager{
		provider: provider,
		engine:   engine,
		logger:   logger,
	}
}

// Hydrate loads and reconstructs a session state from the underlying provider.
// It performs semantic re-constitution including:
// - Type reconstruction using Envelope.ContentType
// - Engine priming by re-injecting LogicalFacts into the Mangle runtime
// - Feedback alignment by restoring FeedbackHistory
//
// Returns nil, nil if the session does not exist (first run).
func (m *DurableStateManager) Hydrate(ctx context.Context, sessionID string) (*core.SessionState, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id cannot be empty")
	}

	m.logger.Debug("Hydrating session state", "session_id", sessionID)

	// 1. Retrieve raw state from provider
	rawState, err := m.provider.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get state from provider: %w", err)
	}

	// If no state exists, return nil (first run)
	if rawState == nil {
		m.logger.Debug("No existing state found for session", "session_id", sessionID)
		return nil, nil
	}

	// 2. Unmarshal into SessionState
	var state core.SessionState

	// Handle different raw state types
	switch v := rawState.(type) {
	case []byte:
		if err := json.Unmarshal(v, &state); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state from bytes: %w", err)
		}
	case string:
		if err := json.Unmarshal([]byte(v), &state); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state from string: %w", err)
		}
	case *core.SessionState:
		state = *v
	case core.SessionState:
		state = v
	default:
		// Try JSON round-trip for other types
		data, err := json.Marshal(rawState)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw state: %w", err)
		}
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	// 3. Validate state
	if err := state.Validate(); err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	// 4. Semantic Re-constitution

	// 4a. Type Reconstruction
	// The payload is already unmarshaled, but we preserve the ContentType
	// for downstream processing

	// 4b. Engine Priming - Re-inject LogicalFacts
	if m.engine != nil && len(state.LogicalFacts) > 0 {
		m.logger.Debug("Priming engine with logical facts",
			"session_id", sessionID,
			"fact_count", len(state.LogicalFacts))

		if err := m.engine.LoadFacts(ctx, state.LogicalFacts); err != nil {
			// Log warning but don't fail hydration
			m.logger.Warn("Failed to prime engine with facts", "error", err)
		}
	}

	// 4c. Feedback Alignment
	// FeedbackHistory is already restored in the ExecutionCtx
	// The SDK loop will use this to avoid repeating past mistakes

	m.logger.Info("Successfully hydrated session state",
		"session_id", sessionID,
		"retry_count", state.ExecutionCtx.RetryCount,
		"history_length", len(state.ExecutionCtx.CurrentHistory),
		"fact_count", len(state.LogicalFacts))

	return &state, nil
}

// Checkpoint saves the current session state to the underlying provider.
// This should only be called after a successful Reflect phase to prevent
// persisting invalid or "poisoned" states.
func (m *DurableStateManager) Checkpoint(ctx context.Context, state *core.SessionState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	// Validate state before persisting
	if err := state.Validate(); err != nil {
		return fmt.Errorf("invalid state: %w", err)
	}

	m.logger.Debug("Checkpointing session state", "session_id", state.SessionID)

	// Marshal state to JSON
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Persist to provider
	if err := m.provider.Set(ctx, state.SessionID, data); err != nil {
		return fmt.Errorf("failed to save state to provider: %w", err)
	}

	m.logger.Info("Successfully checkpointed session state",
		"session_id", state.SessionID,
		"state_size_bytes", len(data))

	return nil
}

// ExtractFacts extracts Datalog facts from an envelope.
// This is used during checkpoint to capture the logical state.
func (m *DurableStateManager) ExtractFacts(ctx context.Context, envelope core.Envelope) ([]string, error) {
	// Extract facts from the envelope's Facts field
	facts := make([]string, 0, len(envelope.Facts))

	// Copy facts from envelope
	if len(envelope.Facts) > 0 {
		facts = append(facts, envelope.Facts...)
	}

	// TODO: In the future, we could query the engine for all derived facts
	// For now, we rely on the envelope's Facts field which should be populated
	// during the Reflect phase

	return facts, nil
}

// Delete removes the state for a session.
func (m *DurableStateManager) Delete(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session_id cannot be empty")
	}

	m.logger.Debug("Deleting session state", "session_id", sessionID)

	if err := m.provider.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	m.logger.Info("Successfully deleted session state", "session_id", sessionID)
	return nil
}

// Close cleans up resources held by the state manager.
func (m *DurableStateManager) Close(ctx context.Context) error {
	if m.provider != nil {
		return m.provider.Close(ctx)
	}
	return nil
}
