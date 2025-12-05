package manglekit

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// --- Aliases ---
type Client = sdk.Client
type ClientOption = sdk.ClientOption
type ExecuteOption = sdk.ExecuteOption

// --- Facade Functions ---

// NewClient initializes the client with a default Slog logger.
// It implements the "Batteries Included" philosophy by injecting a default logger.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	// Inject default logger first (can be overridden by opts)
	defaultOpts := []ClientOption{
		sdk.WithLogger(getDefaultLogger()),
	}
	finalOpts := append(defaultOpts, opts...)
	return sdk.NewClient(ctx, finalOpts...)
}

// Must helper for panic-on-error initialization
func Must(c *Client, err error) *Client {
	return sdk.Must(c, err)
}

// Define is the public entry point for creating Actions
func Define[In any, Out any](
	c *Client,
	name string,
	handler func(context.Context, In) (Out, error),
) *sdk.Runnable[In, Out] {
	return sdk.Define(c, name, handler)
}

// --- Option Wrappers ---
func WithPolicyPath(path string) ClientOption { return sdk.WithPolicyPath(path) }
func WithFailMode(mode string) ClientOption { return sdk.WithFailMode(mode) }
func WithLogger(l core.Logger) ClientOption { return sdk.WithLogger(l) }
func WithMemory(store core.MemoryStore) ClientOption { return sdk.WithMemory(store) }

func WithSessionID(id string) ExecuteOption { return sdk.WithSessionID(id) }
func WithTransientMemory() ExecuteOption { return sdk.WithTransientMemory() }
func WithMetadata(key, value string) ExecuteOption { return sdk.WithMetadata(key, value) }
