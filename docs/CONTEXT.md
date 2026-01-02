---
context_type: kernel_source_dump
project: manglekit
language: go, datalog
last_updated: 2026-01-02T07:15:00Z
scan_mode: logic_focused
---

## 1. THE COMPLETE FILE MAP

.
├── adapters/                    # Universal adapters for external libraries
│   ├── ai/                     # Genkit LLM integration
│   ├── extractor/              # Structured data extraction from LLM
│   ├── func/                   # Go function wrapper
│   ├── knowledge/              # RDF/Knowledge Graph loading
│   ├── logger/                 # Logger implementations (Zap)
│   ├── mcp/                    # Model Context Protocol tool discovery
│   ├── memory/                 # Memory adapters (in-memory stores)
│   ├── resilience/             # Circuit breaker pattern
│   └── vector/                 # Vector store & RAG retrieval
├── cmd/                         # CLI tool (excluded from dump)
│   └── mkit/                   # CLI commands
├── config/                      # Configuration loading (YAML)
├── core/                        # Core contracts and interfaces
├── docs/                        # Technical documentation
│   └── designs/                # Design documents for new features
│       └── attention_sink.md   # Attention Sink mechanism design
├── examples/                    # Example applications (excluded from dump)
├── internal/                    # Internal implementation
│   ├── engine/                 # Neuro-Symbolic Core (Mangle Runtime)
│   │   ├── memory/              # Memory implementations
│   │   ├── parse/               # Datalog parsing utilities
│   │   └── resources/           # Embedded Datalog rules
│   ├── logger/                 # Default logger implementation
│   ├── resources/              # Shared resources (ICL golden samples)
│   ├── statehelper/            # Conversation state management
│   ├── supervisor/             # Action Sandwich (Trace-Assess-Exec-Reflect)
│   ├── telemetry/              # OpenTelemetry setup
│   ├── testproviders/          # Mocks for testing
│   ├── tools/                  # Internal tooling
│   └── util/                   # Utilities (Schema, etc.)
├── providers/                   # Plugin factories
│   ├── google/                 # Google GenAI plugin
│   ├── memory/                 # Memory providers
│   └── openai/                 # OpenAI plugin (LLM & Embeddings)
├── sdk/                         # Orchestration layer (Client, Loop, Planner)
├── .env.example
├── .gitignore
├── AGENTS.md
├── CONTRIBUTING.md
├── go.mod
├── go.sum
├── LICENSE
├── Makefile
├── mangle.yaml
├── manglekit.go
├── mkit
└── README.md

---

## 2. COMPONENTS (The Logic)

### Component: core
**1. Responsibilities:**
- Defines the system's contract interfaces and types
- Pure abstract layer with no dependencies
- Provides constants for metadata keys, decisions and observability

**2. Core Structs:**
- **Envelope**: Standard data carrier with ID, Payload, Metadata, SecurityLabels, Facts, ContentType
- **ActionMetadata**: Describes a registered capability (Name, Type, InputType, OutputType)
- **Decision**: Structured result from Policy Engine (Outcome, Target, Reasons, Meta)
- **ExecutionContext**: Captures runtime state for durable state management (RetryCount, FeedbackHistory, CurrentHistory)
- **Message**: Represents a chat message (Role, Content)
- **GenerationConfig**: Holds standard LLM parameters (Temperature, MaxTokens, TopP, Model, JSONMode)
- **Document**: Represents a snippet of knowledge/memory (ID, Content, Vector, Metadata, Score)
- **Answer**: Represents a structured system response (Text, Meta)

**3. Key Functions:**
- `func NewEnvelope(payload any) Envelope` - Creates a new Envelope with generated UUID
- `func (e *Envelope) SetMeta(key string, value any)` - Sets a metadata key-value pair
- `func (e *Envelope) GetMeta(k string) string` - Retrieves metadata value as string
- `func (e *Envelope) SetFeedback(msg string)` - Injects feedback into metadata
- `func (e *Envelope) GetFeedback() string` - Retrieves feedback for AI/Logic
- `func (e *Envelope) AddLabel(label string)` - Adds a security label to envelope
- `func (e *Envelope) HasLabel(label string) bool` - Checks for security label existence
- `func (e *Envelope) MergeLabels(other []string)` - Merges distinct labels from another source
- `func (e *Envelope) SetHistory(msgs []Message)` - Serializes chat messages into metadata

### Component: core/logic.go
**1. Responsibilities:**
- Defines Action interface for executable units
- Defines TextGenerator interface for LLM abstraction
- Defines Extractor interface for structured data extraction

**2. Core Structs:**
- **LLMResponse**: Contains generated text and token usage metadata (Text, Usage)

**3. Key Functions:**
- `type Action interface { Execute(ctx, Envelope) (Envelope, error); Metadata() ActionMetadata }` - Defines executable unit interface
- `type TextGenerator interface { Complete(ctx, prompt) (string, error); Generate(ctx, prompt, opts...) (*LLMResponse, error); Stream(ctx, prompt) (<-chan string, error) }` - Abstracts LLM
- `type Extractor interface { Extract(ctx, input string, schema any) error }` - Converts text to structured data

### Component: core/data.go
**1. Responsibilities:**
- Defines interfaces for knowledge loading and memory persistence
- Provides no-op implementations for stateless mode

**2. Core Structs:**
- **NopStore**: No-op implementation of HistoryStore for stateless mode

