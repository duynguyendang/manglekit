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

// Client is the primary entry point for the Manglekit system.
// It acts as the governance kernel, managing policies, observability, and action execution.
// Applications should create a single Client instance and reuse it.
type Client struct {
	// engine is the internal Policy Engine responsible for Datalog evaluation.
	engine *engine.PolicyEngine
	// tracer is the Manglekit core.Tracer wrapper.
	tracer core.Tracer
	// otelTracer is the raw OpenTelemetry tracer instance.
	otelTracer trace.Tracer
	// logger is the structured logger used by the client and its components.
	logger core.Logger
	// memory is the persistence layer for chat history (optional).
	memory core.MemoryStore
	// registry holds registered actions for dynamic routing.
	registry map[string]core.Action
	// failureMode determines the system's resilience strategy ("open" or "closed").
	failureMode string
	// initialPolicyPath stores the path loaded at startup (for debugging/reloading).
	initialPolicyPath string
	// shutdownFunc is a cleanup function to stop exporters/tracers.
	shutdownFunc func(context.Context) error
}

// NewClient initializes a new Manglekit Client with the provided options.
// It sets up the Policy Engine, Observability (Logging/Tracing), and default configurations.
//
// Parameters:
//   - ctx: The initialization context.
//   - opts: A variadic list of ClientOption configuration functions.
//
// Returns:
//   - A pointer to the initialized Client, or an error if initialization fails.
func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
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
	if c.initialPolicyPath != "" {
		if err := c.engine.LoadFromPath(c.initialPolicyPath); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// NewClientWithConfig initializes a Client using a loaded Config object.
// This is useful when configuration is deserialized from a file or external source.
//
// Parameters:
//   - ctx: The context.
//   - cfg: The loaded configuration struct.
//   - opts: Additional functional options (override config settings).
//
// Returns:
//   - A pointer to the Client, or an error.
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

	// Set failure mode from config
	if cfg != nil && cfg.FailureMode != "" {
		c.failureMode = cfg.FailureMode
	}

	// Log configuration loaded successfully
	if cfg != nil {
		c.logger.Info("Manglekit client initialized with config",
			"service_name", cfg.Observability.ServiceName,
			"observability_enabled", cfg.Observability.Enabled,
			"failure_mode", c.failureMode)
	}

	return c, nil
}

// NewClientFromConfig initializes a Client by loading configuration from a YAML file.
// It supports environment variable expansion in the config file.
//
// Parameters:
//   - ctx: The context.
//   - configPath: Path to the YAML configuration file.
//   - opts: Additional functional options.
//
// Returns:
//   - A pointer to the Client, or an error.
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
		for _, mcpCfg := range cfg.MCP {
			loader := mcpAdapter.NewLoader(mcpCfg)
			actions, err := loader.Load(ctx)
			if err != nil {
				if mcpCfg.FailOnStartup {
					// Critical failure: Bubble up the error to stop initialization
					return nil, fmt.Errorf("critical tool '%s' failed to load: %w", mcpCfg.Name, err)
				}
				// Soft failure: Log warning and continue (Graceful Degradation)
				c.logger.Warn("Optional tool failed to load, skipping", "tool", mcpCfg.Name, "err", err)
				continue
			}

			for _, action := range actions {
				// Protect It
				safeAction := c.Protect(action)
				// Register It
				c.RegisterAction(safeAction.Metadata().Name, safeAction)
				c.logger.Info("Discovered MCP Tool", "name", safeAction.Metadata().Name)
			}
		}
	}

	return c, nil
}

// Protect wraps a raw core.Action in a GuardedAction.
// This applies the "Trace -> Authorize -> Execute -> Validate" governance lifecycle.
// This is the core function of the Manglekit framework.
//
// Parameters:
//   - action: The action to protect.
//
// Returns:
//   - A new core.Action that enforces policies.
func (c *Client) Protect(action core.Action) core.Action {
	if c.tracer != nil {
		return guard.NewWithTracer(action, c.engine, c.tracer, c.failureMode)
	}
	return guard.New(action, c.engine, c.failureMode)
}

// Engine returns the underlying PolicyEngine instance.
// This provides access to low-level engine methods like RecordLineage or explicit evaluation.
func (c *Client) Engine() *engine.PolicyEngine {
	return c.engine
}

// Tracer returns the OpenTelemetry Tracer used by the client.
// This allows users to start their own spans that are linked to the Manglekit trace context.
func (c *Client) Tracer() trace.Tracer {
	return c.otelTracer
}

// Logger returns the configured Logger instance.
func (c *Client) Logger() core.Logger {
	return c.logger
}

// ProtectFunc is a generic helper that wraps a Go function into a protected Action.
// It combines the functional adapter with the governance guard in one step.
//
// Parameters:
//   - c: The Manglekit Client.
//   - name: The name for the action (used in policy and tracing).
//   - fn: The function to wrap. It must accept Context and Input, and return Output and error.
//
// Returns:
//   - A protected core.Action.
func ProtectFunc[In any, Out any](c *Client, name string, fn func(context.Context, In) (Out, error)) core.Action {
	adapter := funcAdapter.New(name, fn)
	return c.Protect(adapter)
}


// NewDefault initializes a Client with sensible default settings:
//   - Zap production logger (or standard output if Zap fails).
//   - No-op tracer.
//   - No policy loaded (allow-all default).
//
// Returns:
//   - A pointer to the Client, or an error.
func NewDefault() (*Client, error) {
	return NewClient(context.Background(), WithLogger(newDefaultLogger()))
}

func newDefaultLogger() core.Logger {
	z, err := zap.NewProduction()
	if err != nil {
		return logger.NewStdLogger()
	}
	return logger.NewZapAdapter(z.Sugar())
}

// Call is a generic helper to execute a protected action with type-safe input and output.
// It handles packing the input into an Envelope and unpacking the output.
//
// Parameters:
//   - ctx: The execution context.
//   - action: The protected action to execute.
//   - payload: The strongly-typed input payload.
//
// Returns:
//   - The strongly-typed output payload, or an error.
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

// RegisterAction adds an action to the client's internal registry.
// Registered actions can be invoked by name using ExecuteByName, enabling dynamic routing.
//
// Parameters:
//   - name: The unique name for the action.
//   - action: The action instance.
func (c *Client) RegisterAction(name string, action core.Action) {
	c.registry[name] = action
}

// Shutdown cleans up resources used by the client, such as flushing traces.
func (c *Client) Shutdown(ctx context.Context) error {
	if c.shutdownFunc != nil {
		return c.shutdownFunc(ctx)
	}
	return nil
}
