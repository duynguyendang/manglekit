package sdk

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit/adapters/knowledge"
	mcpAdapter "github.com/duynguyendang/manglekit/adapters/mcp"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/supervisor"
	"github.com/duynguyendang/manglekit/internal/telemetry"
)

const (
	// TracerName is the instrumentation scope name for Manglekit tracing.
	TracerName = "github.com/duynguyendang/manglekit/sdk"
)

// Client is the primary entry point for the Manglekit system.
// It acts as the governance kernel, managing blueprints, observability, and action execution.
// Applications should create a single Client instance and reuse it.
type Client struct {
	// engine is the internal Policy Engine responsible for Datalog evaluation.
	engine core.Evaluator
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
	// blueprintPath stores the path loaded at startup (for debugging/reloading).
	blueprintPath string
	// shutdownFunc is a cleanup function to stop exporters/tracers.
	shutdownFunc func(context.Context) error
	// llm is the plugged-in text generation backend.
	llm core.TextGenerator
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
		logger:   logger.NewDefault(),
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

	// Load blueprint from file if provided
	if c.blueprintPath != "" {
		content, err := os.ReadFile(c.blueprintPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read blueprint file: %w", err)
		}
		if err := c.engine.LoadPolicy(ctx, string(content)); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// NewClientWithConfig initializes a Client using a loaded Config object.
// It performs full wiring: Engine, Policy, Knowledge, and Actions.
//
// Parameters:
//   - ctx: The context.
//   - cfg: The loaded configuration struct.
//   - opts: Additional functional options (override config settings).
//
// Returns:
//   - A pointer to the Client, or an error.
func NewClientWithConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
	// Initialize logger
	var log core.Logger
	if cfg != nil && cfg.Observability.LogLevel != "" {
		log = logger.New(cfg.Observability.LogLevel)
	} else {
		log = logger.NewDefault()
	}

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

	if cfg == nil {
		return c, nil
	}

	// 1. Load Policy
	if cfg.Policy.Path != "" {
		content, err := os.ReadFile(cfg.Policy.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read policy file %q: %w", cfg.Policy.Path, err)
		}
		if err := c.engine.LoadPolicy(ctx, string(content)); err != nil {
			return nil, fmt.Errorf("failed to load policy from %q: %w", cfg.Policy.Path, err)
		}
	}

	// 2. Load Knowledge
	if cfg.Knowledge.Path != "" {
		path := cfg.Knowledge.Path
		var facts []string
		var err error

		if strings.HasSuffix(path, ".nt") {
			f, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("failed to open knowledge file %q: %w", path, err)
			}
			loader := knowledge.NewNTriplesLoader()
			facts, err = loader.Parse(f)
			f.Close()
		} else {
			// Default to RDF Loader (Turtle/XML)
			loader := knowledge.NewRDFLoader()
			facts, err = loader.Parse(path)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to parse knowledge from %q: %w", path, err)
		}

		if err := c.engine.LoadFacts(facts); err != nil {
			return nil, fmt.Errorf("failed to load knowledge facts from %q: %w", path, err)
		}
	}

	// 3. Hydrate Actions
	if len(cfg.Actions) > 0 {
		actions, err := HydrateActions(cfg.Actions)
		if err != nil {
			return nil, err
		}
		for _, action := range actions {
			// Register it
			c.RegisterAction(action.Metadata().Name, action)
		}
	}

	// 4. Load MCP Actions
	if len(cfg.MCP) > 0 {
		for _, mcpCfg := range cfg.MCP {
			loader := mcpAdapter.NewLoader(mcpCfg).WithLogger(c.logger)
			actions, err := loader.Load(ctx)
			if err != nil {
				return nil, fmt.Errorf("critical tool '%s' failed to load: %w", mcpCfg.Name, err)
			}

			for _, action := range actions {
				safeAction := c.Supervise(action)
				c.RegisterAction(safeAction.Metadata().Name, safeAction)
				c.logger.Info("Discovered MCP Tool", "name", safeAction.Metadata().Name)
			}
		}
	}

	// Set failure mode
	if cfg.FailureMode != "" {
		c.failureMode = cfg.FailureMode
	}

	// Log configuration loaded successfully
	c.logger.Info("Manglekit client initialized with config",
		"service_name", cfg.Observability.ServiceName,
		"observability_enabled", cfg.Observability.Enabled,
		"failure_mode", c.failureMode)

	return c, nil
}

// NewClientFromFile initializes a Client by loading configuration from a YAML file.
// It supports environment variable expansion in the config file.
//
// Parameters:
//   - ctx: The context.
//   - configPath: Path to the YAML configuration file.
//   - opts: Additional functional options.
//
// Returns:
//   - A pointer to the Client, or an error.
func NewClientFromFile(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error) {
	// Load configuration from file
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return NewClientWithConfig(ctx, cfg, opts...)
}

// NewClientFromConfig is a wrapper for NewClientWithConfig to match user requirements.
func NewClientFromConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
	return NewClientWithConfig(ctx, cfg, opts...)
}

// Supervise wraps a raw core.Action in a SupervisedAction.
// This applies the "Trace -> Authorize -> Execute -> Validate" governance lifecycle.
// This is the core function of the Manglekit framework.
//
// Parameters:
//   - action: The action to supervise.
//
// Returns:
//   - A new core.Action that enforces blueprints.
func (c *Client) Supervise(action core.Action) core.Action {
	if c.tracer != nil {
		return supervisor.NewSupervisedActionWithTracer(action, c.engine, c.tracer, c.failureMode)
	}
	return supervisor.NewSupervisedAction(action, c.engine, c.failureMode)
}

func (c *Client) Engine() core.Evaluator {
	return c.engine
}

// LoadFacts allows manually injecting straight Datalog facts into the engine.
// This supports the "Explicit Loading" workflow where adapters parse data first.
func (c *Client) LoadFacts(facts []string) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return c.engine.LoadFacts(facts)
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

// NewDefault initializes a Client with sensible default settings:
//   - Default internal logger (slog).
//   - No-op tracer.
//   - No policy loaded (allow-all default).
//
// Returns:
//   - A pointer to the Client, or an error.
func NewDefault() (*Client, error) {
	return NewClient(context.Background())
}

// RegisterAction adds an action to the client's internal registry.
// Registered actions can be invoked by name using ExecuteByName, enabling dynamic routing.
//
// Parameters:
//   - name: The unique name for the action.
//   - action: The action instance.
func (c *Client) RegisterAction(name string, action core.Action) {
	c.registry[name] = action
	if c.engine != nil {
		if err := c.engine.RegisterAction(action.Metadata()); err != nil {
			c.logger.Warn("failed to register action metadata to engine", "action", name, "error", err)
		}
	}
}

// Shutdown cleans up resources used by the client, such as flushing traces.
func (c *Client) Shutdown(ctx context.Context) error {
	if c.shutdownFunc != nil {
		return c.shutdownFunc(ctx)
	}
	return nil
}