**3. Key Functions:**
- `type FactLoader interface { LoadFacts(ctx, source string) ([]string, error) }` - Loads external data into Engine
- `type HistoryStore interface { Read(ctx, sessionID string) ([]Message, error); Append(ctx, sessionID string, msgs []Message) error }` - Manages chat history persistence
- `func (n NopStore) Read(_ context.Context, _ string) ([]Message, error)` - Returns empty history
- `func (n NopStore) Append(_ context.Context, _ string, _ []Message) error` - Successful no-op

### Component: core/governance.go
**1. Responsibilities:**
- Defines Evaluator interface for policy execution
- Defines PreProcessor, RiskEngine, ResourceMonitor interfaces

**2. Core Structs:**
- N/A (interface definitions only)

**3. Key Functions:**
- `type Evaluator interface { AssessPlan(ctx, input Envelope) (Decision, error); Assess(ctx, actionMeta ActionMetadata, input Envelope) error; Reflect(ctx, actionMeta ActionMetadata, output Envelope) (Envelope, error); EvaluateSteering(ctx, input Envelope) (string, map[string]string, error); GetActionConfig(ctx, input Envelope) (map[string]string, error); CheckRequirement(ctx, input Envelope, reqName string) (bool, error); LoadPolicy(ctx, source string) error; LoadFacts(facts []string) error; RegisterAction(meta ActionMetadata) error; Query(ctx, facts []string, queryStr string) ([]map[string]string, error); Logger() Logger }` - The Mangle Logic Engine contract
- `type PreProcessor interface { Process(ctx, input Envelope) (map[string]any, error) }` - Fast/stateless checks
- `type RiskEngine interface { CalculateRisk(ctx, input Envelope) (float64, error) }` - Specialized risk calculation
- `type ResourceMonitor interface { CountTokens(ctx, text string) (int, error); CheckBudget(ctx, key string, cost int) (bool, error) }` - Cost & rate limiting

### Component: core/embedder.go
**1. Responsibilities:**
- Defines Embedder interface for text-to-vector conversion
- Defines VectorStore interface for vector storage and retrieval

**2. Core Structs:**
- N/A (interface definitions only)

**3. Key Functions:**
- `type Embedder interface { Embed(ctx context.Context, text string) ([]float32, error); EmbedBatch(ctx context.Context, texts []string) ([][]float32, error); Dimension() int }` - Defines contract for text embedding
- `type VectorStore interface { Upsert(ctx context.Context, id string, content string) error; Search(ctx context.Context, query string, topK int) ([]string, error); Get(ctx context.Context, id string) (string, error) }` - Defines contract for vector storage

### Component: sdk/client.go
**1. Responsibilities:**
- Primary entry point for Manglekit system
- Manages governance kernel, blueprints, observability and action execution
- Handles configuration loading and dependency initialization

**2. Core Structs:**
- **Client**: Main entry point holding engine, tracer, logger, agentMemory, registry, failureMode, blueprintPath, shutdownFunc, llm, stateManager

**3. Key Functions:**
- `func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error)` - Initializes SDK client with configured options
- `func NewClientFromFile(ctx context.Context, configPath string, opts ...ClientOption) (*Client, error)` - Initializes Client from YAML file
- `func NewClientFromConfig(ctx context.Context, cfg *config.Config, opts ...ClientOption) (*Client, error)` - Initializes Client from Config object
- `func (c *Client) Supervise(action core.Action) core.Action` - Wraps raw Action in SupervisedAction
- `func (c *Client) Engine() core.Evaluator` - Returns underlying policy engine
- `func (c *Client) LoadFacts(facts []string) error` - Manually injects Datalog facts into engine
- `func (c *Client) Tracer() trace.Tracer` - Returns OpenTelemetry Tracer
- `func (c *Client) Logger() core.Logger` - Returns configured Logger
- `func NewDefault() (*Client, error)` - Creates Client with sensible default settings
- `func (c *Client) SetLLM(gen core.TextGenerator)` - Manually configures TextGenerator
- `func (c *Client) RegisterAction(name string, action core.Action)` - Adds action to internal registry
- `func (c *Client) Shutdown(ctx context.Context) error` - Cleans up resources
- `func (c *Client) Memory() core.AgentMemory` - Returns active memory provider

### Component: sdk/loop.go
**1. Responsibilities:**
- Implements Semantic State Machine loop
- Manages retry logic, routing, and execution limits
- Handles memory storage and durable state management

**2. Core Structs:**
- **ExecutionParams**: Context object for RunLoop tracking retries and feedback

**3. Key Functions:**
- `func (c *Client) ExecuteByName(ctx context.Context, actionName string, input any, opts ...ExecuteOption) (core.Envelope, error)` - Executes named action within governance loop
- `func (c *Client) runLoopInternal(ctx context.Context, startAction string, payload any, params ExecutionParams) (core.Envelope, error)` - Implements core Semantic State Machine loop
- `func (c *Client) ExecuteSingleStep(ctx context.Context, actionName string, payload any, params *ExecutionParams) (core.Envelope, error)` - Runs one step of action and returns decision
- `func (c *Client) injectContext(ctx context.Context, env *core.Envelope, payload any, params *ExecutionParams)` - Populates envelope with feedback, history, RAG context, metadata, facts
- `func (c *Client) handleExecutionError(ctx context.Context, err error, payload any, params *ExecutionParams) (core.Envelope, error)` - Processes errors with retry logic
- `func (c *Client) updateHistory(ctx context.Context, payload any, result core.Envelope, params *ExecutionParams)` - Appends exchange to history and persists
- `func (c *Client) handleDecision(ctx context.Context, actionName string, result core.Envelope, payload any, params *ExecutionParams) (core.Envelope, error)` - Processes steering decision
- `func (c *Client) handleRetryDecision(ctx context.Context, actionName string, result core.Envelope, params *ExecutionParams) (core.Envelope, error)` - Processes RETRY with backoff
- `func (c *Client) buildHaltError(result core.Envelope) error` - Extracts reason from metadata and builds halt error
- `func (c *Client) backoff(ctx context.Context, retryCount int) error` - Handles sleep and context cancellation
- `func (c *Client) recallContext(ctx context.Context, payload any, env *core.Envelope)` - Handles RAG lookup logic
- `func (c *Client) asyncMemorize(input any, output any)` - Handles fire-and-forget storage logic
- `func normalizeString(s string) string` - Lowercases, removes punctuation, collapses spaces
- `func isSemanticallySimilar(s1, s2 string) bool` - Checks if two strings are semantically equivalent

