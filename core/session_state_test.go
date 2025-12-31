package core

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestSessionState_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		state   *SessionState
		wantErr bool
	}{
		{
			name: "valid state with struct payload",
			state: &SessionState{
				SessionID: "test-session-1",
				ActiveEnvelope: Envelope{
					ID:          uuid.New(),
					Payload:     map[string]string{"key": "value"},
					ContentType: TypeJSON,
					Metadata:    map[string]any{"test": "meta"},
				},
				ExecutionCtx: ExecutionContext{
					RetryCount:      1,
					FeedbackHistory: []string{"feedback1"},
					CurrentHistory:  []Message{{Role: "user", Content: "hello"}},
				},
				LogicalFacts: []string{"fact(a, b)."},
			},
			wantErr: false,
		},
		{
			name: "empty state",
			state: &SessionState{
				SessionID: "empty-session",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(data) == 0 {
				t.Error("MarshalJSON() returned empty data")
			}
		})
	}
}

func TestSessionState_UnmarshalJSON(t *testing.T) {
	original := &SessionState{
		SessionID: "test-session-2",
		ActiveEnvelope: Envelope{
			ID:          uuid.New(),
			Payload:     "test payload",
			ContentType: TypeStruct,
			Metadata:    map[string]any{"key": "value"},
		},
		ExecutionCtx: ExecutionContext{
			RetryCount:      2,
			FeedbackHistory: []string{"fb1", "fb2"},
		},
		LogicalFacts: []string{"fact1.", "fact2."},
	}

	// Marshal
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Unmarshal
	var restored SessionState
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify
	if restored.SessionID != original.SessionID {
		t.Errorf("SessionID mismatch: got %v, want %v", restored.SessionID, original.SessionID)
	}
	if restored.ExecutionCtx.RetryCount != original.ExecutionCtx.RetryCount {
		t.Errorf("RetryCount mismatch: got %v, want %v", restored.ExecutionCtx.RetryCount, original.ExecutionCtx.RetryCount)
	}
	if len(restored.LogicalFacts) != len(original.LogicalFacts) {
		t.Errorf("LogicalFacts length mismatch: got %v, want %v", len(restored.LogicalFacts), len(original.LogicalFacts))
	}
}

func TestSessionState_Clone(t *testing.T) {
	original := &SessionState{
		SessionID: "clone-test",
		ActiveEnvelope: Envelope{
			ID:             uuid.New(),
			Payload:        "test",
			Metadata:       map[string]any{"key": "value"},
			SecurityLabels: []string{"label1"},
			Facts:          []string{"fact1"},
		},
		ExecutionCtx: ExecutionContext{
			RetryCount:      1,
			FeedbackHistory: []string{"feedback"},
			CurrentHistory:  []Message{{Role: "user", Content: "test"}},
		},
		LogicalFacts: []string{"fact1", "fact2"},
	}

	cloned := original.Clone()

	// Verify deep copy
	if cloned.SessionID != original.SessionID {
		t.Error("SessionID not cloned correctly")
	}

	// Modify clone and ensure original is unchanged
	cloned.ExecutionCtx.RetryCount = 99
	if original.ExecutionCtx.RetryCount == 99 {
		t.Error("Clone is not independent - RetryCount modified in original")
	}

	cloned.LogicalFacts = append(cloned.LogicalFacts, "fact3")
	if len(original.LogicalFacts) != 2 {
		t.Error("Clone is not independent - LogicalFacts modified in original")
	}
}

func TestSessionState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		state   *SessionState
		wantErr bool
	}{
		{
			name: "valid state",
			state: &SessionState{
				SessionID: "valid",
				ActiveEnvelope: Envelope{
					ID: uuid.New(),
				},
			},
			wantErr: false,
		},
		{
			name: "missing session ID",
			state: &SessionState{
				ActiveEnvelope: Envelope{
					ID: uuid.New(),
				},
			},
			wantErr: true,
		},
		{
			name: "invalid envelope ID",
			state: &SessionState{
				SessionID: "test",
				ActiveEnvelope: Envelope{
					ID: uuid.UUID{}, // Zero UUID
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.state.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
