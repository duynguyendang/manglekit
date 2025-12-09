package sdk

import (
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/telemetry"
)

// ClientOption configures the Manglekit Client during initialization.
type ClientOption func(*Client)

// WithPolicyPath specifies the file path to load Datalog rules from.
//
// Parameters:
//   - path: A file path, directory path, or glob pattern containing policy files.
func WithPolicyPath(path string) ClientOption {
	return func(c *Client) {
		c.initialPolicyPath = path
	}
}

// WithFailMode sets the resilience strategy for the client.
//
// Parameters:
//   - mode: "open" (allow execution on error) or "closed" (block execution on error).
func WithFailMode(mode string) ClientOption {
	return func(c *Client) {
		c.failureMode = mode
	}
}

// WithLogger sets a custom logger for the client.
//
// Parameters:
//   - l: A core.Logger implementation.
func WithLogger(l core.Logger) ClientOption {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithTracerProvider configures the OpenTelemetry tracer provider.
// This enables Manglekit to emit spans to your existing tracing infrastructure.
//
// Parameters:
//   - tp: The OpenTelemetry TracerProvider.
func WithTracerProvider(tp trace.TracerProvider) ClientOption {
	return func(c *Client) {
		if tp != nil {
			otelTracer := tp.Tracer(TracerName)
			c.otelTracer = otelTracer
			c.tracer = telemetry.NewOTelTracer(otelTracer)
		}
	}
}

// WithMemory configures a custom persistence store for chat history.
//
// Parameters:
//   - store: A core.MemoryStore implementation (e.g., Redis backed).
func WithMemory(store core.MemoryStore) ClientOption {
	return func(c *Client) {
		if store != nil {
			c.memory = store
		}
	}
}

// WithLLM configures the AI backend for the client.
// This supports the "Explicit AI Adapter" pattern where the application initializes the model
// (e.g., via Genkit) and passes it to the SDK.
//
// Parameters:
//   - gen: A TextGenerator implementation (e.g., adapters.ai.NewGenkitAdapter(model)).
func WithLLM(gen TextGenerator) ClientOption {
	return func(c *Client) {
		if gen != nil {
			c.llm = gen
		}
	}
}

// ExecutionParams holds the configuration for a specific execution run.
type ExecutionParams struct {
	// SessionID is the unique identifier for a conversation/session.
	SessionID string
	// MemoryMode determines how chat history is handled (None, Transient, Persist).
	MemoryMode core.MemoryMode
	// Metadata contains additional context to be injected into the execution envelope.
	Metadata map[string]string
	// Timeout (unused currently) specifies the max duration for the execution.
	Timeout time.Duration

	// State fields (Managed by ExecuteSingleStep/Loop)
	Store           core.MemoryStore   `json:"-"` // Internal store reference
	CurrentHistory  []core.ChatMessage `json:"history,omitempty"`
	FeedbackHistory []string           `json:"feedback_history,omitempty"`
	LastFeedback    string             `json:"last_feedback,omitempty"`
	RetryCount      int                `json:"retry_count,omitempty"`
}

// ExecuteOption configures a single execution call (e.g., ExecuteByName).
type ExecuteOption func(*ExecutionParams)

// WithSessionID activates persistent stateful mode for the execution.
// It links the execution to a specific session history.
//
// Parameters:
//   - id: The session identifier.
func WithSessionID(id string) ExecuteOption {
	return func(p *ExecutionParams) {
		p.SessionID = id
		p.MemoryMode = core.MemoryModePersist
	}
}

// WithTransientMemory activates in-memory stateful mode.
// History is tracked for the duration of the loop/process but not persisted.
func WithTransientMemory() ExecuteOption {
	return func(p *ExecutionParams) {
		p.MemoryMode = core.MemoryModeTransient
	}
}

// WithMetadata injects custom key-value pairs into the execution envelope's metadata.
// This is useful for passing context like "user_id" or "source" to the policy engine.
//
// Parameters:
//   - key: The metadata key.
//   - value: The metadata value.
func WithMetadata(key, value string) ExecuteOption {
	return func(p *ExecutionParams) {
		if p.Metadata == nil {
			p.Metadata = make(map[string]string)
		}
		p.Metadata[key] = value
	}
}