### Component: internal/engine/runtime.go
**1. Responsibilities:**
- Encapsulates Google Mangle Datalog engine
- Handles loading, parsing, analysis, and stratification of Datalog programs
- Manages fact stores and query execution

**2. Core Structs:**
- **MangleRuntime**: Low-level wrapper around google/mangle with programInfo, strata, predToStratum, baseFactStore, ruleUnits, ready flag

**3. Key Functions:**
- `func NewMangleRuntime() *MangleRuntime` - Initializes a new, empty MangleRuntime
- `func (r *MangleRuntime) Load(path string) error` - Loads Datalog rules and facts from specified path (REPLACES current program state)
- `func (r *MangleRuntime) LoadFromSource(source string) error` - Parses and loads full Datalog program from a string (REPLACES current program state)
- `func (r *MangleRuntime) LoadFacts(facts []string) error` - Injects a list of raw Datalog fact strings into runtime's base knowledge
- `func (r *MangleRuntime) LoadFromString(rule string) error` - Parses and loads Datalog from string (alias for LoadFromSource)
- `func (r *MangleRuntime) AddPolicy(source string) error` - Adds new rules to existing program state (Incremental Loading)
- `func (r *MangleRuntime) ExecuteQuery(facts []ast.Atom, queryStr string) (bool, error)` - Runs boolean Datalog query
- `func (r *MangleRuntime) QueryWithSolutions(facts []ast.Atom, queryStr string, onSolution func(map[string]any) error) error` - Executes query and invokes callback for solutions
- `func (r *MangleRuntime) evaluate(store factstore.FactStore) error` - Helper for internal evaluation
- `func isRuleFile(p string) bool` - Checks if file is a rule file (.dlog or .dl)
- `func isFactFile(p string) bool` - Checks if file is a fact file (.facts, .fact, .edb, .data)
- `func cleanSource(raw string) string` - Strips UTF-8 BOM, normalizes line endings, removes comments
- `func parseRuleFile(file string) (parse.SourceUnit, error)` - Parses a rule file into SourceUnit
- `func constantToString(term ast.BaseTerm) (string, error)` - Converts constant term to string representation
- `func resolveFiles(path string) ([]string, error)` - Resolves files from path (file or directory/glob)

### Component: internal/engine/solver.go
**1. Responsibilities:**
- Core decision-making component of Manglekit
- Orchestrates loading of policies, maintaining Datalog runtime
- Executes authorization (Pre-Check) and validation (Post-Check) logic
- Integrates with observability (Tracing/Logging)

**2. Core Structs:**
- **PolicyEngine**: Main logic engine wrapping MangleRuntime with tracer, logger, runtime

**3. Key Functions:**
- `func New() (*PolicyEngine, error)` - Creates new PolicyEngine with default no-op observability
- `func NewWithTracer(tracer core.Tracer) *PolicyEngine` - Creates new PolicyEngine with tracing enabled (deprecated)
- `func NewWithObservability(tracer core.Tracer, logger core.Logger) (*PolicyEngine, error)` - Creates new PolicyEngine with tracing and logging (recommended for production)
- `func (e *PolicyEngine) RecordLineage(ctx context.Context, childID, parentID string)` - Records data lineage relationship
- `func (e *PolicyEngine) Logger() core.Logger` - Returns engine's configured Logger
- `func (e *PolicyEngine) LoadFacts(facts []string) error` - Injects raw Datalog fact strings into runtime's base knowledge
- `func (e *PolicyEngine) RegisterAction(meta core.ActionMetadata) error` - Injects metadata about a registered action into Datalog runtime
- `func (e *PolicyEngine) LoadPolicy(ctx context.Context, policy string) error` - Loads policy rules from a raw Datalog string
- `func (e *PolicyEngine) LoadGherkinPolicy(ctx context.Context, featureContent string) error` - Loads Gherkin feature file and compiles to Datalog
- `func (e *PolicyEngine) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error)` - High-level assessment mapping Assess logic to a Decision
- `func (e *PolicyEngine) GetActionConfig(ctx context.Context, input core.Envelope) (map[string]string, error)` - Queries engine for dynamic configuration parameters
- `func (e *PolicyEngine) Assess(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error` - Performs Pre-Check phase (input validation)
- `func (e *PolicyEngine) Reflect(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error)` - Performs Post-Check phase (output validation)
- `func (e *PolicyEngine) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error)` - Determines next step (Retry/Route) based on output
- `func (e *PolicyEngine) assessInternal(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error` - Internal core authorization logic
- `func (e *PolicyEngine) reflectInternal(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error)` - Internal core validation logic
- `func (e *PolicyEngine) evaluateGate(ctx context.Context, actionName string, entityID string, env core.Envelope, extraFacts ...ast.Atom) error` - Centralizes logic for Check -> Deny -> Explain
- `func (e *PolicyEngine) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)` - Executes a Datalog query and returns all matching solutions
- `func escapeString(s string) string` - Escapes special characters to ensure valid Mangle string constant

