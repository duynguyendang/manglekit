package sdk

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit-wip/config"
	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/duynguyendang/manglekit-wip/internal/logger"
	"github.com/duynguyendang/manglekit-wip/internal/supervisor"
)

const (
	// TracerName is the instrumentation scope name for Manglekit tracing.
	TracerName = "github.com/duynguyendang/manglekit-wip/sdk"

	// Failure modes determine the system's resilience strategy.
	FailModeOpen   = "open"   // Allow execution on system error (Fail-Open)
	FailModeClosed = "closed" // Block execution on system error (Fail-Closed)
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
	// agentMemory is the unified memory provider (History + RAG).
	agentMemory core.AgentMemory
	// registry holds registered actions for dynamic routing.
	registry map[string]core.Action
	// failureMode determines the system's resilience strategy ("open" or "closed").
	failureMode string
	// blueprintPath stores the path loaded at startup (for debugging/reloading).
	blueprintPath string
	// shutdownFunc is a cleanup function to stop exporters/tracers.
	shutdownFunc func(context.Context) error
	// llm is the plugged-in text generation backend (e.g., Google, OpenAI).
	llm core.TextGenerator
	// stateManager handles durable state persistence and recovery.
	stateManager interface {
		Hydrate(ctx context.Context, sessionID string) (*core.SessionState, error)
		Checkpoint(ctx context.Context, state *core.SessionState) error
		ExtractFacts(ctx context.Context, envelope core.Envelope) ([]string, error)
	}
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
		logger: logger.NewDefault(),
		// Default to HybridMemory with Nop stores
		agentMemory: NewHybridMemory(core.NopStore{}, core.NopVectorStore{}, core.NopEmbedder{}),
		registry:    make(map[string]core.Action),
		failureMode: FailModeClosed, // Default to closed
	}

	// Apply options
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	// Ensure dependencies (Engine/Tracer) are initialized if JIT init didn't happen
	if err := ensureDependencies(c); err != nil {
		return nil, err
	}

	return c, nil
}

// NewClientFromFile initializes a Client by loading configuration from a YAML file.
func NewClientFromFile(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error) {
	// Load configuration from file
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Prepend WithConfig to opts
	newOpts := append([]ClientOption{WithConfig(cfg)}, opts...)
	return NewClient(ctx, newOpts...)
}

// NewClientFromConfig initializes a Client using a pre-loaded Config object.
func NewClientFromConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error) {
	newOpts := append([]ClientOption{WithConfig(cfg)}, opts...)
	return NewClient(ctx, newOpts...)
}

// Supervise wraps a raw core.Action in a SupervisedAction using v2 patterns.
func (c *Client) Supervise(action core.Action) core.Action {
	return supervisor.NewSupervisedActionFromSDK(action, c.engine)
}

// Engine returns the underlying policy engine (Evaluator).
func (c *Client) Engine() core.Evaluator {
	return c.engine
}

// LoadFacts allows manually injecting straight Datalog facts into the engine.
func (c *Client) LoadFacts(facts []string) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return c.engine.LoadFacts(facts)
}

// Tracer returns the OpenTelemetry Tracer used by the client.
func (c *Client) Tracer() trace.Tracer {
	return c.otelTracer
}

// Logger returns the configured Logger instance.
func (c *Client) Logger() core.Logger {
	return c.logger
}

// NewDefault initializes a Client with sensible default settings.
func NewDefault() (*Client, error) {
	return NewClient(context.Background())
}

// SetLLM manually configures the TextGenerator (LLM) for the client.
// This is useful for code-first wiring or when using provider factories.
func (c *Client) SetLLM(gen core.TextGenerator) {
	c.llm = gen
}

// RegisterAction adds an action to the client's internal registry.
func (c *Client) RegisterAction(name string, action core.Action) {
	c.registry[name] = action
	if c.engine != nil {
		if err := c.engine.RegisterAction(action.Metadata()); err != nil {
			c.logger.Warn("failed to register action metadata to engine", "action", name, "error", err)
		}
	}
}

// Shutdown cleans up resources used by the client.
func (c *Client) Shutdown(ctx context.Context) error {
	if c.shutdownFunc != nil {
		return c.shutdownFunc(ctx)
	}
	return nil
}

// Memory returns the active memory provider (if any).
func (c *Client) Memory() core.AgentMemory {
	return c.agentMemory
}
