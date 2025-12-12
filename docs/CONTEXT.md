---
context_type: full_source_dump
project: manglekit
language: go
last_updated: 2025-12-11
scan_mode: exhaustive
---

## 1. THE COMPLETE FILE MAP

```
.
├── AGENTS.md
├── CONTRIBUTING.md
├── LICENSE
├── Makefile
├── README.md
├── adapters/                  # Integration layer for external systems
│   ├── ai/                    # LLM Providers (Genkit, Gemini)
│   │   ├── adapter.go
│   │   ├── adapter_test.go
│   │   ├── gemini.go
│   │   ├── genkit.go
│   │   └── utils.go
│   ├── extractor/             # Structured Data Extraction
│   │   ├── adapter.go
│   │   └── adapter_test.go
│   ├── func/                  # Generic Function Wrapper
│   │   └── wrapper.go
│   ├── knowledge/             # RDF/Graph Knowledge Loaders
│   │   ├── nquads.go
│   │   ├── nquads_test.go
│   │   ├── ntriples.go
│   │   ├── rdf.go
│   │   └── rdf_stub.go
│   ├── logger/                # Logging Adapters (Zap)
│   │   └── zap_adapter.go
│   ├── mcp/                   # Model Context Protocol
│   │   ├── action.go
│   │   ├── loader.go
│   │   └── loader_test.go
│   ├── resilience/            # Circuit Breakers
│   │   ├── circuit_breaker.go
│   │   └── circuit_breaker_test.go
│   └── vector/                # Vector Store Retrievers
│       ├── genkit_retriever.go
│       ├── retriever_adapter.go
│       └── retriever_adapter_test.go
├── cmd/
│   └── mkit/                  # CLI Tool
│       ├── commands/
│       │   ├── eval/
│       │   ├── gen/
│       │   ├── inspect/
│       │   └── kg/
│       └── main.go
├── config/                    # Configuration Loading
│   ├── loader.go
│   ├── loader_test.go
│   └── schema.go
├── core/                      # Core Interfaces & Types (Dependency-free)
│   ├── action.go              # The 'Action' Interface
│   ├── constants.go
│   ├── context_lineage.go
│   ├── envelope.go            # The 'Envelope' Data Carrier
│   ├── errors.go
│   ├── logger.go
│   ├── logger_test.go
│   ├── memory.go
│   ├── state.go
│   ├── tracer.go
│   └── types.go
├── docs/                      # Documentation
│   ├── ADR.md
│   ├── ARCHITECTURE_RULES.md
│   ├── CONFIG.md
│   ├── CONTEXT.md             # THIS FILE
│   ├── CSD.md
│   ├── HLD.md
│   ├── LLD.md
│   ├── LOGGING.md
│   ├── Mangle-quickstart.md
│   ├── TRACING.md
│   └── reports/
│       ├── CONFIG_LOADER_MIGRATION_SUMMARY.md
│       ├── IMPLEMENTATION_CHECKLIST.md
│       ├── IMPLEMENTATION_SUMMARY.md
│       ├── POLICY_COPILOT_GUIDE.md
│       ├── POLICY_COPILOT_IMPLEMENTATION.md
│       ├── code_audit_20250604.md
│       └── feature_audit_v1.0.0.md
├── examples/                  # Runnable Examples
│   ├── AGENTS.md
│   ├── README.md
│   ├── chat_chit/
│   ├── context_aware_rag/
│   ├── dynamic_pricing/
│   ├── extractor_demo/
│   ├── fintech_approval/
│   ├── kernel_knowledge_demo/
│   ├── lineage_demo/
│   ├── openrouter_demo/     # NEW: OpenRouter/LocalAI Example
│   │   ├── README.md
│   │   ├── blueprint.dl
│   │   └── main.go
│   ├── planner/
│   ├── policy-copilot/
│   ├── rag_flow/
│   ├── semantic_feedback/
│   ├── sre_guardrail/
│   ├── steering/
│   └── taint_demo/
├── go.mod
├── go.sum
├── internal/                  # Internal Logic (Private)
│   ├── engine/                # The Policy Engine & Runtime
│   │   ├── dual_mode_test.go
│   │   ├── evaluator.go
│   │   ├── evaluator_test.go
│   │   ├── flattener.go
│   │   ├── flattener_test.go
│   │   ├── memory/
│   │   │   ├── volatile.go
│   │   │   └── volatile_test.go
│   │   ├── reflection.go
│   │   ├── reflection_test.go
│   │   ├── resources/
│   │   │   ├── embed.go
│   │   │   ├── planner.dl
│   │   │   └── std.dl
│   │   ├── runtime.go
│   │   ├── solver.go
│   │   └── solver_test.go
│   ├── logger/
│   │   └── std.go
│   ├── statehelper/
│   │   └── statehelper.go
│   ├── supervisor/            # The 'Guard' (Decorator)
│   │   ├── supervisor.go
│   │   ├── supervisor_test.go
│   │   └── trace_test.go
│   ├── telemetry/
│   │   └── otel.go
│   ├── testproviders/
│   ├── tools/
│   │   └── rulegen/
│   └── util/
│       └── schema/
│           ├── generator.go
│           ├── schema_test.go
│           └── validator.go
├── mangle.yaml
├── manglekit.go               # Facade API
└── sdk/                       # Public SDK
    ├── action_test.go
    ├── client.go              # Main Entry Point
    ├── context.go
    ├── executor.go
    ├── generics.go
    ├── helpers.go
    ├── integration_test.go
    ├── interfaces.go
    ├── loader_ext.go
    ├── loop.go                # Execution Loop (SSM)
    ├── options.go
    ├── options_ext.go
    ├── planner.go
    ├── policy_generator.go    # Policy Copilot
    ├── policy_generator_test.go
    ├── reflection_test.go
    ├── tracing.go
    └── types.go
```

## 2. COMPONENT ANALYSIS

### A. Engine (`internal/engine`)
The heart of the system. It hosts the Google Mangle Datalog runtime and provides the decision-making logic.
*   **Key Structs:** `PolicyEngine` (Orchestrator), `MangleRuntime` (Datalog Wrapper).
*   **Responsibilities:**
    *   Loading and parsing Datalog policies (`.dl`).
    *   Maintaining the knowledge base (Facts).
    *   **Pre-Check (Authorization):** `deny(Req)`.
    *   **Post-Check (Validation):** `deny(Output)`.
    *   **Steering:** `correction(Hint)` / `next_step(Target)`.
    *   **Reflection:** Converting Go structs/JSON into Datalog facts (`ToFacts`, `Flatten`).

### B. SDK (`sdk/`)
The user-facing API and execution orchestration layer.
*   **Key Structs:** `Client` (God Object), `Runnable` (Typed Action).
*   **Responsibilities:**
    *   Initializing the system (`NewClient`).
    *   Managing the Action Registry (`RegisterAction`).
    *   **Execution Loop (`ExecuteByName`):** Implements the "Semantic State Machine" (Try -> Fail -> Feedback -> Retry).
    *   History Management (Persistence).
    *   **Policy Generator:** LLM-based policy authoring.

### C. Supervisor (`internal/supervisor`)
The enforcement layer. It implements the "Guarded Action" pattern via the Decorator pattern.
*   **Key Structs:** `SupervisedAction`.
*   **Responsibilities:**
    *   **Tracing:** Starting OpenTelemetry spans.
    *   **Lifecycle Management:** `Authorize` -> `Execute` -> `Validate`.
    *   **Taint Propagation:** Merging security labels from Input to Output.
    *   **Fail-Mode Handling:** Blocking or Logging (Open/Closed) based on configuration.

### D. Adapters (`adapters/`)
The integration layer for external capabilities.
*   **Key Structs:** `genkitAdapter`, `MCPAction`, `CircuitBreaker`.
*   **Responsibilities:**
    *   Bridging `core.Action` to external libraries (Firebase Genkit, MCP, etc.).
    *   Standardizing inputs/outputs to `core.Envelope`.

### E. Config (`config/`)
The static configuration loader.
*   **Key Structs:** `Config`, `PolicyConfig`, `MCPServerConfig`.
*   **Responsibilities:**
    *   Loading YAML configuration (`mangle.yaml`).
    *   Environment variable expansion.

## 3. CRITICAL PATH & DATA

### Execution Flow: `Client.ExecuteByName`

