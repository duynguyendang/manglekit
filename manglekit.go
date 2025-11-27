package manglekit

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
	TracerName = "github.com/duynguyendang/manglekit"
)

// Client is the public struct representing the initialized Manglekit system.
// It provides the core governance capabilities through a simple, unified API.
type Client struct {
	engine     *engine.PolicyEngine
	tracer     core.Tracer
	otelTracer trace.Tracer
	logger     core.Logger
	actions    map[string]core.Action
}

type ClientOption func(*Client)

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
		logger:  core.NopLogger{},
		actions: make(map[string]core.Action),
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
		logger:  log,
		actions: make(map[string]core.Action),
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
//	client, err := manglekit.NewClientFromConfig(ctx, "mangle.yaml")
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
//	protectedAction := manglekit.ProtectFunc(client, "checkStock", CheckStock)
//	result, err := manglekit.Call(ctx, protectedAction, input)
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
//	client := manglekit.Must(manglekit.NewClient(ctx, "policy.dl", manglekit.WithLogger(log)))
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
//	client, err := manglekit.NewDefault()
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
//	result, err := manglekit.Call[StockResponse](ctx, protectedAction, StockRequest{SKU: "IPHONE"})
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
	c.actions[name] = action
}

// RunLoop executes a Semantic State Machine starting from startActionName.
// It handles steering decisions (ALLOW, RETRY, ROUTE) returned by the Policy Engine.
func (c *Client) RunLoop(ctx context.Context, startActionName string, inputPayload any) (core.Envelope, error) {
	currentActionName := startActionName
	payload := inputPayload
	var feedback []string

	const maxSteps = 10 // Prevent infinite loops
	steps := 0

	for steps < maxSteps {
		steps++
		c.logger.Info("RunLoop step", "step", steps, "action", currentActionName)

		// 1. Look up action
		action, ok := c.actions[currentActionName]
		if !ok {
			return core.Envelope{}, fmt.Errorf("action not found: %s", currentActionName)
		}

		// 2. Create Envelope
		env := core.NewEnvelope(payload)
		if len(feedback) > 0 {
			// In a real implementation, we might want to inject this into the payload
			// if it's an LLM prompt. For now, we attach to metadata.
			env.Metadata["prev_feedback"] = fmt.Sprintf("%v", feedback)
		}

		// 3. Execute Action
		// Note: The action should be wrapped by GuardedAction (via Client.Protect)
		// which calls Authorize and Validate.
		// However, steering happens *outside* the action execution but *inside* the loop.
		// Wait, the prompt says "Handle Decision: Check resEnv.Metadata[core.KeyDecision]".
		// This implies the Action (likely GuardedAction) returns the decision in metadata.
		// But currently GuardedAction (in guard/guard.go) might not run EvaluateSteering.
		// I need to check guard/guard.go. If it doesn't, I should run EvaluateSteering here?
		// "Upgrade the Engine and SDK to support Flow Control... Implement the RunLoop... Handle Decision: Check resEnv.Metadata[core.KeyDecision]."
		// This strongly suggests GuardedAction should inject this metadata.
		// But wait, the prompt puts steering logic in Engine.
		// If GuardedAction calls Authorize -> Execute -> Validate, where does Steering fit?
		// If Authorize returns DENY, execution stops.
		// If Validate returns ALLOW, we proceed.
		// Maybe Steering should be called explicitly here in RunLoop?
		// Or maybe Validate should call Steering?
		// The prompt says: "Engine Implementation... Upgrade the Engine to support 'Steering queries'... Implement EvaluateSteering...".
		// And "RunLoop... Handle Decision: Check resEnv.Metadata[core.KeyDecision]".
		// If RunLoop calls `action.Execute`, and that action is a `GuardedAction`, then `GuardedAction` *must* run `EvaluateSteering` and populate metadata.
		//
		// BUT, I haven't modified `guard/guard.go` yet.
		// If I don't modify `guard/guard.go`, `resEnv` won't have the decision from `EvaluateSteering`.
		// So I MUST modify `guard/guard.go` OR call `EvaluateSteering` directly in `RunLoop`.
		// calling it directly in RunLoop seems safer given I wasn't explicitly asked to modify `guard/guard.go`,
		// BUT the prompt says "Check resEnv.Metadata[core.KeyDecision]". This implies the decision is *already* in the result envelope.
		// So `GuardedAction` is the right place.
		//
		// However, `GuardedAction` returns `Envelope`.
		// If I run `EvaluateSteering` *after* execution (in Validate phase?), I can attach it.
		//
		// Let's look at `RunLoop` logic again:
		// "Execute: resEnv, err := currentAction.Execute(ctx, env)."
		// "Handle Decision: Check resEnv.Metadata[core.KeyDecision]."
		//
		// So `currentAction.Execute` must return the decision.
		// I will modify `RunLoop` to call `EvaluateSteering` IF it's not present?
		// No, the prompt is about "Steering & Routing Subsystem".
		// It's cleaner if `RunLoop` coordinates this using the Engine directly if the Action didn't.
		// BUT, `Client` has `engine`.
		//
		// Let's implement it in `RunLoop` by calling `c.engine.EvaluateSteering` explicitly for now,
		// and enriching the result envelope.
		//
		// WAIT, `EvaluateSteering` takes `input` (the request).
		// Should we steer based on *Input* (Pre-check) or *Output* (Post-check)?
		// The prompt says: "Query: correction(_, Hint)? (where _ matches the current request ID)."
		// And the example Datalog: `correction(Req, ...) :- payload.sql(Req, SQL) ...`
		// The example Datalog matches on `payload.sql`, which sounds like *Input* if it's a SQL generation tool, or *Output* if it generated SQL.
		// "Retry Logic: If SQL contains DROP, ask to fix". This sounds like we generated a SQL (Output) and we are checking it.
		// So we steer based on the *Result* of the action.
		// So `EvaluateSteering` should be called on `resEnv` (the result of the action).
		//
		// Let's re-read the prompt logic for `EvaluateSteering`:
		// "Query: correction(_, Hint)? (where _ matches the current request ID)."
		// In `Authorize` (Pre-check), we use `Req`.
		// In `Validate` (Post-check), we use `Output`.
		//
		// The `EvaluateSteering` signature I added takes `input core.Envelope`.
		// In `RunLoop`, after `action.Execute`, we have `resEnv`.
		// So we should pass `resEnv` to `EvaluateSteering`.
		// And we should probably use "Output" or keep "Req" depending on convention.
		// `toMangleFacts` in `policy.go` uses "Req" hardcoded in my implementation of `EvaluateSteering`.
		// I should probably allow customizing the ID or just stick to "Req" and map the envelope payload to it.
		//
		// So, in `RunLoop`:
		// 1. Execute action -> resEnv
		// 2. Call `c.engine.EvaluateSteering(ctx, action.Metadata(), resEnv)`
		// 3. Update `resEnv.Metadata` with the decision and keys.
		// 4. Then proceed with the switch case.

		// Execute the action
		resEnv, err := action.Execute(ctx, env)
		if err != nil {
			return core.Envelope{}, err
		}

		// Evaluate Steering based on the RESULT
		decision, meta := c.engine.EvaluateSteering(ctx, action.Metadata(), resEnv)

		// Merge steering metadata into response envelope
		if resEnv.Metadata == nil {
			resEnv.Metadata = make(map[string]string)
		}
		resEnv.Metadata[core.KeyDecision] = decision
		for k, v := range meta {
			resEnv.Metadata[k] = v
		}

		// Handle Decision
		switch decision {
		case core.DecisionAllow:
			return resEnv, nil
		case core.DecisionDeny:
			return core.Envelope{}, fmt.Errorf("action denied by steering policy")
		case core.DecisionRetry:
			if hint, ok := meta[core.KeyFeedback]; ok {
				feedback = append(feedback, hint)
				c.logger.Info("steering: retry", "feedback", hint)
				continue // Loop same action
			}
			// If no feedback, treat as allow? Or error?
			// Default to allow if something is wrong
			return resEnv, nil
		case core.DecisionRoute:
			if next, ok := meta[core.KeyNextStep]; ok {
				currentActionName = next
				payload = resEnv.Payload // Pipe output to input
				feedback = nil // Reset feedback
				c.logger.Info("steering: route", "next_step", next)
				continue
			}
			return resEnv, nil
		default:
			return resEnv, nil
		}
	}

	return core.Envelope{}, fmt.Errorf("steering loop exceeded max steps (%d)", maxSteps)
}
