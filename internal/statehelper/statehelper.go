package statehelper

import (
	"context"
	"encoding/json"

	"github.com/duynguyendang/manglekit-wip/core"
)

// ConversationManager provides helper methods for managing conversational state.
type ConversationManager struct{}

// NewConversationManager creates a new ConversationManager.
func NewConversationManager() *ConversationManager {
	return &ConversationManager{}
}

// LoadHistory loads and deserializes the conversation history for a given sessionID.
func (cm *ConversationManager) LoadHistory(ctx context.Context, sessionID string, sp core.StateProvider, logger core.Logger) *core.ConversationHistory {
	history := &core.ConversationHistory{}
	if sp != nil && sessionID != "" {
		rawState, err := sp.Get(ctx, sessionID)
		if err != nil {
			logger.Warn("Failed to retrieve state for session", "sessionID", sessionID, "error", err)
			// Do not fail the request, just proceed without history.
		}

		if rawState != nil {
			// Use a type assertion or a helper function to convert the state.
			// For simplicity, we assume the state is stored as a JSON string.
			if stateBytes, ok := rawState.([]byte); ok {
				if err := json.Unmarshal(stateBytes, history); err != nil {
					logger.Warn("Failed to unmarshal state for session", "sessionID", sessionID, "error", err)
				}
			}
		}
	}
	return history
}

// UpdateAndSaveHistory updates the history with the current query and answer,
// then serializes and saves it back to the state provider.
func (cm *ConversationManager) UpdateAndSaveHistory(ctx context.Context, sessionID string, sp core.StateProvider, logger core.Logger, history *core.ConversationHistory, q core.Query, a core.Answer) {
	if sp != nil && sessionID != "" && history != nil {
		// Append the user's query and the model's answer to the history.
		history.Messages = append(history.Messages, core.Message{Role: "user", Content: q.Text})
		history.Messages = append(history.Messages, core.Message{Role: "model", Content: a.Text})

		// Marshal the updated history to JSON before saving.
		updatedStateBytes, err := json.Marshal(history)
		if err != nil {
			logger.Warn("Failed to marshal updated state for session", "sessionID", sessionID, "error", err)
		} else {
			if err := sp.Set(ctx, sessionID, updatedStateBytes); err != nil {
				logger.Warn("Failed to save state for session", "sessionID", sessionID, "error", err)
			}
		}
	}
}