1.  **Entry:** User calls `client.ExecuteByName(ctx, "my-action", payload)`.
2.  **Loop (`sdk/loop.go`):** The `runLoopInternal` starts a retry/route loop (default 10 steps).
3.  **Step Execution:**
    *   **Context Injection:** Feedback, History, and Metadata are injected into a new `Envelope`.
    *   **Supervisor (`internal/supervisor.go`):**
        *   **Trace:** Span started.
        *   **Authorize:** Engine checks `deny(Req)`. If blocked -> `ErrAlignment`.
        *   **Execute:** Inner Action runs (e.g., Genkit LLM).
        *   **Validate:** Engine checks `deny(Output)`. If blocked -> `ErrAlignment`.
        *   **Steering:** Engine checks `correction`/`next_step`.
4.  **Decision Handling:**
    *   **RETRY:** If `ErrAlignment` or Steering `RETRY`: Increment counter, inject feedback, backoff, loop.
    *   **ROUTE:** If Steering `ROUTE`: Switch target action, loop.
    *   **ALLOW:** Return result to user.

### Data Structures

*   **`core.Envelope`:** The universal container.
    *   `ID` (UUID): Unique trace ID.
    *   `Payload` (Any): The actual data (Struct or JSON).
    *   `Metadata` (Map): Control plane signals (Feedback, Decision).
    *   `SecurityLabels` (Slice): Taint tracking tags (e.g., "secret").
*   **Mangle Facts:** Datalog representation of data.
    *   `field(ID, "name", "value")` (Reflection).
    *   `action("name")`, `has_input("name", "type")` (Metadata).

## 4. SOURCE CODE DUMP

### `manglekit.go`
```go
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
func WithPolicyPath(path string) ClientOption        { return sdk.WithBlueprintPath(path) }
func WithFailMode(mode string) ClientOption          { return sdk.WithFailMode(mode) }
func WithLogger(l core.Logger) ClientOption          { return sdk.WithLogger(l) }
func WithMemory(store core.MemoryStore) ClientOption { return sdk.WithMemory(store) }

func WithSessionID(id string) ExecuteOption        { return sdk.WithSessionID(id) }
func WithTransientMemory() ExecuteOption           { return sdk.WithTransientMemory() }
func WithMetadata(key, value string) ExecuteOption { return sdk.WithMetadata(key, value) }
```

### `sdk/client.go`
```go
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
	// blueprintPath stores the path loaded at startup (for debugging/reloading).
	blueprintPath string
	// shutdownFunc is a cleanup function to stop exporters/tracers.
	shutdownFunc func(context.Context) error
	// llm is the plugged-in text generation backend.
	llm TextGenerator
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
		if err := c.engine.LoadPolicy(string(content)); err != nil {
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
	log := logger.NewDefault()

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
		content, err := os.ReadFile(cfg.Policy.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to read policy file %q: %w", cfg.Policy.Path, err)
		}
		if err := c.engine.LoadPolicy(string(content)); err != nil {
			return nil, fmt.Errorf("failed to load policy from %q: %w", cfg.Policy.Path, err)
		}
	}

	// Load knowledge from the configured path
	if cfg != nil && cfg.Knowledge.Path != "" {
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
			loader := mcpAdapter.NewLoader(mcpCfg).WithLogger(c.logger)
			actions, err := loader.Load(ctx)
			if err != nil {
				// Because loader.Load now handles Soft Failure internally (returning nil error),
				// any error returned here implies FailOnStartup=true or a critical loader error.
				return nil, fmt.Errorf("critical tool '%s' failed to load: %w", mcpCfg.Name, err)
			}

			for _, action := range actions {
				// Supervise It
				safeAction := c.Supervise(action)
				// Register It
				c.RegisterAction(safeAction.Metadata().Name, safeAction)
				c.logger.Info("Discovered MCP Tool", "name", safeAction.Metadata().Name)
			}
		}
	}

	return c, nil
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

func (c *Client) Engine() *engine.PolicyEngine {
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
		if err := c.engine.RegisterActionMetadata(action.Metadata()); err != nil {
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
```

### `sdk/loop.go`
```go
package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/core"
	engine_memory "github.com/duynguyendang/manglekit/internal/engine/memory"
)

// Define constants or config struct
const (
	DefaultMaxSteps   = 10
	DefaultMaxRetries = 3
	BackoffBase       = 100 * time.Millisecond
)

// ExecuteByName executes a registered action by its name, handling the Semantic State Machine loop.
func (c *Client) ExecuteByName(ctx context.Context, actionName string, input any, opts ...ExecuteOption) (core.Envelope, error) {
	params := ExecutionParams{
		MemoryMode: core.MemoryModeNone,
	}
	for _, opt := range opts {
		opt(&params)
	}
	return c.runLoopInternal(ctx, actionName, input, params)
}

func (c *Client) runLoopInternal(ctx context.Context, startAction string, payload any, params ExecutionParams) (core.Envelope, error) {
	if c.logger != nil {
		ctx = core.ContextWithLogger(ctx, c.logger)
	}

	// 1. Determine Store Strategy
	switch params.MemoryMode {
	case core.MemoryModePersist:
		params.Store = c.memory
		if params.SessionID != "" {
			var err error
			// Load initial history
			params.CurrentHistory, err = params.Store.Read(ctx, params.SessionID)
			if err != nil && c.logger != nil {
				c.logger.Warn("RunLoop failed to hydrate history", "error", err)
			}
		}
	case core.MemoryModeTransient:
		params.Store = &engine_memory.VolatileStore{}
	default:
		params.Store = &core.NoOpStore{}
	}

	currentAction := startAction
	currentPayload := payload

	for step := 0; step < DefaultMaxSteps; step++ {
		// Check context before starting new step
		if err := ctx.Err(); err != nil {
			return core.Envelope{}, err
		}

		if c.logger != nil {
			c.logger.Info("RunLoop step", "step", step, "action", currentAction)
		}

		result, err := c.ExecuteSingleStep(ctx, currentAction, currentPayload, &params)
		if err != nil {
			return core.Envelope{}, err
		}

		decision := result.Metadata[core.KeyDecision]
		if decision == core.DecisionRoute {
			// Update flow for next loop
			next := result.Metadata[core.KeyNextStep]
			if next == "" {
				return core.Envelope{}, fmt.Errorf("route decision missing next_step")
			}

			// Validate/Log Payload Handover
			if c.logger != nil {
				c.logger.Info("RunLoop: Routing to next action", "from", currentAction, "to", next, "payload_type", fmt.Sprintf("%T", result.Payload))
			}

			currentAction = next
			currentPayload = result.Payload
			continue
		}

		if decision == core.DecisionRetry {
			// Params internal state (RetryCount, Feedback) already updated by ExecuteSingleStep
			// Validation (MaxRetries) and Backoff (Sleep) also handled by ExecuteSingleStep
			// Just loop again with same action/payload
			continue
		}

		// ALLOW or empty -> Done
		return result, nil
	}
	return core.Envelope{}, fmt.Errorf("max steps exceeded")
}

// ExecuteSingleStep runs one step of the action and returns the decision.
// It handles: Action Execution, History Persistence, Blueprint Alignment Backoff, and Steering Logic (Retry/Route updates).
func (c *Client) ExecuteSingleStep(ctx context.Context, actionName string, payload any, params *ExecutionParams) (core.Envelope, error) {
	// 1. Resolve Action
	action, ok := c.registry[actionName]
	if !ok {
		return core.Envelope{}, fmt.Errorf("action not found: %s", actionName)
	}

	env := core.NewEnvelope(payload)
	env.ContentType = action.Metadata().InputContentType

	// --- Phase 1: Context Injection ---
	// 1.1 Inject Feedback (History & Last Hint)
	if len(params.FeedbackHistory) > 0 {
		env.Metadata[core.KeyPrevFeedback] = strings.Join(params.FeedbackHistory, "; ")
	}
	if params.LastFeedback != "" {
		env.SetFeedback(params.LastFeedback)
		env.Metadata["mangle_feedback"] = params.LastFeedback
	}

	// 1.2 Inject Chat History
	if len(params.CurrentHistory) > 0 && params.MemoryMode != core.MemoryModeNone {
		env.SetHistory(params.CurrentHistory)
	}

	// 1.3 Inject Explicit Metadata
	if params.Metadata != nil {
		for k, v := range params.Metadata {
			env.Metadata[k] = v
		}
	}

	// 1.4 Inject Context Facts (e.g. User Role, Budget from sdk.WithFact)
	// This ensures facts propagate to the Engine for Blueprint Checks.
	if facts := ContextFacts(ctx); facts != nil {
		for k, v := range facts {
			env.Metadata[k] = v
		}
	}

	// --- Phase 2: Blueprint Check (Pre-check) & Phase 3: Execution (Intuition) ---
	// The Action.Execute wrapper handles the Pre-check (Blueprint) and the actual Genkit Execution.
	result, err := action.Execute(ctx, env)
	if err != nil {
		var pve *core.AlignmentError
		if errors.As(err, &pve) {
			if params.RetryCount >= DefaultMaxRetries {
				return core.Envelope{}, fmt.Errorf("max retries exceeded: %w", err)
			}
			params.RetryCount++
			params.LastFeedback = pve.Message

			if c.logger != nil {
				c.logger.Warn("RunLoop: Blueprint Alignment Issue", "feedback", params.LastFeedback, "attempt", params.RetryCount)
			}

			// Context-aware Backoff
			if err := c.backoff(ctx, params.RetryCount); err != nil {
				return core.Envelope{}, err
			}

			// Return a mock result to signal RETRY to caller
			res := core.NewEnvelope(payload)
			res.Metadata[core.KeyDecision] = core.DecisionRetry
			return res, nil
		}
		return core.Envelope{}, err
	}

	params.LastFeedback = ""

	// 4. Update History
	if params.MemoryMode != core.MemoryModeNone {
		userContent := safelyStringify(payload)
		assistContent := safelyStringify(result.Payload)

		newExchange := []core.ChatMessage{
			{Role: "user", Content: userContent},
			{Role: "assistant", Content: assistContent},
		}
		params.CurrentHistory = append(params.CurrentHistory, newExchange...)
	}

	// 5. Persist
	if params.Store != nil && params.SessionID != "" && params.MemoryMode == core.MemoryModePersist {
		if err := params.Store.Write(ctx, params.SessionID, params.CurrentHistory); err != nil && c.logger != nil {
			c.logger.Warn("RunLoop failed to persist history", "error", err)
		}
	}

	// --- Phase 4: Evaluation & Correction (Post-check) ---
	// The Engine evaluates the result against the Blueprint and decides next steps (Retry/Route/Allow).
	decision := result.Metadata[core.KeyDecision]
	if c.logger != nil {
		c.logger.Debug("RunLoop decision", "decision", decision, "action", actionName)
	}

	switch decision {
	case core.DecisionRetry:
		if params.RetryCount >= DefaultMaxRetries {
			return core.Envelope{}, fmt.Errorf("max retries exceeded for action %s", actionName)
		}
		params.RetryCount++
		hint := result.GetFeedback()
		params.LastFeedback = hint
		params.FeedbackHistory = append(params.FeedbackHistory, hint)

		if c.logger != nil {
			c.logger.Warn("RunLoop: RETRY triggered", "feedback", hint)
		}

		// Context-aware Backoff
		if err := c.backoff(ctx, params.RetryCount); err != nil {
			return core.Envelope{}, err
		}
		// Return result so caller loops
		return result, nil

	case core.DecisionRoute:
		// Reset retry count for new action
		params.RetryCount = 0
		params.FeedbackHistory = nil

		if c.logger != nil {
			c.logger.Info("RunLoop: Feedback history cleared for new action route")
		}
		// Return result so caller loops
		return result, nil

	case core.DecisionAllow, "":
		return result, nil

	case core.DecisionDeny:
		reason := result.Metadata["reason"]
		if reason == "" {
			reason = result.Metadata["violation_msg"]
		}
		if reason == "" {
			reason = "blueprint violation"
		}
		return core.Envelope{}, fmt.Errorf("action denied by blueprint: %s", reason)
	}

	// Should not reach here for standard decisions
	return result, nil
}

// Helper for better logging
func safelyStringify(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	// Try JSON first for structured data
	b, err := json.Marshal(v)
	if err == nil {
		return string(b)
	}
	// Fallback
	return fmt.Sprintf("%v", v)
}

// backoff handles the sleep and context cancellation check
func (c *Client) backoff(ctx context.Context, retryCount int) error {
	sleepDuration := time.Duration(retryCount) * BackoffBase
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(sleepDuration):
		return nil
	}
}
```