### Component: internal/engine/reflection.go
**1. Responsibilities:**
- Converts Go data structures into Mangle Datalog facts
- Entry point for turning runtime objects into logic predicates
- Handles cycle detection and field tag processing

**2. Core Structs:**
- N/A (pure functions for conversion)

**3. Key Functions:**
- `func ToFacts(id string, input any) ([]string, error)` - Converts a Go data structure into Mangle Datalog facts
- `func LabelsToFacts(entityID string, labels []string) ([]string, error)` - Converts security label strings into Mangle Datalog facts
- `func escapeString(s string) string` - Escapes special characters to ensure valid Mangle string constant
- `func toFactsRecursive(id, path string, v reflect.Value, facts *[]string, visited map[uintptr]bool, args ...string) error` - Recursive reflection logic
- `func generatePrimitiveFact(id, path string, v reflect.Value, facts *[]string, args ...string)` - Creates final Datalog string for primitive values

### Component: internal/engine/flattener.go
**1. Responsibilities:**
- Converts ANY dynamic structure (maps, slices of any type) into graph facts
- Handles JSON flattening for Datalog consumption
- Supports cycle detection and complex type handling

**2. Core Structs:**
- N/A (pure functions for flattening)

**3. Key Functions:**
- `func Flatten(rootID string, input any) ([]string, error)` - Converts dynamic structure into graph facts
- `func flattenRecursive(nodeID string, v reflect.Value, facts *[]string, counter *int, visited map[uintptr]bool) error` - Recursive flattening logic
- `func isComplexKind(k reflect.Kind) bool` - Helper to check complexity based on Kind
- `func addPrimitiveReflect(nodeID, key string, v reflect.Value, facts *[]string)` - Uses reflect.Value to handle types precisely

### Component: internal/supervisor/supervisor.go
**1. Responsibilities:**
- Implements "Guarded Action" pattern
- Enforces "Trace -> Assess -> Execute -> Reflect" lifecycle
- Handles tracing, security label propagation, and failure modes

**2. Core Structs:**
- **SupervisedAction**: Decorator that wraps core.Action with inner, engine, tracer, failureMode

**3. Key Functions:**
- `func NewSupervisedAction(action core.Action, eng core.Evaluator, failureMode string) *SupervisedAction` - Creates new SupervisedAction with default settings (no tracing)
- `func NewSupervisedActionWithTracer(action core.Action, eng core.Evaluator, tracer core.Tracer, failureMode string) *SupervisedAction` - Creates new SupervisedAction with tracing enabled
- `func (g *SupervisedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error)` - Runs supervised action with full governance lifecycle
- `func (g *SupervisedAction) Metadata() core.ActionMetadata` - Delegates to inner action's Metadata method
- `func (g *SupervisedAction) shouldBlock(err error) bool` - Determines if action should be blocked based on error and failure mode
- `func (g *SupervisedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error)` - Contains actual execution logic orchestrating phases
- `func (g *SupervisedAction) performAssessment(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) error` - Executes pre-check phase
- `func (g *SupervisedAction) injectDynamicConfig(ctx context.Context, logger core.Logger, input *core.Envelope)` - Queries engine for configuration overrides
- `func (g *SupervisedAction) executeAction(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) (core.Envelope, error)` - Runs inner action with context propagation
- `func (g *SupervisedAction) performReflection(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) (core.Envelope, error)` - Executes post-check phase
- `func (g *SupervisedAction) applySteering(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) core.Envelope` - Evaluates steering decisions and stamps metadata
- `func (g *SupervisedAction) isSensitive(labels []string) bool` - Checks if input envelope contains sensitive security labels

### Component: adapters/ai
**1. Responsibilities:**
- Wraps Genkit AI models as core.Action
- Handles LLM generation and streaming
- Maps core.Envelope to Genkit-compatible prompts

**2. Core Structs:**
- **LLMAction**: Implements core.Action for Genkit models
- **genkitAdapter**: Wrapper for genkit.Genkit

**3. Key Functions:**
- `func NewLLMAction(name string, generator core.TextGenerator) *LLMAction` - Creates a new AI action

### Component: adapters/extractor
**1. Responsibilities:**
- Uses an LLM to extract structured data from text
- Validates output against a generated JSON Schema

**2. Core Structs:**
- **ExtractorAction**: Wraps an LLM Action for extraction

**3. Key Functions:**
- `func New(name string, generator core.Action, schema any) (*ExtractorAction, error)` - Creates an extractor for a target struct
- `func (e *ExtractorAction) Execute(ctx, input) (Envelope, error)` - Extracts data and returns a struct envelope

### Component: adapters/func
**1. Responsibilities:**
- Adapts any Go function (generic func(In) (Out, error)) into a core.Action
- Handles type assertion and envelope wrapping

**2. Core Structs:**
- **Wrapper[In, Out]**: Generic wrapper struct

**3. Key Functions:**
- `func NewWrapper[In, Out](name string, fn ToolFunc[In, Out]) *Wrapper[In, Out]` - Wraps a Go function
- `func (w *Wrapper[In, Out]) Execute(ctx, input) (Envelope, error)` - Invokes function

