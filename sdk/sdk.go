package sdk

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	funcAdapter "github.com/duynguyendang/manglekit/adapters/func"
	mcpAdapter "github.com/duynguyendang/manglekit/adapters/mcp"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/engine"
	"github.com/duynguyendang/manglekit/guard"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/telemetry"
	"go.uber.org/zap"
)

const (
	// TracerName is the instrumentation scope name for Manglekit tracing.
	TracerName = "github.com/duynguyendang/manglekit/sdk"
)

// Client is the public struct representing the initialized Manglekit system.
// It provides the core governance capabilities through a simple, unified API.
type Client struct {
	engine     *engine.PolicyEngine
	tracer     core.Tracer
	otelTracer trace.Tracer
	logger     core.Logger
	memory     core.MemoryStore
	registry   map[string]core.Action
}

type ClientOption func(*Client)

func WithMemory(store core.MemoryStore) ClientOption {
	return func(c *Client) {
		if store != nil {
			c.memory = store
		}
	}
}

func WithTracerProvider(tp trace.TracerProvider) ClientOption {
	return func(c *Client) {
		if tp != nil {
			otelTracer := tp.Tracer(TracerName)
			c.otelTracer = otelTracer
			c.tracer = telemetry.NewOTelTracer(otelTracer)
		}
	}
}

func WithLogger(logger core.Logger) ClientOption {
	return func(c *Client) {
		if logger != nil {
			c.logger = logger
		}
	}
}