### `internal/supervisor/supervisor.go`
```go
package supervisor

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SupervisedAction is a decorator that wraps any `core.Action` to enforce governance blueprints.
// It implements the standard "Trace -> Authorize -> Execute -> Validate" lifecycle.
//
// Lifecycle:
//  1. Trace: Starts an OpenTelemetry span for the operation.
//  2. Authorize: Checks Pre-Check blueprints (e.g., "deny(Req)").
//  3. Execute: Runs the inner action (e.g., calls the LLM).
//  4. Validate: Checks Post-Check blueprints (e.g., "deny(Output)").
//  5. Steering: Evaluates steering blueprints for routing or correction.
type SupervisedAction struct {
	inner       core.Action
	engine      *engine.PolicyEngine
	tracer      core.Tracer
	failureMode string
}

// NewSupervisedAction creates a new SupervisedAction with default settings (no tracing).
//
// Parameters:
//   - action: The inner action to supervise.
//   - eng: The policy engine to use for governance.
//   - failureMode: The resilience strategy ("open" or "closed").
//
// Returns:
//   - A new SupervisedAction instance.
func NewSupervisedAction(action core.Action, eng *engine.PolicyEngine, failureMode string) *SupervisedAction {
	return &SupervisedAction{
		inner:       action,
		engine:      eng,
		tracer:      &core.NopTracer{},
		failureMode: failureMode,
	}
}

// NewSupervisedActionWithTracer creates a new SupervisedAction with tracing enabled.
//
// Parameters:
//   - action: The inner action to supervise.
//   - eng: The policy engine.
//   - tracer: The tracer implementation.
//   - failureMode: "open" (log only on system error) or "closed" (block on system error).
//
// Returns:
//   - A new SupervisedAction instance.
func NewSupervisedActionWithTracer(action core.Action, eng *engine.PolicyEngine, tracer core.Tracer, failureMode string) *SupervisedAction {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &SupervisedAction{
		inner:       action,
		engine:      eng,
		tracer:      tracer,
		failureMode: failureMode,
	}
}

// Execute runs the supervised action, orchestrating the full governance lifecycle.
//
// It performs the following steps:
//  1. Starts a span.
//  2. Injects the logger into the context.
//  3. Runs Authorize(). If it fails, execution halts (unless Fail-Open).
//  4. Runs the inner Action.Execute().
//  5. Propagates taint labels from input to output.
//  6. Runs Validate(). If it fails, the result is blocked.
//  7. Runs EvaluateSteering() to determine next steps (Retry/Route).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The data envelope.
//
// Returns:
//   - The result envelope (possibly modified by blueprint), or an error.
func (g *SupervisedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Auto-Tracing (Phase 5)
	// We use the global OTel tracer "manglekit" to create spans automatically.
	// This supersedes the legacy g.tracer usage for the main span,
	// ensuring consistent observability without user configuration.
	tracer := otel.Tracer("manglekit")
	meta := g.inner.Metadata()

	ctx, span := tracer.Start(ctx, fmt.Sprintf("Action.%s", meta.Name),
		trace.WithAttributes(
			attribute.String("mangle.action_name", meta.Name),
			attribute.String("mangle.action_type", string(meta.Type)),
			attribute.String("mangle.input_id", input.ID.String()),
		),
	)
	defer span.End()

	result, err := g.executeInternal(ctx, input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		// Distinguish between Blueprint DENIAL and System ERROR
		if g.isAlignmentIssue(err) {
			span.SetAttributes(attribute.String("mangle.outcome", "DENIED"))
		} else {
			span.SetAttributes(attribute.String("mangle.outcome", "ERROR"))
		}
		return core.Envelope{}, err
	}

	span.SetAttributes(
		attribute.String("mangle.outcome", "ALLOWED"),
		attribute.String("mangle.output_id", result.ID.String()),
	)
	return result, nil
}

// isAlignmentIssue checks if the error is a wrapped alignment check violation
func (g *SupervisedAction) isAlignmentIssue(err error) bool {
	return core.IsAlignmentError(err)
}

// Metadata delegates to the inner action's Metadata method.
// This allows the SupervisedAction to transparently represent the underlying capability.
func (g *SupervisedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
}

// shouldBlock determines if the action should be blocked based on the error and failure mode.
func (g *SupervisedAction) shouldBlock(err error) bool {
	if err == nil {
		return false
	}
	// Always block on explicit alignment issues
	if core.IsAlignmentError(err) {
		return true
	}
	// If mode is "open" (Fail-Open), allow execution (return false)
	// Otherwise (default/closed), block execution (return true)
	if g.failureMode == "open" {
		return false
	}
	return true
}

// executeInternal contains the actual execution logic.
// It receives the context with the active span so child spans can be created.
// The logger is injected into the context here, ensuring all downstream code
// can access it via core.LoggerFromContext(ctx).
func (g *SupervisedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Inject the logger into the context for downstream access
	ctx = core.ContextWithLogger(ctx, g.engine.Logger())

	logger := core.LoggerFromContext(ctx)
	meta := g.inner.Metadata()
	logger.Info("Action started",
		"action", meta.Name,
		"input_id", input.ID.String(),
	)

	// 1. Ingestion: Link Input to Parent (Tracing only)
	if parentID, ok := core.GetParentID(ctx); ok {
		g.engine.RecordLineage(ctx, input.ID.String(), parentID)
	}

	// 2. Pre-Check: Authorization
	if err := g.engine.Authorize(ctx, g.inner.Metadata(), input); err != nil {
		if g.shouldBlock(err) {
			logger.Warn("authorization failed",
				core.AttrActionName, meta.Name,
				"error", err.Error(),
			)
			return core.Envelope{}, fmt.Errorf("authorization failed: %w", err)
		}
		// Fail-Open
		logger.Warn("engine failed but Fail-Open active. Proceeding.", "error", err)
	}

	// 3. Context Propagation: Pass the Gene
	// Propagate the current input ID as the new parent for the inner action
	childCtx := core.WithParentID(ctx, input.ID.String())

	// 4. Execution: Run inner action
	result, err := g.inner.Execute(childCtx, input)
	if err != nil {
		logger.Error("action execution failed",
			core.AttrActionName, meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("action execution failed: %w", err)
	}

	// 5. Propagation: Output inherits Input's security labels
	if len(input.SecurityLabels) > 0 {
		result.MergeLabels(input.SecurityLabels)
	}

	// 6. Linking: Link Output to Input (Tracing only)
	g.engine.RecordLineage(ctx, result.ID.String(), input.ID.String())
	if result.Metadata == nil {
		result.Metadata = make(map[string]string)
	}
	result.Metadata["derived_from"] = input.ID.String()

	// 7. Post-Check: Validation
	validatedResult, err := g.engine.Validate(ctx, g.inner.Metadata(), result)
	if err != nil {
		if g.shouldBlock(err) {
			logger.Warn("validation failed",
				"action", meta.Name,
				"error", err.Error(),
			)
			return core.Envelope{}, fmt.Errorf("validation failed: %w", err)
		}
		// Fail-Open: use result as validatedResult
		logger.Warn("engine validation failed but Fail-Open active. Proceeding.", "error", err)
		validatedResult = result
	}

	// 8. Steering: Evaluate next steps (Correction/Routing)
	decision, steeringMeta, err := g.engine.EvaluateSteering(ctx, validatedResult)
	if err != nil {
		logger.Warn("steering evaluation failed",
			"action", meta.Name,
			"error", err.Error(),
		)
		return core.Envelope{}, fmt.Errorf("steering evaluation failed: %w", err)
	}

	// Stamp metadata
	if validatedResult.Metadata == nil {
		validatedResult.Metadata = make(map[string]string)
	}
	validatedResult.Metadata[core.KeyDecision] = decision
	for k, v := range steeringMeta {
		validatedResult.Metadata[k] = v
	}

	logger.Info("Action completed",
		"action", meta.Name,
		"result", "success", // Simplified as per doc
	)

	return validatedResult, nil
}
```