### Component: adapters/knowledge
**1. Responsibilities:**
- Loads Knowledge Graphs (RDF/Turtle/N-Quads) into Datalog facts
- Parses external graph files

**2. Core Structs:**
- **RDFLoader**: Handles file parsing using knakk/rdf

**3. Key Functions:**
- `func (l *RDFLoader) Parse(path string) ([]string, error)` - Parses RDF file to 'triple(S,P,O)' facts

### Component: adapters/vector
**1. Responsibilities:**
- Abstracts Vector DB operations (Search/Upsert)
- Provides RAG capabilities via RetrieverAction

**2. Core Structs:**
- **RetrieverAction**: Wraps a DocumentRetriever
- **Document**: Standard snippet format

**3. Key Functions:**
- `func NewRetrieverAction(name string, retriever DocumentRetriever) *RetrieverAction` - Creates a retrieval action
- `func (r *RetrieverAction) Execute(ctx, input) (Envelope, error)` - Performs search and returns JSON doc list

### Component: adapters/mcp
**1. Responsibilities:**
- Connects to Model Context Protocol servers
- Discovers tools and exposes them as core.Action

**2. Core Structs:**
- **Loader**: Handles MCP connection and tool discovery
- **MCPAction**: Wraps an MCP tool

**3. Key Functions:**
- `func (l *Loader) Load(ctx) ([]Action, error)` - Connects to MCP and returns actions

### Component: adapters/logger
**1. Responsibilities:**
- Provides logger adapter implementations

**2. Core Structs:**
- **ZapAdapter**: Zap logger implementation

**3. Key Functions:**
- (Implementation details in zap_adapter.go)

### Component: adapters/resilience
**1. Responsibilities:**
- Implements circuit breaker pattern for resilience

**2. Core Structs:**
- **CircuitBreaker**: Circuit breaker implementation

**3. Key Functions:**
- `func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker` - Creates new circuit breaker
- `func (cb *CircuitBreaker) Execute(action core.Action) (core.Envelope, error)` - Executes action with circuit breaking

### Component: adapters/memory
**1. Responsibilities:**
- Provides memory adapter implementations for different backends

**2. Core Structs:**
- (Implementation details in memory package)

### Component: providers/google
**1. Responsibilities:**
- Initializes Google GenAI (Gemini) plugin
- Implements a Proxy Pattern to handle configuration issues in the official plugin

**2. Core Structs:**
- N/A (functional initialization)

**3. Key Functions:**
- `func Init(ctx, globalG, apiKey, modelName) (string, error)` - Registers Google model and returns global name

### Component: providers/openai
**1. Responsibilities:**
- Initializes OpenAI plugin for LLM generation
- Provides OpenAI-compatible embedding support

**2. Core Structs:**
- **OpenAIEmbedder**: Implements core.Embedder using OpenAI's embedding API

**3. Key Functions:**
- `func NewEmbedder(apiKey, baseURL, modelName string) (*OpenAIEmbedder, error)` - Creates new embedder with OpenAI API
- `func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error)` - Generates vector for single text
- `func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)` - Generates vectors for multiple texts
- `func (e *OpenAIEmbedder) Dimension() int` - Returns vector dimension (1536 for text-embedding-3-small, 3072 for text-embedding-3-large)

### Component: providers/memory
**1. Responsibilities:**
- Provides memory provider implementations

**2. Core Structs:**
- (Implementation details in memory package)

### Component: config
**1. Responsibilities:**
- Configuration loading from YAML files
- Schema validation

**2. Core Structs:**
- **Config**: Configuration structure
- **PolicyConfig**: Policy configuration section
- **ObservabilityConfig**: Observability configuration section

**3. Key Functions:**
- `func Load(path string) (*Config, error)` - Loads configuration from YAML file
- `func (c *Config) Validate() error` - Validates configuration against schema

---

## 3. CRITICAL PATH & DATA (The Flow)

### Execution Sequence (High Level)

```mermaid
sequenceDiagram
    participant User
    participant SDK as sdk.Client
    participant Loop as RunLoop
    participant Sup as Supervisor
    participant Eng as Engine (Datalog)
    participant Act as Adapter/Action

    User->>SDK: ExecuteByName("chat", payload)
    SDK->>Loop: runLoopInternal()
    
    loop Semantic Loop
        Loop->>Sup: Action.Execute(Envelope)
        
        rect rgb(240, 240, 240)
            note right of Sup: Governance Lifecycle
            Sup->>Eng: Assess(Input) [Pre-Check]
            Eng-->>Sup: OK / Block
            Sup->>Act: Execute(Input)
            Act-->>Sup: Result
            Sup->>Eng: Reflect(Result) [Post-Check]
            Eng-->>Sup: OK / Block
            Sup->>Eng: EvaluateSteering(Result)
            Eng-->>Sup: Decision (Retry/Route)
        end
        
        Sup-->>Loop: Envelope + Metadata
    
        alt Decision == RETRY
            Loop->>Loop: Increment Retry Count
            Loop->>Loop: Inject Feedback
            note right of Loop: Backoff & Repeat
        else Decision == ROUTE
            Loop->>Loop: Switch Action Target
        else Decision == PROCEED
            Loop-->>SDK: Result
        end
    end
    SDK-->>User: Final Result
```

### Data Transformation Flow

