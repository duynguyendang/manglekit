package manglekit

import (
	"context"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/duynguyendang/manglekit-wip/sdk"
)

// --- Aliases ---
type Client = sdk.Client
type ClientOption = sdk.ClientOption
type ExecuteOption = sdk.ExecuteOption

// --- Facade Functions ---

// NewClient initializes the client with defaults.
// It implements the "Batteries Included" philosophy by leveraging SDK defaults.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	return sdk.NewClient(ctx, opts...)
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
func WithBlueprintPath(path string) ClientOption { return sdk.WithBlueprintPath(path) }

// Deprecated: Use WithBlueprintPath instead.
func WithPolicyPath(path string) ClientOption         { return sdk.WithBlueprintPath(path) }
func WithFailMode(mode string) ClientOption           { return sdk.WithFailMode(mode) }
func WithLogger(l core.Logger) ClientOption           { return sdk.WithLogger(l) }
func WithHistory(store core.HistoryStore) ClientOption { return sdk.WithHistory(store) }
func WithMemory(mem core.AgentMemory) ClientOption    { return sdk.WithMemory(mem) }

func WithSessionID(id string) ExecuteOption        { return sdk.WithSessionID(id) }
func WithTransientMemory() ExecuteOption           { return sdk.WithTransientMemory() }
func WithMetadata(key, value string) ExecuteOption { return sdk.WithMetadata(key, value) }