### `internal/engine/runtime.go`
```go
package engine

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/google/mangle/analysis"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/engine"
	"github.com/google/mangle/factstore"
	"github.com/google/mangle/parse"
)

// MangleRuntime encapsulates the Google Mangle Datalog engine.
type MangleRuntime struct {
	mu sync.RWMutex

	// State fields (protected by mu)
	programInfo   *analysis.ProgramInfo
	strata        []analysis.Nodeset
	predToStratum map[ast.PredicateSym]int
	baseFactStore factstore.SimpleInMemoryStore
	ruleUnits     []parse.SourceUnit
	ready         bool // Flag to indicate if the runtime is initialized
}

// NewMangleRuntime initializes a new, empty MangleRuntime.
func NewMangleRuntime() *MangleRuntime {
	return &MangleRuntime{
		predToStratum: make(map[ast.PredicateSym]int),
		baseFactStore: factstore.NewSimpleInMemoryStore(),
		ready:         false,
	}
}

// Load loads Datalog rules and facts from the specified path.
// CRITICAL CHANGE: This REPLACES the current program state.
func (r *MangleRuntime) Load(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// 1. Resolve files (I/O) - No lock needed yet
	files, err := resolveFiles(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	var ruleFiles, factFiles []string
	for _, file := range files {
		switch {
		case isRuleFile(file):
			ruleFiles = append(ruleFiles, file)
		case isFactFile(file):
			factFiles = append(factFiles, file)
		}
	}

	if len(ruleFiles) == 0 && len(factFiles) == 0 {
		return fmt.Errorf("no .dlog or fact files found in %s", path)
	}

	// 2. Parse and Build State (Local Variables)
	// We build everything locally to ensure atomicity. If parsing fails,
	// the existing runtime state remains untouched.
	var newRuleUnits []parse.SourceUnit
	edbDeclarations := make(map[ast.PredicateSym]ast.Decl)

	// Parse Rules
	for _, ruleFile := range ruleFiles {
		unit, err := parseRuleFile(ruleFile)
		if err != nil {
			return fmt.Errorf("failed to parse rule file %s: %w", ruleFile, err)
		}
		newRuleUnits = append(newRuleUnits, unit)
	}

	// Parse Facts and build Base Store
	newBaseStore := factstore.NewSimpleInMemoryStore()
	for _, factFile := range factFiles {
		unit, err := parseRuleFile(factFile)
		if err != nil {
			return fmt.Errorf("failed to parse fact file %s: %w", factFile, err)
		}
		for _, clause := range unit.Clauses {
			if len(clause.Premises) == 0 {
				newBaseStore.Add(clause.Head)
			}
		}
	}

	// 3. Analyze and Stratify (CPU Intensive)
	programInfo, err := analysis.Analyze(newRuleUnits, edbDeclarations)
	if err != nil {
		return fmt.Errorf("failed to analyze program: %w", err)
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return fmt.Errorf("failed to stratify program: %w", err)
	}

	// 4. Initial Evaluation (Validation)
	// We run this on the local store to ensure the program doesn't crash on init.
	if _, err := engine.EvalStratifiedProgramWithStats(programInfo, strata, predToStratum, newBaseStore); err != nil {
		return fmt.Errorf("failed to evaluate base program: %w", err)
	}

	// 5. Atomic Swap (Critical Section)
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ruleUnits = newRuleUnits
	r.baseFactStore = newBaseStore
	r.programInfo = programInfo
	r.strata = strata
	r.predToStratum = predToStratum
	r.ready = true

	return nil
}

// LoadFromSource parses and loads a full Datalog program from a string.
// REPLACES current state.
func (r *MangleRuntime) LoadFromSource(source string) error {
	if source == "" {
		return fmt.Errorf("source cannot be empty")
	}

	cleaned := cleanSource(source)
	unit, err := parse.Unit(strings.NewReader(cleaned))
	if err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	// Local state build
	newRuleUnits := []parse.SourceUnit{unit}
	edbDeclarations := make(map[ast.PredicateSym]ast.Decl)

	programInfo, err := analysis.Analyze(newRuleUnits, edbDeclarations)
	if err != nil {
		return fmt.Errorf("failed to analyze program: %w", err)
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return fmt.Errorf("failed to stratify program: %w", err)
	}

	// Create new store (resetting old facts if this is a full reload)
	newBaseStore := factstore.NewSimpleInMemoryStore()

	// Atomic Swap
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ruleUnits = newRuleUnits
	r.baseFactStore = newBaseStore
	r.programInfo = programInfo
	r.strata = strata
	r.predToStratum = predToStratum
	r.ready = true

	// Evaluate with empty base store
	if err := r.evaluate(r.baseFactStore); err != nil {
		return fmt.Errorf("failed to evaluate program: %w", err)
	}

	return nil
}

// LoadFacts injects a list of raw Datalog fact strings into the runtime's base knowledge.
func (r *MangleRuntime) LoadFacts(facts []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, factStr := range facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			return fmt.Errorf("failed to parse fact '%s': %w", factStr, err)
		}
		r.baseFactStore.Add(atom)
	}

	if r.ready {
		if err := r.evaluate(r.baseFactStore); err != nil {
			return fmt.Errorf("failed to evaluate program with new facts: %w", err)
		}
	}
	return nil
}

// LoadFromString parses and loads a full Datalog program provided as a string.
// IMPORTANT: This REPLACES the current program state.
func (r *MangleRuntime) LoadFromString(rule string) error {
	return r.LoadFromSource(rule)
}

// AddPolicy adds new rules to the existing program state (Incremental Loading).
func (r *MangleRuntime) AddPolicy(source string) error {
	if source == "" {
		return nil
	}

	cleaned := cleanSource(source)
	unit, err := parse.Unit(strings.NewReader(cleaned))
	if err != nil {
		return fmt.Errorf("failed to parse source: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Append to existing rules
	newRuleUnits := make([]parse.SourceUnit, len(r.ruleUnits)+1)
	copy(newRuleUnits, r.ruleUnits)
	newRuleUnits[len(r.ruleUnits)] = unit

	// Re-Analyze
	edbDeclarations := make(map[ast.PredicateSym]ast.Decl)
	programInfo, err := analysis.Analyze(newRuleUnits, edbDeclarations)
	if err != nil {
		return fmt.Errorf("failed to analyze combined program: %w", err)
	}

	strata, predToStratum, err := analysis.Stratify(analysis.Program{
		EdbPredicates: programInfo.EdbPredicates,
		IdbPredicates: programInfo.IdbPredicates,
		Rules:         programInfo.Rules,
	})
	if err != nil {
		return fmt.Errorf("failed to stratify combined program: %w", err)
	}

	// Update State
	r.ruleUnits = newRuleUnits
	r.programInfo = programInfo
	r.strata = strata
	r.predToStratum = predToStratum
	r.ready = true

	// Re-evaluate base facts with new rules
	if err := r.evaluate(r.baseFactStore); err != nil {
		return fmt.Errorf("failed to evaluate combined program: %w", err)
	}

	return nil
}

// ExecuteQuery runs a boolean Datalog query.
func (r *MangleRuntime) ExecuteQuery(facts []ast.Atom, queryStr string) (bool, error) {
	r.mu.RLock()
	// Check readiness under lock
	if !r.ready {
		r.mu.RUnlock()
		return false, fmt.Errorf("runtime not initialized")
	}

	// 1. Snapshot the state needed for evaluation
	// We copy the base store to avoid contaminating the global state with request-scoped facts.
	// Note: This is O(N) where N is base facts.
	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)

	// Capture pointers to analysis structures (they are read-only during eval)
	pInfo := r.programInfo
	strata := r.strata
	pStratum := r.predToStratum

	r.mu.RUnlock() // Release lock early to allow concurrent evaluations

	// 2. Add temporary facts
	for _, fact := range facts {
		workingStore.Add(fact)
	}

	// 3. Evaluate (Expensive part, runs without blocking main lock)
	if _, err := engine.EvalStratifiedProgramWithStats(pInfo, strata, pStratum, workingStore); err != nil {
		return false, fmt.Errorf("evaluation failed: %w", err)
	}

	// 4. Check result
	queryAtom, err := parse.Atom(queryStr)
	if err != nil {
		return false, fmt.Errorf("failed to parse query '%s': %w", queryStr, err)
	}

	return workingStore.Contains(queryAtom), nil
}

// QueryWithSolutions executes a query and invokes callback for solutions.
func (r *MangleRuntime) QueryWithSolutions(facts []ast.Atom, queryStr string, onSolution func(map[string]any) error) error {
	r.mu.RLock()
	if !r.ready {
		r.mu.RUnlock()
		return fmt.Errorf("runtime not initialized")
	}

	workingStore := factstore.NewSimpleInMemoryStore()
	workingStore.Merge(r.baseFactStore)
	pInfo := r.programInfo
	strata := r.strata
	pStratum := r.predToStratum
	r.mu.RUnlock()

	for _, fact := range facts {
		workingStore.Add(fact)
	}

	if _, err := engine.EvalStratifiedProgramWithStats(pInfo, strata, pStratum, workingStore); err != nil {
		return fmt.Errorf("evaluation failed: %w", err)
	}

	queryAtom, err := parse.Atom(queryStr)
	if err != nil {
		return fmt.Errorf("failed to parse query '%s': %w", queryStr, err)
	}

	q := ast.NewQuery(queryAtom.Predicate)

	return workingStore.GetFacts(q, func(factAtom ast.Atom) error {
		// Manual Unification
		if len(factAtom.Args) != len(queryAtom.Args) {
			return nil
		}

		solution := make(map[string]any)
		match := true

		for i, queryArg := range queryAtom.Args {
			if v, isVar := queryArg.(ast.Variable); isVar {
				// It's a variable, capture the value
				valStr, err := constantToString(factAtom.Args[i])
				if err != nil {
					return fmt.Errorf("error converting term: %w", err)
				}
				solution[v.Symbol] = valStr
			} else {
				// It's a constant, check equality
				if !queryArg.Equals(factAtom.Args[i]) {
					match = false
					break
				}
			}
		}

		if match {
			return onSolution(solution)
		}
		return nil
	})
}

// evaluate helper (internal use only, assumes lock is held or local store)
func (r *MangleRuntime) evaluate(store factstore.FactStore) error {
	_, err := engine.EvalStratifiedProgramWithStats(r.programInfo, r.strata, r.predToStratum, store)
	return err
}

// --- Helper Functions ---

func isRuleFile(p string) bool {
	return strings.HasSuffix(p, ".dlog") || strings.HasSuffix(p, ".dl")
}

func isFactFile(p string) bool {
	return strings.HasSuffix(p, ".facts") ||
		strings.HasSuffix(p, ".fact") ||
		strings.HasSuffix(p, ".edb") ||
		strings.HasSuffix(p, ".data")
}

func cleanSource(raw string) string {
	// Strip UTF-8 BOM
	if strings.HasPrefix(raw, "\xef\xbb\xbf") {
		raw = raw[3:]
	}

	// Normalize line endings
	s := strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(s, "\n")
	kept := lines[:0]

	for _, ln := range lines {
		trimLn := strings.TrimSpace(ln)

		// 1. Skip empty lines
		if trimLn == "" {
			continue
		}

		// 2. Skip full line comments
		if strings.HasPrefix(trimLn, "%") || strings.HasPrefix(trimLn, "//") {
			continue
		}

		// 3. Handle inline comments while respecting quotes
		// We iterate through the string to find the start of a comment that is NOT inside a string.
		commentIdx := -1
		inQuote := false
		for i := 0; i < len(ln); i++ {
			char := ln[i]
			if char == '"' {
				// Handle escaped quotes if necessary, though Datalog usually implies simple escaping?
				// For now, toggle state. strictly speaking, we should check for backslash.
				escaped := false
				if i > 0 && ln[i-1] == '\\' {
					escaped = true
				}
				if !escaped {
					inQuote = !inQuote
				}
			}

			if !inQuote {
				// Check for %
				if char == '%' {
					commentIdx = i
					break
				}
				// Check for //
				if char == '/' && i+1 < len(ln) && ln[i+1] == '/' {
					commentIdx = i
					break
				}
			}
		}

		if commentIdx >= 0 {
			ln = ln[:commentIdx]
		}

		if strings.TrimSpace(ln) == "" {
			continue
		}

		kept = append(kept, ln)
	}
	return strings.Join(kept, "\n")
}

func parseRuleFile(file string) (parse.SourceUnit, error) {
	b, err := os.ReadFile(file)
	if err != nil {
		return parse.SourceUnit{}, fmt.Errorf("could not open rule file %s: %w", file, err)
	}
	cleaned := cleanSource(string(b))
	unit, err := parse.Unit(strings.NewReader(cleaned))
	if err != nil {
		return parse.SourceUnit{}, fmt.Errorf("could not parse rule file %s: %w", file, err)
	}
	return unit, nil
}

func constantToString(term ast.BaseTerm) (string, error) {
	if c, ok := term.(ast.Constant); ok {
		if v, err := c.StringValue(); err == nil {
			return v, nil
		}
		if v, err := c.NameValue(); err == nil {
			return v, nil
		}
		if v, err := c.NumberValue(); err == nil {
			return fmt.Sprintf("%d", v), nil
		}
		return "", fmt.Errorf("unsupported constant type: %v", c.Type)
	}
	return fmt.Sprintf("%v", term), nil
}

// resolveFiles remains the same as your original implementation...
func resolveFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.IsDir() {
			return collectFiles(path)
		}
		return []string{path}, nil
	case errors.Is(err, fs.ErrNotExist):
		if hasMeta(path) {
			matches, globErr := filepath.Glob(path)
			if globErr != nil {
				return nil, fmt.Errorf("path globbing failed for %q: %w", path, globErr)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no files matched %q", path)
			}
			var files []string
			for _, match := range matches {
				resolved, err := resolveFiles(match)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve glob match %q: %w", match, err)
				}
				files = append(files, resolved...)
			}
			sort.Strings(files)
			return files, nil
		}
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	default:
		return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
	}
}

func collectFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %q: %w", root, err)
	}
	sort.Strings(files)
	return files, nil
}

func hasMeta(path string) bool {
	return strings.ContainsAny(path, "*?[")
}
```