```mermaid
flowchart LR
    Input[User Input Struct/JSON] --> Env[core.Envelope]
    Env --> Reflector{Reflection Engine}
    
    Reflector - TypeStruct --> StructToFacts[internal/engine/reflection.go]
    Reflector - TypeJSON --> Flattener[internal/engine/flattener.go]
    
    StructToFacts --> Facts[(Datalog Facts)]
    Flattener --> Facts
    
    Facts --> Solver[Mangle Runtime]
    Policies[std.dl / User Policies] --> Solver
    
    Solver --> QueryHalt{Halt/Deny?}
    QueryHalt -- Yes --> Block[AlignmentError]
    QueryHalt -- No --> Action[Execute Action]
    
    Action --> Result[Result Envelope]
    Result --> ReflectPostCheck{Reflect Check}
    ReflectPostCheck -- Fail --> Block
    ReflectPostCheck -- Pass --> Steering{Evaluate Steering}
    
    Steering --> Decision[Decision: RETRY/ROUTE/PROCEED]
    Decision --> Output[Final Output]
```

---

## 4. SOURCE CODE DUMP

### internal/engine/resources/std.dl
```datalog
% Standard Vocabulary for Manglekit v1.0
% This file defines the core predicates and rules used across the system

% --- Core Predicates ---

% Action Registration
% action_meta(ActionID, Name, Type, InputType, OutputType)
action_meta(ID, Name, Type, InputType, OutputType) :-
    registered_action(ID, Name, Type, InputType, OutputType).

% Execution Context
% execution_context(ActionID, RetryCount, FeedbackHistory)
execution_context(ID, RetryCount, Feedback) :-
    current_execution(ID, RetryCount, Feedback).

% Envelope Metadata
% envelope_meta(EnvID, Key, Value)
envelope_meta(ID, Key, Value) :-
    envelope(ID), envelope_field(ID, Key, Value).

% Security Labels
% has_label(EnvID, Label)
has_label(ID, Label) :-
    envelope(ID), envelope_label(ID, Label).

% Facts Injection
% fact(FactString)
fact(F) :-
    injected_fact(F).

% --- Decision Predicates ---

% Decision outcomes
decision_outcome(EnvID, Outcome) :-
    decision(EnvID, Outcome, _, _).

decision_target(EnvID, Target) :-
    decision(EnvID, _, Target, _).

decision_reason(EnvID, Reason) :-
    decision(EnvID, _, _, Reason).

% --- Gate Predicates ---

% Pre-check gates
allow_action(ActionID, EnvID) :-
    action_meta(ActionID, _, _, _, _),
    envelope_meta(EnvID, _, _),
    not deny_action(ActionID, EnvID).

deny_action(ActionID, EnvID) :-
    action_meta(ActionID, _, _, _, _),
    envelope_meta(EnvID, _, _),
    violation(ActionID, EnvID).

% Post-check gates
allow_output(ActionID, EnvID) :-
    action_meta(ActionID, _, _, _, _),
    envelope_meta(EnvID, _, _),
    not deny_output(ActionID, EnvID).

deny_output(ActionID, EnvID) :-
    action_meta(ActionID, _, _, _, _),
    envelope_meta(EnvID, _, _),
    output_violation(ActionID, EnvID).

% --- Steering Predicates ---

% Steering decisions
steering_decision(EnvID, Decision) :-
    decision(EnvID, _, Decision, _).

% Retry conditions
should_retry(EnvID) :-
    steering_decision(EnvID, "RETRY").

should_retry(EnvID) :-
    execution_context(_, RetryCount, _),
    RetryCount < 3,
    steering_decision(EnvID, "RETRY").

% Route conditions
should_route(EnvID, TargetAction) :-
    steering_decision(EnvID, "ROUTE"),
    decision_target(EnvID, TargetAction).

% --- Helper Predicates ---

% Check for specific metadata value
has_meta_value(EnvID, Key, Value) :-
    envelope_meta(EnvID, Key, Value).

% Check for specific label
has_label_value(EnvID, Label) :-
    has_label(EnvID, Label).

% Count labels
label_count(EnvID, Count) :-
    aggregate(Count, has_label(EnvID, _)).
```

### internal/engine/resources/planner.dl
```datalog
% Planner Rules for Manglekit v1.0
% Defines rules for action planning and routing

% --- Action Selection ---

% Select default action if no specific action is requested
select_action("default") :-
    not requested_action(_).

% Use requested action if specified
select_action(Action) :-
    requested_action(Action).

% --- Route Planning ---

% Route to fallback action if primary fails
route_to_fallback(PrimaryAction, FallbackAction) :-
    action_meta(PrimaryAction, _, _, _, _),
    action_meta(FallbackAction, _, _, _, _),
    fallback_mapping(PrimaryAction, FallbackAction).

% --- Retry Planning ---

% Calculate backoff delay based on retry count
backoff_delay(RetryCount, Delay) :-
    RetryCount >= 0,
    Delay = 2 ^ RetryCount.

% --- Context Planning ---

% Include history in context if requested
include_history(EnvID) :-
    requested_feature("history_inclusion"),
    envelope(EnvID).

% Include RAG context if requested
include_rag(EnvID) :-
    requested_feature("rag_context"),
    envelope(EnvID).

% --- Priority Planning ---

% Higher priority actions execute first
action_priority(Action, Priority) :-
    action_meta(Action, _, _, _, _),
    priority_mapping(Action, Priority).

% Default priority is 0
action_priority(Action, 0) :-
    action_meta(Action, _, _, _, _),
    not priority_mapping(Action, _).
```

