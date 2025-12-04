package manglekit

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"go.opentelemetry.io/otel/trace"
)

// Client represents the initialized Manglekit system.
type Client = sdk.Client

// ClientOption configures the Client.
type ClientOption = sdk.ClientOption

// ExecuteOption configures the execution of an Action.
type ExecuteOption = sdk.ExecuteOption

// NewClient creates a new Manglekit Client.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	// 1. Inject Defaults (Logger)
	defaultOpts := []ClientOption{
		WithLogger(getDefaultLogger()),
	}

	// 2. Merge User Options (User overrides default if duplicate keys exist)
	finalOpts := append(defaultOpts, opts...)

	return sdk.NewClient(ctx, finalOpts...)
}

// NewDefault creates a Client with default settings.
func NewDefault() (*Client, error) {
	return sdk.NewDefault()
}

// WithPolicyPath loads Datalog rules from a file path.
func WithPolicyPath(path string) ClientOption {
	return sdk.WithPolicyPath(path)
}

// WithFailMode sets the failure strategy ("open" or "closed").
func WithFailMode(mode string) ClientOption {
	return sdk.WithFailMode(mode)
}

// WithLogger sets a custom logger.
func WithLogger(l core.Logger) ClientOption {
	return sdk.WithLogger(l)
}

// WithTracerProvider sets the OTel provider.
func WithTracerProvider(tp trace.TracerProvider) ClientOption {
	return sdk.WithTracerProvider(tp)
}

// WithMemory sets a custom memory store.
func WithMemory(store core.MemoryStore) ClientOption {
	return sdk.WithMemory(store)
}

// WithSessionID activates Stateful mode (Persistent).
func WithSessionID(id string) ExecuteOption {
	return sdk.WithSessionID(id)
}

// WithTransientMemory activates In-Memory mode without persistence.
func WithTransientMemory() ExecuteOption {
	return sdk.WithTransientMemory()
}

// WithMetadata injects custom context (e.g., source, user_tier).
func WithMetadata(key, value string) ExecuteOption {
	return sdk.WithMetadata(key, value)
}
