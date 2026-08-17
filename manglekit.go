package manglekit

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/duynguyendang/manglekit/sdk/ports"
	"go.opentelemetry.io/otel/trace"
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

// QuickClient is the shortest path to a governed client: it constructs a
// Client and loads the Datalog policy at policyPath (same as
// WithPolicyPath), returning typed errors for a missing file or an invalid
// policy. For anything beyond a policy file, use NewClient with options.
//
//	client, err := manglekit.QuickClient(ctx, "policy.dl")
func QuickClient(ctx context.Context, policyPath string) (*Client, error) {
	return sdk.NewClient(ctx, sdk.WithPolicyPath(policyPath))
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
//
// Every sdk.ClientOption is re-exported here so application code never
// needs to import the sdk package directly.

// WithPolicyPath specifies the file path of the Datalog policy to load at
// client construction. This is the canonical option name; it matches config
// `policy.path` and the CLI `--policy` flag.
func WithPolicyPath(path string) ClientOption { return sdk.WithPolicyPath(path) }

// WithBlueprintPath specifies the file path to load Datalog rules from.
//
// Deprecated: use WithPolicyPath instead (removal target: v0.8).
func WithBlueprintPath(path string) ClientOption { return sdk.WithBlueprintPath(path) }

// WithEngine allows injecting a custom or mock core.Evaluator.
func WithEngine(e core.Evaluator) ClientOption { return sdk.WithEngine(e) }

// WithSteeringEnabled enables policy-driven routing (route/retry).
func WithSteeringEnabled() ClientOption { return sdk.WithSteeringEnabled() }

// WithLogger sets a custom logger for the client.
func WithLogger(l core.Logger) ClientOption { return sdk.WithLogger(l) }

// WithTracerProvider configures the OpenTelemetry tracer provider.
func WithTracerProvider(tp trace.TracerProvider) ClientOption {
	return sdk.WithTracerProvider(tp)
}

// WithStdoutTracer enables a console tracer for debugging.
func WithStdoutTracer() ClientOption { return sdk.WithStdoutTracer() }

// WithMemory replaces the whole memory implementation (history + RAG).
func WithMemory(mem core.AgentMemory) ClientOption { return sdk.WithMemory(mem) }

// WithAgentMemory is an alias of WithMemory.
//
// Deprecated: use WithMemory instead.
func WithAgentMemory(mem core.AgentMemory) ClientOption { return sdk.WithAgentMemory(mem) }

// WithHistory sets only the chat-history component of the memory, composing
// with an existing HybridMemory. It returns an error at construction time
// (rather than silently discarding it) if a custom non-hybrid memory was
// configured via WithMemory.
func WithHistory(store core.HistoryStore) ClientOption { return sdk.WithHistory(store) }

// WithLLM configures the AI backend for the client.
func WithLLM(gen core.TextGenerator) ClientOption { return sdk.WithLLM(gen) }

// WithStateProvider configures durable session persistence.
func WithStateProvider(provider core.StateProvider) ClientOption {
	return sdk.WithStateProvider(provider)
}

// WithConfigFile loads configuration from a YAML file and applies it.
func WithConfigFile(path string) ClientOption { return sdk.WithConfigFile(path) }

// WithConfig applies settings from a loaded configuration struct.
func WithConfig(cfg *config.Config) ClientOption { return sdk.WithConfig(cfg) }

// WithProviderConfig wires a provider-backed action from config.
func WithProviderConfig(name string, cfg config.ActionConfig) ClientOption {
	return sdk.WithProviderConfig(name, cfg)
}

// WithExtractor sets the text-to-struct extractor on a supervised action,
// enabling the neuro-symbolic bridge.
func WithExtractor(action core.Action, ext ports.Extractor) core.Action {
	return sdk.WithExtractor(action, ext)
}

// WithSessionID activates persistent stateful mode for the execution.
func WithSessionID(id string) ExecuteOption { return sdk.WithSessionID(id) }

// WithTransientMemory activates in-memory stateful mode.
func WithTransientMemory() ExecuteOption { return sdk.WithTransientMemory() }

// WithMetadata injects custom key-value pairs into the execution envelope's metadata.
func WithMetadata(key, value string) ExecuteOption { return sdk.WithMetadata(key, value) }

// MustReadFile reads the file at path and panics on error. It is a small
// convenience for example/demo setup code where a missing fixture is fatal.
//
// The path is resolved cwd-independently for relative paths: if the file
// cannot be found relative to the current working directory, it is resolved
// relative to the source directory of MustReadFile's caller (via
// runtime.Caller). This makes examples and tests work whether they are run
// from the repo root, the example's directory, or `go test`.
func MustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err == nil {
		return data
	}
	if !filepath.IsAbs(path) {
		if _, file, _, ok := runtime.Caller(1); ok {
			alt := filepath.Join(filepath.Dir(file), path)
			if data2, err2 := os.ReadFile(alt); err2 == nil {
				return data2
			}
		}
	}
	panic(fmt.Errorf("MustReadFile %q: %w", path, err))
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