### internal/supervisor/supervisor.go
```go
package supervisor

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"go.opentelemetry.io/otel/trace"
)

// SupervisedAction wraps a core.Action with governance lifecycle.
// It implements the "Guarded Action" pattern: Trace -> Assess -> Execute -> Reflect
type SupervisedAction struct {
	inner        core.Action
	engine        core.Evaluator
	tracer       trace.Tracer
	failureMode   string
}

// NewSupervisedAction creates a new SupervisedAction with default settings.
func NewSupervisedAction(action core.Action, eng core.Evaluator, failureMode string) *SupervisedAction {
	return &SupervisedAction{
		inner:      action,
		engine:      eng,
		tracer:     nil,
		failureMode: failureMode,
	}
}

// NewSupervisedActionWithTracer creates a new SupervisedAction with tracing enabled.
func NewSupervisedActionWithTracer(action core.Action, eng core.Evaluator, tracer trace.Tracer, failureMode string) *SupervisedAction {
	return &SupervisedAction{
		inner:      action,
		engine:      eng,
		tracer:     tracer,
		failureMode: failureMode,
	}
}

// Execute runs the supervised action with full governance lifecycle.
func (g *SupervisedAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	logger := g.engine.Logger()
	meta := g.inner.Metadata()

	// Start tracing span if tracer is available
	if g.tracer != nil {
		ctx, span := g.tracer.Start(ctx, fmt.Sprintf("SupervisedAction.Execute:%s", meta.Name))
		defer span.End()
	}

	// Execute internal governance lifecycle
	result, err := g.executeInternal(ctx, input)
	if err != nil {
		logger.Error(ctx, "supervised action failed", "action", meta.Name, "error", err.Error())
		return core.Envelope{}, err
	}

	return result, nil
}

// Metadata returns the action's metadata.
func (g *SupervisedAction) Metadata() core.ActionMetadata {
	return g.inner.Metadata()
}

// executeInternal contains the actual execution logic with all governance phases.
func (g *SupervisedAction) executeInternal(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	logger := g.engine.Logger()
	meta := g.inner.Metadata()

	// Phase 1: Inject dynamic configuration from policy engine
	g.injectDynamicConfig(ctx, logger, &input)

	// Phase 2: Pre-check (Assess)
	if err := g.performAssessment(ctx, logger, meta, input); err != nil {
		return core.Envelope{}, err
	}

	// Phase 3: Execute the inner action
	result, err := g.executeAction(ctx, logger, meta, input)
	if err != nil {
		return core.Envelope{}, err
	}

	// Phase 4: Post-check (Reflect)
	validatedResult, err := g.performReflection(ctx, logger, meta, result)
	if err != nil {
		return core.Envelope{}, err
	}

	// Phase 5: Apply steering decisions
	finalResult := g.applySteering(ctx, logger, meta, validatedResult)

	return finalResult, nil
}

// injectDynamicConfig queries the policy engine for configuration overrides.
func (g *SupervisedAction) injectDynamicConfig(ctx context.Context, logger core.Logger, input *core.Envelope) {
	config, err := g.engine.GetActionConfig(ctx, *input)
	if err != nil {
		logger.Warn(ctx, "failed to get action config", "error", err.Error())
		return
	}

	// Apply configuration overrides to input envelope
	for key, value := range config {
		input.SetMeta(key, value)
	}
}

// performAssessment executes the pre-check phase.
func (g *SupervisedAction) performAssessment(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) error {
	if err := g.engine.Assess(ctx, meta, input); err != nil {
		if g.shouldBlock(err) {
			msg := "assessment failed"
			if core.IsInputError(err) {
				msg = "assessment blocked due to invalid input"
			}
			logger.Warn(msg, "action", meta.Name, "error", err.Error())
			return fmt.Errorf("%s: %w", msg, err)
		}

		logger.Warn("engine assessment failed but Fail-Open active. Proceeding.", "error", err.Error())
	}
	return nil
}

// executeAction runs the inner action with context propagation.
func (g *SupervisedAction) executeAction(ctx context.Context, logger core.Logger, meta core.ActionMetadata, input core.Envelope) (core.Envelope, error) {
	logger.Debug(ctx, "executing inner action", "action", meta.Name)
	return g.inner.Execute(ctx, input)
}

// performReflection executes the post-check phase.
func (g *SupervisedAction) performReflection(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) (core.Envelope, error) {
	validatedResult, err := g.engine.Reflect(ctx, meta, result)
	if err != nil {
		if g.shouldBlock(err) {
			msg := "reflection failed"
			if core.IsInputError(err) {
				msg = "reflection blocked due to invalid input"
			}
			logger.Warn(msg, "action", meta.Name, "error", err.Error())
			return core.Envelope{}, fmt.Errorf("%s: %w", msg, err)
		}

		logger.Warn("engine reflection failed but Fail-Open active. Proceeding.", "error", err)
		validatedResult = result
	}
	return validatedResult, nil
}

// applySteering evaluates steering decisions and stamps metadata.
func (g *SupervisedAction) applySteering(ctx context.Context, logger core.Logger, meta core.ActionMetadata, result core.Envelope) core.Envelope {
	decision, steeringMeta, err := g.engine.EvaluateSteering(ctx, result)
	if err != nil {
		logger.Warn("steering evaluation failed", "action", meta.Name, "error", err.Error())
		decision = ""
		steeringMeta = nil
	}

	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata[core.KeyDecision] = decision
	for k, v := range steeringMeta {
		result.Metadata[k] = v
	}

	return result
}

// shouldBlock determines if action should be blocked based on error and failure mode.
func (g *SupervisedAction) shouldBlock(err error) bool {
	// Fail-Open mode: proceed on errors
	if g.failureMode == "fail-open" {
		return false
	}
	// Fail-Closed mode: block on any error
	return true
}

// isSensitive checks if input envelope contains sensitive security labels.
func (g *SupervisedAction) isSensitive(labels []string) bool {
	sensitiveTags := []string{"pii", "secret", "confidential", "auth_token"}
	for _, l := range labels {
		for _, tag := range sensitiveTags {
			if l == tag {
				return true
			}
		}
	}
	return false
}
```

