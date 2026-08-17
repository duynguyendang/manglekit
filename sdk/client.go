package sdk

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go.opentelemetry.io/otel/trace"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/supervisor"
	"github.com/duynguyendang/manglekit/providers/predicates"
)

// TracerName is the instrumentation scope name for Manglekit tracing.
const TracerName = "github.com/duynguyendang/manglekit/sdk"

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
	// registryLock guards concurrent access to the registry.
	registryLock sync.RWMutex
	// blueprintPath stores the path loaded at startup (for debugging/reloading).
	blueprintPath string
	// shutdownFunc is a cleanup function to stop exporters/tracers.
	shutdownFunc func(context.Context) error
	// llm is the plugged-in text generation backend (e.g., Google, OpenAI).
	llm core.TextGenerator
	// maxSteps limits the total number of loop iterations.
	// Zero means use the SDK default (DefaultMaxSteps).
	maxSteps int
	// steeringEnabled controls whether EAST prompt injection is active.
	steeringEnabled bool
	// paradoxThreshold is the EAST magnitude above which cognitive paradox injection
	// is triggered. Default: 0.8.
	paradoxThreshold float64
	// stateManager handles durable state persistence and recovery.
	stateManager core.StateManager
	// initCtx is the context passed to NewClient, reused during deferred
	// (option-driven) initialization such as blueprint loading and memory
	// hydration, so those I/O operations respect cancellation/timeout.
	initCtx context.Context
	// memWg tracks in-flight background memory operations.
	memWg sync.WaitGroup
	// memMu serializes checks of shuttingDown against memWg.Add,
	// establishing the happens-before edge required by sync.WaitGroup.
	memMu sync.Mutex
	// shuttingDown is set to true by Shutdown and never unset.
	shuttingDown bool
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
		initCtx:     ctx,
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

	// Register reference external predicates (time, rate, identity).
	// These are available to all Datalog policies by default.
	if err := predicates.RegisterAll(c.engine); err != nil {
		return nil, fmt.Errorf("failed to register reference predicates: %w", err)
	}

	// Load deferred policy now that engine is ready.
	// Uses the caller's context for proper cancellation/timeout propagation.
	if c.blueprintPath != "" {
		content, err := os.ReadFile(c.blueprintPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read policy %q: %w", c.blueprintPath, err)
		}
		if err := c.engine.LoadPolicy(ctx, string(content)); err != nil {
			return nil, fmt.Errorf("failed to load policy %q: %w", c.blueprintPath, err)
		}
	}

	return c, nil
}

// Supervise wraps a raw core.Action in a SupervisedAction using v2 patterns.
func (c *Client) Supervise(action core.Action) core.Action {
	return supervisor.NewSupervisedActionFromSDK(action, c.engine, c.logger)
}

// Engine returns the underlying policy engine (Evaluator).
func (c *Client) Engine() core.Evaluator {
	return c.engine
}

// ParadoxThreshold returns the configured EAST paradox-injection threshold.
// Pass it to ooda.OODAFlowConfig.ParadoxThreshold to steer the OODA loop.
func (c *Client) ParadoxThreshold() float64 {
	return c.paradoxThreshold
}

// LoadFacts allows manually injecting straight Datalog facts into the engine.
func (c *Client) LoadFacts(ctx context.Context, facts []string) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return c.engine.LoadFacts(ctx, facts)
}

// RegisterExternalPredicate registers a Go callback as an external Datalog
// predicate, allowing policies to call into Go (e.g. for PII scans, HTTP
// requests, custom checks). External predicates registered this way are
// auto-declared (`Decl ... external()`) by all policy load paths
// (WithPolicyPath, LoadPolicy, LoadFromSource), so the registration may
// happen before or after the policy is loaded.
//
// Example:
//
//	client.RegisterExternalPredicate("pii_scan",
//	    func(_ context.Context, inputs []any) ([][]any, error) {
//	        s, _ := inputs[0].(string)
//	        if ssnPattern.MatchString(s) {
//	            return [][]any{{s}}, nil
//	        }
//	        return nil, nil
//	    })
func (c *Client) RegisterExternalPredicate(name string, fn func(ctx context.Context, inputs []any) ([][]any, error)) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return c.engine.RegisterExternalPredicate(name, fn)
}

