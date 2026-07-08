package manglekit

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
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
func WithPolicyPath(path string) ClientOption { return sdk.WithBlueprintPath(path) }
func WithFailMode(mode string) ClientOption           { return sdk.WithFailMode(mode) }
func WithLogger(l core.Logger) ClientOption           { return sdk.WithLogger(l) }
func WithHistory(store core.HistoryStore) ClientOption { return sdk.WithHistory(store) }
func WithMemory(mem core.AgentMemory) ClientOption    { return sdk.WithMemory(mem) }

func WithSessionID(id string) ExecuteOption        { return sdk.WithSessionID(id) }
func WithTransientMemory() ExecuteOption           { return sdk.WithTransientMemory() }
func WithMetadata(key, value string) ExecuteOption { return sdk.WithMetadata(key, value) }

// MustReadFile reads the file at path and panics on error. It is a small
// convenience for example/demo setup code where a missing fixture is fatal.
func MustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return data
}

// MustNewClient creates a client and panics if initialization fails.
func MustNewClient(ctx context.Context, opts ...ClientOption) *Client {
	c, err := NewClient(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return c
}

// NewRequestEnv builds an envelope for a request gated by a policy. It sets the
// payload to text, attaches the provided security labels (e.g. "tainted"), and
// emits an action_operation fact so policies keyed on the action name can match.
func NewRequestEnv(text, action string, labels []string) core.Envelope {
	env := core.NewEnvelope(text)
	env.SecurityLabels = labels
	if action != "" {
		env.Facts = append(env.Facts, fmt.Sprintf("action_operation(%q, %q)", core.EntityInput, action))
	}
	return env
}