---

## 5. CHANGELOG

### [2026-01-01 07:50:00Z] - Attention Sink Design Document Added
Created comprehensive design document for Attention Sink mechanism in `docs/designs/attention_sink.md`:
- **Architecture Design**: Three-tier memory hierarchy (Hot/Warm/Cold) with HNSW-based semantic retrieval
- **Component Specifications**: AttentionManager, CandidateGatherer, MultiCriteriaScorer, ContextCompressor
- **Integration Points**: SDK Loop integration, Policy Engine Datalog rules, Configuration schema
- **Go Implementation**: Detailed Go code examples aligned with Manglekit's architectural patterns
- **Testing Strategy**: Unit, integration, and performance testing approaches
- **Migration Path**: Phased implementation plan from core components to advanced features

### [2026-01-01 07:15:00Z] - Kernel Resync Complete
Full regeneration of CONTEXT.md with comprehensive codebase analysis:
- **File Map Updated**: Complete directory tree with directory purpose annotations
- **Components Documented**: All packages analyzed with responsibilities, structs, and key functions
- **Critical Path & Data Flow**: Mermaid diagrams for execution sequence and data transformation
- **Source Code Dump**: Complete source code from core, sdk, internal/engine and internal/supervisor
- **Standard Library**: std.dl and planner.dl documented
- **Core Contracts**: All core interfaces and types fully documented
- **Engine Logic**: MangleRuntime, PolicyEngine, reflection, and flattening logic captured
- **Supervisor Pattern**: Full SupervisedAction lifecycle documented
- **SDK Loop**: Semantic State Machine and retry logic documented

**Files Analyzed:**
- core/types.go, core/logic.go, core/data.go, core/governance.go
- sdk/client.go, sdk/loop.go
- internal/engine/runtime.go, internal/engine/solver.go, internal/engine/reflection.go, internal/engine/flattener.go
- internal/supervisor/supervisor.go
- internal/engine/resources/std.dl, internal/engine/resources/planner.dl

### [2026-01-02 07:15:00Z] - OpenAI Embedder Added
Added OpenAI-compatible embedding support to the providers/openai package:
- **New File**: `providers/openai/embedder.go` - Implements core.Embedder interface using OpenAI's embedding API
- **Features**: 
  - Supports OpenAI embedding models (text-embedding-3-small, text-embedding-3-large, text-embedding-ada-002)
  - Configurable API key and base URL for OpenAI-compatible endpoints
  - Batch embedding support for efficiency
  - Vector dimensions: 1536 (text-embedding-3-small), 3072 (text-embedding-3-large)
- **Example Updated**: `examples/hybrid_rag/main.go` migrated from Google GenAI embedder to OpenAI embedder
  - Environment variable changed from `GOOGLE_API_KEY` to `OPENAI_API_KEY`
  - Default model changed from "text-embedding-004" to "text-embedding-3-small"
  - Import path updated from `providers/google` to `providers/openai`

---

## 6. KNOWN GAPS & LIMITATIONS

- **PII Detection**: Full string matching for PII keywords is not supported in Mangle v0.3.0. The `string_contains` function is not available, limiting automated PII detection capabilities.
- **Egress Policy**: Egress policies are not being evaluated in test scenarios because access control policy is blocking requests first. The security labels are being propagated correctly, but egress check happens after access control.
- **Retry Logic**: The RETRY decision is defined in the standard vocabulary, but the actual retry mechanism with feedback injection needs to be implemented in the RunLoop.
- **Lineage Tracking**: Explicit lineage recording is minimal; currently handled via context propagation and tracing spans rather than explicit in-memory storage.
- **Memory Persistence**: Durable state manager exists but full implementation details need verification.
- **Attention Sink Implementation**: Design document created but implementation not yet started. Requires HNSW adapter, embedder integration, and SDK loop modifications.

---

## 7. DEPENDENCY RULES

- `supervisor` depends on `engine` and `core`.
- `engine` depends on `core` and `google/mangle`.
- `adapters` depend on `core` and external drivers.
- `core` has NO dependencies.

---

## 8. EXAMPLES & PATTERNS

### Example: Transitive Access Control
```prolog
% User can access Doc if:
% - User is a member of a Group
% - Group owns a Project
% - Project contains Doc
can_access(User, Doc) :-
    member_of(User, Group),
    owns(Group, Project),
    contains(Project, Doc).
```

### Example: Security Label Propagation
```go
// CustomHybridMemory.RecallWithFacts() injects security labels
switch id {
case "doc_project_x", "doc_project_x_spec":
    securityLabels = append(securityLabels, "TOP_SECRET")
case "doc_project_y":
    securityLabels = append(securityLabels, "CONFIDENTIAL")
case "doc_remote_work":
    securityLabels = append(securityLabels, "PUBLIC")
}
```

---

## 9. TESTING NOTES

- Use `go run examples/hybrid_rag/main.go` to test enhanced system
- The MockEmbedder returns `{0.9, 0.1}` for queries containing "launch" or "Project X"
- All test scenarios use the same query "What are the launch codes for Project X?" to test access control
- Set `OPENAI_API_KEY` environment variable to use real OpenAI embeddings
