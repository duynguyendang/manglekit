package sdk

import (
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/telemetry"
)

// ClientOption configures the Client.
type ClientOption func(*Client)

// WithPolicyPath loads Datalog rules from a file path.
func WithPolicyPath(path string) ClientOption {
	return func(c *Client) {
		c.initialPolicyPath = path
	}
}

// WithFailMode sets the failure strategy ("open" or "closed").
func WithFailMode(mode string) ClientOption {
	return func(c *Client) {
		c.failureMode = mode
	}
}

// WithLogger sets a custom logger.
func WithLogger(l core.Logger) ClientOption {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithTracerProvider sets the OTel provider.
func WithTracerProvider(tp trace.TracerProvider) ClientOption {
	return func(c *Client) {
		if tp != nil {
			otelTracer := tp.Tracer(TracerName)
			c.otelTracer = otelTracer
			c.tracer = telemetry.NewOTelTracer(otelTracer)
		}
	}
}

// WithMemory sets a custom memory store.
func WithMemory(store core.MemoryStore) ClientOption {
	return func(c *Client) {
		if store != nil {
			c.memory = store
		}
	}
}

// ExecutionParams holds the runtime configuration.
type ExecutionParams struct {
	SessionID  string
	MemoryMode core.MemoryMode
	Metadata   map[string]string
	Timeout    time.Duration
}

type ExecuteOption func(*ExecutionParams)

// WithSessionID activates Stateful mode (Persistent).
func WithSessionID(id string) ExecuteOption {
	return func(p *ExecutionParams) {
		p.SessionID = id
		p.MemoryMode = core.MemoryModePersist
	}
}

// WithTransientMemory activates In-Memory mode without persistence.
func WithTransientMemory() ExecuteOption {
	return func(p *ExecutionParams) {
		p.MemoryMode = core.MemoryModeTransient
	}
}

// WithMetadata injects custom context (e.g., source, user_tier).
func WithMetadata(key, value string) ExecuteOption {
	return func(p *ExecutionParams) {
		if p.Metadata == nil {
			p.Metadata = make(map[string]string)
		}
		p.Metadata[key] = value
	}
}