### `internal/engine/solver.go`
```go
package engine

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
	"github.com/google/mangle/ast"
	"github.com/google/mangle/parse"
)

// PolicyEngine is the core decision-making component of Manglekit.
// It orchestrates the loading of policies, maintaining the Datalog runtime,
// and executing authorization (Pre-Check) and validation (Post-Check) logic.
// It also integrates with observability (Tracing/Logging) to provide transparent governance.
type PolicyEngine struct {
	tracer  core.Tracer
	logger  core.Logger
	runtime *MangleRuntime
}

// New creates a new PolicyEngine with default no-op observability.
// This is suitable for basic usage where tracing and structured logging are not required.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func New() *PolicyEngine {
	pe := &PolicyEngine{
		tracer:  &core.NopTracer{},
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}

	// Auto-load Standard Library
	if err := pe.runtime.AddPolicy(resources.GetStdLib()); err != nil {
		panic("manglekit: failed to load std.dl: " + err.Error())
	}

	return pe
}

// NewWithTracer creates a new PolicyEngine with tracing enabled.
//
// Deprecated: Use NewWithObservability instead.
//
// Parameters:
//   - tracer: The tracer implementation to use.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func NewWithTracer(tracer core.Tracer) *PolicyEngine {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	return &PolicyEngine{
		tracer:  tracer,
		logger:  core.NopLogger{},
		runtime: NewMangleRuntime(),
	}
}

// NewWithObservability creates a new PolicyEngine with both tracing and logging enabled.
// This is the recommended constructor for production use.
//
// Parameters:
//   - tracer: The tracer implementation (e.g., OpenTelemetry).
//   - logger: The logger implementation.
//
// Returns:
//   - A pointer to a new PolicyEngine instance.
func NewWithObservability(tracer core.Tracer, logger core.Logger) *PolicyEngine {
	if tracer == nil {
		tracer = &core.NopTracer{}
	}
	if logger == nil {
		logger = core.NopLogger{}
	}

	pe := &PolicyEngine{
		tracer:  tracer,
		logger:  logger,
		runtime: NewMangleRuntime(),
	}

	// Load Planner Core Rules
	if err := pe.runtime.AddPolicy(resources.GetPlannerRules()); err != nil {
		if logger != nil {
			logger.Error("failed to load planner core schema", "error", err)
		}
	}

	// Load Manglekit Standard Library (std.dl)
	if err := pe.runtime.AddPolicy(resources.GetStdLib()); err != nil {
		if logger != nil {
			logger.Error("failed to load standard library", "error", err)
		}
		// Failure to load stdlib is critical for dynamic features
		panic("manglekit: failed to load std.dl: " + err.Error())
	}

	return pe
}

// RecordLineage records a data lineage relationship between a child and a parent.
// Note: In the current architecture, lineage is primarily handled via context propagation
// and tracing spans rather than explicit in-memory storage.
//
// Parameters:
//   - ctx: The context.
//   - childID: The ID of the derived data.
//   - parentID: The ID of the source data.
func (e *PolicyEngine) RecordLineage(ctx context.Context, childID, parentID string) {
	if e.tracer != nil {
		// Lineage linking is handled via context propagation in GuardedAction.
		// If explicit linking span events are needed, they can be added here.
	}
}

// Logger returns the engine's configured Logger instance.
// This allows other components (like GuardedAction) to reuse the engine's logger.
//
// Returns:
//   - The configured Logger, or a NopLogger if none was set.
func (e *PolicyEngine) Logger() core.Logger {
	if e.logger == nil {
		return core.NopLogger{}
	}
	return e.logger
}

// LoadFacts injects a list of raw Datalog fact strings into the runtime's base knowledge.
// This allows adding dynamic context or configuration at runtime (e.g., feature flags).
//
// Parameters:
//   - facts: A slice of Datalog fact strings.
//
// Returns:
//   - An error if parsing or evaluation fails.
func (e *PolicyEngine) LoadFacts(facts []string) error {
	return e.runtime.LoadFacts(facts)
}

// RegisterActionMetadata injects metadata about a registered action into the Datalog runtime.
// It generates facts that describe the action's interface, enabling the Planner to reason about it.
//
// Generated Facts:
//   - action("name")
//   - has_input("name", "InputType")
//   - has_output("name", "OutputType")
//
// Parameters:
//   - meta: The metadata of the action.
//
// Returns:
//   - An error if fact loading fails.
func (e *PolicyEngine) RegisterActionMetadata(meta core.ActionMetadata) error {
	var facts []string
	safeName := escapeString(meta.Name)

	facts = append(facts, fmt.Sprintf("action(\"%s\")", safeName))

	if meta.InputType != "" {
		facts = append(facts, fmt.Sprintf("has_input(\"%s\", \"%s\")", safeName, escapeString(meta.InputType)))
	}

	if meta.OutputType != "" {
		facts = append(facts, fmt.Sprintf("has_output(\"%s\", \"%s\")", safeName, escapeString(meta.OutputType)))
	}

	return e.LoadFacts(facts)
}

// LoadPolicy loads policy rules from a raw Datalog string.
// This decouples the engine from file I/O.
//
// Parameters:
//   - policy: The Datalog rules as a string.
//
// Returns:
//   - An error if parsing or loading fails.
func (e *PolicyEngine) LoadPolicy(policy string) error {
	if policy == "" {
		return nil
	}

	// Load the rules into the Mangle runtime
	if err := e.runtime.AddPolicy(policy); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	// Log successful load
	if e.logger != nil {
		e.logger.Debug("policy loaded from string", "length", len(policy))
	}

	return nil
}

// Authorize performs the Pre-Check phase of governance.
// It checks if the input is allowed to proceed based on the loaded policies.
// If the `deny(Req)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.PreCheck`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being authorized.
//   - input: The input envelope containing the payload and security labels.
//
// Returns:
//   - core.ErrAlignment if blocked, or nil if allowed.
func (e *PolicyEngine) Authorize(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.tracer == nil {
		return e.authorizeInternal(ctx, actionMeta, input)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPreCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(input.SecurityLabels) > 0 {
		span.SetAttr(core.AttrLabels, input.SecurityLabels)
	}

	err := e.authorizeInternal(ctx, actionMeta, input)
	if err != nil {
		span.Error(err)
	} else {
		span.SetAttr(core.AttrOutcome, core.OutcomeAllowed)
	}
	return err
}

// authorizeInternal executes the core authorization logic.
func (e *PolicyEngine) authorizeInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return nil // No runtime or program loaded, allow by default
	}

	// Convert the input payload to Mangle facts
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return fmt.Errorf("fact conversion error: %w", err)
	}

	// Inject Action Metadata facts
	// action_operation("Req", "Name")
	if actionMeta.Name != "" {
		safeName := core.EntityInput
		safeOp := actionMeta.Name
		opFactStr := fmt.Sprintf("action_operation(\"%s\", \"%s\")", escapeString(safeName), escapeString(safeOp))
		opAtom, err := parse.Atom(opFactStr)
		if err == nil {
			facts = append(facts, opAtom)
		}
	}

	// Generate facts for security labels using safe helper
	labelFacts, err := LabelsToFacts(core.EntityInput, input.SecurityLabels)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("failed to generate label facts", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return fmt.Errorf("label conversion error: %w", err)
	}

	for _, factStr := range labelFacts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse label fact", "fact", factStr, "error", err)
			}
			// Return actual error
			return fmt.Errorf("label parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// [NEW] Inject Explicit Facts from Envelope
	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", factStr, "error", err)
			}
			return fmt.Errorf("envelope fact parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// Execute the deny(Req) query
	denied, err := e.runtime.ExecuteQuery(facts, fmt.Sprintf(`deny("%s")`, core.EntityInput))
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("policy evaluation failed", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return fmt.Errorf("policy evaluation error: %w", err)
	}

	if denied {
		if e.logger != nil {
			e.logger.Debug("alignment issue detected", "action", actionMeta.Name)
		}
		return core.ErrAlignment
	}

	return nil
}

// Validate performs the Post-Check phase of governance.
// It checks if the output is allowed to be returned to the caller.
// If the `deny(Output)` predicate is derived, it returns `core.ErrAlignment`.
//
// It automatically starts a tracing span (`Datalog.PostCheck`) and logs attributes.
//
// Parameters:
//   - ctx: The execution context.
//   - actionMeta: Metadata about the action being validated.
//   - output: The output envelope containing the result.
//
// Returns:
//   - The validated envelope (potentially modified, though currently pass-through), or an error.
func (e *PolicyEngine) Validate(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.tracer == nil {
		return e.validateInternal(ctx, actionMeta, output)
	}

	ctx, span := e.tracer.Start(ctx, core.SpanPostCheck)
	defer span.End()

	// Log security labels to span attributes for audit
	if len(output.SecurityLabels) > 0 {
		span.SetAttr(core.AttrLabels, output.SecurityLabels)
	}

	result, err := e.validateInternal(ctx, actionMeta, output)
	if err != nil {
		span.Error(err)
		return core.Envelope{}, err
	}
	span.SetAttr(core.AttrOutcome, core.OutcomeAllowed)
	return result, nil
}

// validateInternal executes the core validation logic.
func (e *PolicyEngine) validateInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	if e.runtime == nil || e.runtime.programInfo == nil {
		return output, nil // No runtime or program loaded, allow by default
	}

	// Convert the output payload to Mangle facts
	facts, err := toMangleFacts(core.EntityOutput, output.Payload, output.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert output to facts", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return core.Envelope{}, fmt.Errorf("fact conversion error: %w", err)
	}

	// Generate facts for security labels using safe helper
	labelFacts, err := LabelsToFacts(core.EntityOutput, output.SecurityLabels)
	if err != nil {
		if e.logger != nil {
			e.logger.Error("failed to generate label facts", "error", err)
		}
		// Return actual error
		return core.Envelope{}, fmt.Errorf("label conversion error: %w", err)
	}

	for _, factStr := range labelFacts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse label fact", "fact", factStr, "error", err)
			}
			// Return actual error
			return core.Envelope{}, fmt.Errorf("label parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// [NEW] Inject Explicit Facts from Envelope
	for _, factStr := range output.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", factStr, "error", err)
			}
			return core.Envelope{}, fmt.Errorf("envelope fact parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// Execute the deny(Output) query for post-check validation
	denied, err := e.runtime.ExecuteQuery(facts, fmt.Sprintf(`deny("%s")`, core.EntityOutput))
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("post-check validation failed", "error", err)
		}
		// Return actual error to allow Fail-Open handling
		return core.Envelope{}, fmt.Errorf("policy evaluation error: %w", err)
	}

	if denied {
		// Teacher-Student Protocol: Try to extract a human-readable violation message
		// Query: violation_msg(Msg)
		var violationMsg string
		// We reuse the facts which already contain the Output payload
		_ = e.runtime.QueryWithSolutions(facts, "violation_msg(Msg)", func(solution map[string]any) error {
			if msg, ok := solution["Msg"].(string); ok {
				violationMsg = msg
				return fmt.Errorf("found") // Stop searching
			}
			return nil
		})

		if e.logger != nil {
			e.logger.Debug("post-check validation violation detected", "action", actionMeta.Name, "violation_msg", violationMsg)
		}

		if violationMsg != "" {
			return core.Envelope{}, &core.AlignmentError{Message: violationMsg}
		}
		return core.Envelope{}, core.ErrAlignment
	}

	return output, nil
}

// EvaluateSteering executes "Steering Policies" which determine what to do next.
// Unlike Authorize/Validate (which are binary Allow/Deny), Steering returns decisions like "Retry" or "Route".
//
// Logic Priority:
//  1. Correction (Retry): If `correction(Req, Hint)` is derived, we return `RETRY` with the hint.
//  2. Routing (Route): If `next_step(Req, Target)` is derived, we return `ROUTE` with the target.
//  3. Default: `ALLOW` (Proceed as normal).
//
// Parameters:
//   - ctx: The execution context.
//   - input: The input envelope.
//
// Returns:
//   - decision: The decision string (e.g., "RETRY", "ROUTE", "ALLOW").
//   - metadata: A map containing steering details (e.g., {"feedback": "hint"}).
//   - error: An error if evaluation fails.
func (e *PolicyEngine) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	decision := core.DecisionAllow
	metadata := make(map[string]string)

	if e.runtime == nil || e.runtime.programInfo == nil {
		return decision, metadata, nil
	}

	// Convert the input payload to Mangle facts
	// We use "Req" as the entity ID, consistent with Authorize
	facts, err := toMangleFacts(core.EntityInput, input.Payload, input.ContentType)
	if err != nil {
		if e.logger != nil {
			e.logger.Debug("failed to convert input to facts for steering", "error", err)
		}
		return decision, metadata, fmt.Errorf("failed to convert input to facts: %w", err)
	}

	// [NEW] Inject Explicit Facts from Envelope
	for _, factStr := range input.Facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			if e.logger != nil {
				e.logger.Error("failed to parse envelop fact", "fact", factStr, "error", err)
			}
			return decision, metadata, fmt.Errorf("envelope fact parsing error: %w", err)
		}
		facts = append(facts, atom)
	}

	// 1. Check Correction (Retry)
	// Query: correction("Req", Hint)
	_ = e.runtime.QueryWithSolutions(facts, fmt.Sprintf(`correction("%s", Hint)`, core.EntityInput), func(solution map[string]any) error {
		if hint, ok := solution["Hint"].(string); ok {
			decision = core.DecisionRetry
			metadata[core.KeyFeedback] = hint
			// Stop searching after first match
			return fmt.Errorf("found") // Use error to break early
		}
		return nil
	})

	if decision == core.DecisionRetry {
		return decision, metadata, nil
	}

	// 2. Check Routing
	// Query: next_step("Req", Target)
	_ = e.runtime.QueryWithSolutions(facts, fmt.Sprintf(`next_step("%s", Target)`, core.EntityInput), func(solution map[string]any) error {
		if target, ok := solution["Target"].(string); ok {
			decision = core.DecisionRoute
			metadata[core.KeyNextStep] = target
			return fmt.Errorf("found") // Use error to break early
		}
		return nil
	})

	return decision, metadata, nil
}

// ExecuteQuery executes a raw Datalog query against the current program state.
// It wraps the underlying runtime execution with observability (tracing).
//
// Parameters:
//   - ctx: The execution context.
//   - facts: Temporary facts to include in this specific query execution.
//   - queryStr: The Datalog query atom.
//
// Returns:
//   - true if derived, false otherwise.
//   - error if execution fails.
func (e *PolicyEngine) ExecuteQuery(ctx context.Context, facts []ast.Atom, queryStr string) (bool, error) {
	if e.tracer == nil {
		return e.runtime.ExecuteQuery(facts, queryStr)
	}

	ctx, span := e.tracer.Start(ctx, "Datalog.ExecuteQuery")
	defer span.End()

	span.SetAttr("datalog.query", queryStr)

	res, err := e.runtime.ExecuteQuery(facts, queryStr)
	if err != nil {
		span.Error(err)
		return false, err
	}

	span.SetAttr("datalog.result", res)
	return res, nil
}

// Query executes a Datalog query and returns all matching solutions.
// Each solution is a map where keys are variable names (e.g., "Action") and values are stringified constants.
//
// Parameters:
//   - ctx: The execution context.
//   - facts: Temporary facts (strings) to include.
//   - queryStr: The Datalog query with variables (e.g., 'plan_step(Action, Order)').
//
// Returns:
//   - A list of solution maps.
//   - An error if execution fails.
func (e *PolicyEngine) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error) {
	var results []map[string]string

	if e.runtime == nil {
		return nil, fmt.Errorf("runtime not initialized")
	}

	// Parse temporary facts
	var atomFacts []ast.Atom
	for _, f := range facts {
		atom, err := parse.Atom(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", f, err)
		}
		atomFacts = append(atomFacts, atom)
	}

	if e.tracer != nil {
		var span core.Span
		ctx, span = e.tracer.Start(ctx, "Datalog.Query")
		defer span.End()
		span.SetAttr("datalog.query", queryStr)
	}

	err := e.runtime.QueryWithSolutions(atomFacts, queryStr, func(solution map[string]any) error {
		// Convert map[string]any to map[string]string
		strMap := make(map[string]string)
		for k, v := range solution {
			if s, ok := v.(string); ok {
				strMap[k] = s
			} else {
				strMap[k] = fmt.Sprintf("%v", v)
			}
		}
		results = append(results, strMap)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

// toMangleFacts helper converts a Go struct to Mangle atoms via the Reflection API.
func toMangleFacts(entityID string, input any, contentType core.ContentType) ([]ast.Atom, error) {
	if input == nil {
		return nil, nil
	}

	var atoms []ast.Atom
	var facts []string
	var err error

	// Choose strategy based on ContentType
	if contentType == core.TypeJSON {
		facts, err = Flatten(entityID, input)
	} else {
		// Default to Reflection (Struct)
		facts, err = ToFacts(entityID, input)
	}

	if err != nil {
		return nil, err
	}

	// Parse each fact string back into an ast.Atom
	for _, factStr := range facts {
		atom, err := parse.Atom(factStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fact '%s': %w", factStr, err)
		}
		atoms = append(atoms, atom)
	}

	return atoms, nil
}
```