func NewClient(ctx context.Context, policyFile string, opts ...ClientOption) (*Client, error) {
	c := &Client{
		logger:   core.NopLogger{},
		registry: make(map[string]core.Action),
		memory:   core.NoOpStore{},
	}

	for _, opt := range opts {
		opt(c)
	}

	// SAFETY FIX: Ensure tracer is never nil to prevent nil pointer dereferences
	if c.tracer == nil {
		c.otelTracer = trace.NewNoopTracerProvider().Tracer(TracerName)
		c.tracer = telemetry.NewOTelTracer(c.otelTracer)
	}

	// Initialize Engine with observability
	c.engine = engine.NewWithObservability(c.tracer, c.logger)

	// Load policy from file if provided
	if policyFile != "" {
		if err := c.engine.LoadFromPath(policyFile); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// NewClientWithConfig creates a new Manglekit Client from a pre-loaded configuration object.
func NewClientWithConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
	// Initialize logger (use default for now)
	log := newDefaultLogger()

	// Create client with loaded configuration
	c := &Client{
		logger:   log,
		registry: make(map[string]core.Action),
		memory:   core.NoOpStore{},
	}

	for _, opt := range opts {
		opt(c)
	}

	// SAFETY FIX: Ensure tracer is never nil to prevent nil pointer dereferences
	if c.tracer == nil {
		c.otelTracer = trace.NewNoopTracerProvider().Tracer(TracerName)
		c.tracer = telemetry.NewOTelTracer(c.otelTracer)
	}

	// Initialize Engine with observability
	c.engine = engine.NewWithObservability(c.tracer, c.logger)

	// Load policy from the configured path
	if cfg != nil && cfg.Policy.Path != "" {
		if err := c.engine.LoadFromPath(cfg.Policy.Path); err != nil {
			return nil, fmt.Errorf("failed to load policy from %q: %w", cfg.Policy.Path, err)
		}
	}

	// Load knowledge from the configured path
	if cfg != nil && cfg.Knowledge.Path != "" {
		if err := c.engine.LoadKnowledge(cfg.Knowledge.Path); err != nil {
			return nil, fmt.Errorf("failed to load knowledge from %q: %w", cfg.Knowledge.Path, err)
		}
	}

	// Log configuration loaded successfully
	if cfg != nil {
		c.logger.Info("Manglekit client initialized with config",
			"service_name", cfg.Observability.ServiceName,
			"observability_enabled", cfg.Observability.Enabled)
	}

	return c, nil
}

// NewClientFromConfig creates a new Manglekit Client from a configuration file.
// The config file is expected to be in YAML format and can use environment variable
// expansion (e.g., ${API_KEY}).
//
// This is the recommended way to initialize Manglekit in production environments,
// as it allows configuration to be managed externally via files and environment variables.
//
// Example:
//
//	client, err := sdk.NewClientFromConfig(ctx, "mangle.yaml")
//	if err != nil {
//		log.Fatal(err)
//	}
func NewClientFromConfig(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error) {
	// Load configuration from file
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	c, err := NewClientWithConfig(ctx, cfg, opts...)
	if err != nil {
		return nil, err
	}

	// Load MCP Actions
	if len(cfg.MCP) > 0 {
		mcpActions, err := mcpAdapter.Load(ctx, cfg.MCP, c.logger)
		if err != nil {
			c.logger.Error("failed to load MCP actions", "error", err)
		}

		for _, action := range mcpActions {
			// Protect It
			safeAction := c.Protect(action)
			// Register It
			c.RegisterAction(safeAction.Metadata().Name, safeAction)
		}
	}

	return c, nil
}

// Protect is the final, public API method.
// It takes any raw Action and wraps it with the governance Guard.
//
// This is the single-line value proposition of Manglekit:
// any Action becomes governed by policies with zero code changes.
//
// Example:
//
//	rawAction := myservice.NewDatabaseAction()
//	protectedAction := client.Protect(rawAction)
//	result, err := protectedAction.Execute(ctx, input)
func (c *Client) Protect(action core.Action) core.Action {
	if c.tracer != nil {
		return guard.NewWithTracer(action, c.engine, c.tracer)
	}
	return guard.New(action, c.engine)
}

// Engine returns the underlying PolicyEngine for advanced use cases.
// Most users should use Protect() instead.
func (c *Client) Engine() *engine.PolicyEngine {
	return c.engine
}

// Tracer returns the OTel tracer used by this client.
// This can be used for custom instrumentation in user code.
func (c *Client) Tracer() trace.Tracer {
	return c.otelTracer
}

// Logger returns the Logger used by this client.
// This can be used for custom logging in user code.
func (c *Client) Logger() core.Logger {
	return c.logger
}

// ProtectFunc is a generic helper that wraps a Go function into an Action
// and then protects it with the Guard.
// It uses generics to ensure type safety of the function adapter.
//
// Example:
//
//	protectedAction := sdk.ProtectFunc(client, "checkStock", CheckStock)
//	result, err := sdk.Call(ctx, protectedAction, input)
func ProtectFunc[In any, Out any](c *Client, name string, fn func(context.Context, In) (Out, error)) core.Action {
	adapter := funcAdapter.New(name, fn)
	return c.Protect(adapter)
}

// Must is a helper that panics if the error is not nil.
// Useful for initializing the client in main() or init() when you want
// to fail-fast on startup errors.
//
// Example:
//
//	client := sdk.Must(sdk.NewClient(ctx, "policy.dl", sdk.WithLogger(log)))
func Must(c *Client, err error) *Client {
	if err != nil {
		panic(err)
	}
	return c
}

// NewDefault creates a Client with default settings (Zap logger, empty policy).
// This is the simplest way to get started with Manglekit.
//
// Example:
//
//	client, err := sdk.NewDefault()
func NewDefault() (*Client, error) {
	return NewClient(context.Background(), "", WithLogger(newDefaultLogger()))
}

func newDefaultLogger() core.Logger {
	z, err := zap.NewProduction()
	if err != nil {
		return logger.NewStdLogger()
	}
	return logger.NewZapAdapter(z.Sugar())
}

// Call is a generic helper to execute an action with strongly typed input/output.
// It handles Envelope packing and unpacking automatically, eliminating the need
// for manual type assertions.
//
// Example:
//
//	result, err := sdk.Call[StockResponse](ctx, protectedAction, StockRequest{SKU: "IPHONE"})
func Call[Out any](ctx context.Context, action core.Action, payload any) (Out, error) {
	// 1. Pack payload into Envelope
	req := core.NewEnvelope(payload)

	// 2. Execute action
	res, err := action.Execute(ctx, req)
	if err != nil {
		var zero Out
		return zero, err
	}

	// 3. Unpack and cast result
	out, ok := res.Payload.(Out)
	if !ok {
		var zero Out
		return zero, fmt.Errorf("output payload type mismatch: expected %T, got %T", zero, res.Payload)
	}

	return out, nil
}

// RegisterAction registers an action with the client for use in RunLoop.
func (c *Client) RegisterAction(name string, action core.Action) {
	c.registry[name] = action
}