// Policy loading — three entry points, one rule of thumb:
//
//  1. WithPolicyPath (or QuickClient): the blessed path for file-based
//     policies; the file is read and loaded during NewClient with typed errors.
//  2. LoadPolicy: append Datalog rules from a string at runtime. Since v0.7
//     this path auto-emits `Decl ... external()` declarations for registered
//     external predicates (like LoadFromSource), so load order relative to
//     RegisterExternalPredicate no longer matters.
//  3. LoadFromSource: REPLACE the whole program from a string (base facts
//     loaded beforehand are preserved). Use for full policy reloads.
//
// Use WithPolicyPath/LoadPolicy for incremental loads; LoadFromSource only
// when you intentionally want to discard existing rules.

// LoadPolicy loads policy rules from a raw Datalog string, adding them to
// the existing program. Registered external predicates are auto-declared,
// so a policy referencing RegisterExternalPredicate predicates loads cleanly
// here — no need for the LoadFromSource workaround.
func (c *Client) LoadPolicy(ctx context.Context, policy string) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return c.engine.LoadPolicy(ctx, policy)
}

// LoadFromSource loads a Datalog program from a raw string, replacing any
// existing program state (base facts survive). See the "Policy loading"
// block above for when to prefer it over LoadPolicy.
func (c *Client) LoadFromSource(ctx context.Context, source string) error {
	if c.engine == nil {
		return fmt.Errorf("engine not initialized")
	}
	return c.engine.LoadFromSource(ctx, source)
}

// Tracer returns the OpenTelemetry Tracer used by the client.
func (c *Client) Tracer() trace.Tracer {
	return c.otelTracer
}

// Logger returns the configured Logger instance.
func (c *Client) Logger() core.Logger {
	return c.logger
}

// SetLLM manually configures the TextGenerator (LLM) for the client.
// This is useful for code-first wiring or when using provider factories.
func (c *Client) SetLLM(gen core.TextGenerator) {
	c.llm = gen
}

// RegisterAction adds an action to the client's internal registry.
func (c *Client) RegisterAction(name string, action core.Action) {
	c.registryLock.Lock()
	defer c.registryLock.Unlock()

	c.registry[name] = action
	if c.engine != nil {
		if err := c.engine.RegisterAction(action.Metadata()); err != nil {
			c.logger.Warn("failed to register action metadata to engine", "action", name, "error", err)
		}
	}
}

// RegisterSupervised is the one-liner for the most common setup pattern:
// it wraps act in the Zero-Trust gatekeeper (Supervise) and registers the
// supervised action under name, so it is immediately callable via
// ExecuteByName. It returns the supervised action.
//
//	client.RegisterSupervised("my_action", rawAction)
//
// is equivalent to
//
//	client.RegisterAction("my_action", client.Supervise(rawAction))
func (c *Client) RegisterSupervised(name string, act core.Action) core.Action {
	supervised := c.Supervise(act)
	c.RegisterAction(name, supervised)
	return supervised
}

// Shutdown cleans up resources used by the client.
// Safe to call multiple times; subsequent calls return nil.
func (c *Client) Shutdown(ctx context.Context) error {
	c.memMu.Lock()
	if c.shuttingDown {
		c.memMu.Unlock()
		return nil
	}
	c.shuttingDown = true
	c.memMu.Unlock()

	// Drain in-flight asyncMemorize goroutines, bounded by ctx.
	done := make(chan struct{})
	go func() {
		c.memWg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	if c.shutdownFunc != nil {
		return c.shutdownFunc(ctx)
	}
	return nil
}

// Memory returns the active memory provider (if any).
func (c *Client) Memory() core.AgentMemory {
	return c.agentMemory
}