### `core/action.go`
```go
package core

import "context"

// ActionMetadata provides metadata about an action.
// It is used for routing, observability, and debugging.
type ActionMetadata struct {
	// Name is the unique identifier for the action (e.g., "generate-content").
	Name string
	// Type describes the category of the action (e.g., "llm", "tool", "rag").
	Type string
	// InputContentType specifies the expected input format (Struct or JSON).
	InputContentType ContentType
	// InputType is the string name of the Go input type (e.g., "StockReq").
	InputType string
	// OutputType is the string name of the Go output type (e.g., "StockRes").
	OutputType string
	// IsDynamic indicates if the input type is generic (e.g., map[string]any or JSON).
	IsDynamic bool
}

// Action defines the interface for a unit of work in the Manglekit system.
// Any component that processes data (LLMs, databases, external APIs) must implement this interface.
type Action interface {
	// Execute performs the action's logic.
	//
	// It accepts a context for cancellation/timeout and an input Envelope containing the data.
	// It returns a new Envelope containing the result or an error if execution failed.
	Execute(ctx context.Context, input Envelope) (Envelope, error)

	// Metadata returns the action's metadata, including its name and type.
	Metadata() ActionMetadata
}
```

### `core/envelope.go`
```go
package core

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Envelope struct defines a standard communication structure used across Manglekit.
// It encapsulates data, metadata, and security context (taint labels) to ensure
// safe and traceable propagation through the system.
type Envelope struct {
	// ID is the unique identifier for this specific data envelope.
	ID uuid.UUID
	// Payload is the actual data being transported (e.g., a string, or a struct).
	Payload any
	// Metadata stores key-value pairs for control plane signaling (e.g., decision, latency).
	Metadata map[string]string
	// SecurityLabels holds taint tags (e.g., "secret", "pii") for information flow control.
	SecurityLabels []string
	// Facts holds structured logical facts extracted from the payload (e.g., "topic(billing)").
	// These are fed directly into the Logic Engine for reasoning.
	Facts []string
	// ContentType indicates whether the payload is a Struct or JSON.
	ContentType ContentType
}

// NewEnvelope creates a new envelope with the provided payload.
// It initializes a new UUID, an empty metadata map, and an empty list of security labels.
//
// Parameters:
//   - payload: The data to be wrapped in the envelope.
//
// Returns:
//   - A new Envelope instance.
func NewEnvelope(payload any) Envelope {
	return Envelope{
		ID:             uuid.New(),
		Payload:        payload,
		Metadata:       make(map[string]string),
		SecurityLabels: []string{},
		ContentType:    TypeStruct, // Default to Typed Mode
	}
}

// SetMeta sets a value in the envelope's metadata map.
//
// Parameters:
//   - k: The metadata key (e.g., core.KeyDecision).
//   - v: The metadata value.
func (e *Envelope) SetMeta(k, v string) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata[k] = v
}

// GetMeta retrieves a value from the envelope's metadata map.
//
// Parameters:
//   - k: The metadata key to retrieve.
//
// Returns:
//   - The value associated with the key, or an empty string if not found.
func (e *Envelope) GetMeta(k string) string {
	if e.Metadata == nil {
		return ""
	}
	return e.Metadata[k]
}

// SetFeedback injects the "Teacher's" feedback into metadata
func (e *Envelope) SetFeedback(msg string) {
	if e.Metadata == nil {
		e.Metadata = make(map[string]string)
	}
	e.Metadata["manglekit.feedback"] = msg
}

// GetFeedback retrieves the feedback for the "Student" (AI/Logic)
func (e *Envelope) GetFeedback() string {
	if e.Metadata == nil {
		return ""
	}
	return e.Metadata["manglekit.feedback"]
}

// AddLabel adds a security label to the envelope if it does not already exist.
// This is used for taint tracking (e.g., marking data as "secret").
//
// Parameters:
//   - label: The security label string to add.
func (e *Envelope) AddLabel(label string) {
	if !e.HasLabel(label) {
		e.SecurityLabels = append(e.SecurityLabels, label)
	}
}

// HasLabel checks for the existence of a specific security label on the envelope.
//
// Parameters:
//   - label: The security label to check for.
//
// Returns:
//   - true if the label exists, false otherwise.
func (e *Envelope) HasLabel(label string) bool {
	for _, l := range e.SecurityLabels {
		if l == label {
			return true
		}
	}
	return false
}

// MergeLabels appends distinct labels from another source (e.g., another Envelope) to this one.
//
// Parameters:
//   - other: A slice of label strings to merge.
func (e *Envelope) MergeLabels(other []string) {
	for _, l := range other {
		e.AddLabel(l)
	}
}

// SetHistory serializes a list of chat messages into the envelope's metadata.
// This is used to persist conversation context across stateless executions.
//
// Parameters:
//   - msgs: The slice of ChatMessage objects to serialize.
func (e *Envelope) SetHistory(msgs []ChatMessage) {
	b, err := json.Marshal(msgs)
	if err == nil {
		e.SetMeta(KeyHistory, string(b))
	}
}
```

### `config/loader.go`
```go
package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads a YAML configuration file from the given path and returns a Config object.
// It also expands environment variables in the YAML content.
// This function ports the legacy loading logic to the new architecture.
//
// Environment variable expansion supports the standard ${VAR_NAME} syntax.
// Example: ${API_KEY} will be replaced with the value of the API_KEY environment variable.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}
	return ParseConfig(data)
}

// ParseConfig unmarshals a byte slice into a Config object.
// It also expands environment variables in the YAML content before unmarshaling.
func ParseConfig(data []byte) (*Config, error) {
	// Expand environment variables in the YAML content
	expandedContent := []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(expandedContent, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

// LoadFromReader reads a YAML configuration from the provided reader and returns a Config object.
// It also expands environment variables in the YAML content.
func LoadFromReader(r io.Reader) (*Config, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}
	return ParseConfig(content)
}

// applyDefaults applies sensible defaults to the configuration if not already set.
func applyDefaults(cfg *Config) {
	if cfg.Observability.ServiceName == "" {
		cfg.Observability.ServiceName = "manglekit-app"
	}

	if cfg.Observability.LogLevel == "" {
		cfg.Observability.LogLevel = "info"
	}
}
```

### `adapters/mcp/action.go`
```go
package mcp

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
)

// MCPAction wraps a Genkit AI Tool discovered from an MCP server.
type MCPAction struct {
	tool       ai.Tool
	serverName string
	name       string
	initError  error
}

// NewAction creates a new MCPAction.
// If tool is nil (e.g. startup failure), name must be provided.
func NewAction(serverName string, tool ai.Tool, name string, initError error) *MCPAction {
	actionName := name
	if tool != nil {
		// Default naming strategy: mcp_<server>_<tool>
		actionName = tool.Name()
	}

	return &MCPAction{
		tool:       tool,
		serverName: serverName,
		name:       actionName,
		initError:  initError,
	}
}

// Execute invokes the underlying MCP tool via Genkit.
func (a *MCPAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Check for initialization error (Health Check Pattern)
	if a.initError != nil {
		return core.Envelope{}, fmt.Errorf("tool '%s' is unavailable due to startup failure: %w", a.name, a.initError)
	}

	// Safety check
	if a.tool == nil {
		return core.Envelope{}, fmt.Errorf("tool '%s' is invalid: nil tool implementation", a.name)
	}

	// Invoke the tool using RunRaw, which handles input marshaling if needed.
	res, err := a.tool.RunRaw(ctx, input.Payload)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("execution failed for mcp tool %s: %w", a.tool.Name(), err)
	}

	return core.NewEnvelope(res), nil
}

// Metadata returns the action metadata.
func (a *MCPAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: fmt.Sprintf("mcp_%s_%s", a.serverName, a.name),
		Type: "mcp_tool",
	}
}
```

## 5. CHANGELOG

*   **2025-12-11:** Full Context Resync. Performed exhaustive repository scan and updated documentation to reflect the codebase state, including full source dumps of critical components.
