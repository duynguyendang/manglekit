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
	"github.com/duynguyendang/manglekit/internal/telemetry"
)

// ClientOption configures the Manglekit Client during initialization.
type ClientOption func(*Client) error

// WithEngine allows injecting a custom or mock core.Evaluator.
func WithEngine(e core.Evaluator) ClientOption {
	return func(c *Client) error {
		c.engine = e
		return nil
	}
}

// WithBlueprintPath specifies the file path to load Datalog rules from.
// "Blueprint" is the new terminology for "Policy".
//
// Parameters:
//   - path: A file path to the .dl blueprint file.
func WithBlueprintPath(path string) ClientOption {
	return func(c *Client) error {
		c.blueprintPath = path

		// 1. JIT Init (Critical fix)
		if err := ensureDependencies(c); err != nil {
			return err
		}

		// 2. Load Policy immediately
		// Ensure "os" and "context" are imported!
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read blueprint %q: %w", path, err)
		}
		return c.engine.LoadPolicy(context.Background(), string(content))
	}
}

// WithFailMode sets the resilience strategy for the client.
//
// Parameters:
//   - mode: "open" (allow execution on error) or "closed" (block execution on error).
func WithFailMode(mode string) ClientOption {
	return func(c *Client) error {
		c.failureMode = mode
		return nil
	}
}

// WithLogger sets a custom logger for the client.
//
// Parameters:
//   - l: A core.Logger implementation.
func WithLogger(l core.Logger) ClientOption {
	return func(c *Client) error {
		if l != nil {
			c.logger = l
		}
		return nil
	}
}

// WithAgentMemory configures the semantic memory (RAG) provider.
//
// Parameters:
//   - mem: A core.AgentMemory implementation.
func WithAgentMemory(mem core.AgentMemory) ClientOption {
	return func(c *Client) error {
		if mem != nil {
			c.agentMemory = mem
		}
		return nil
	}
}

// WithTracerProvider configures the OpenTelemetry tracer provider.
// This enables Manglekit to emit spans to your existing tracing infrastructure.
//
// Parameters:
//   - tp: The OpenTelemetry TracerProvider.
func WithTracerProvider(tp trace.TracerProvider) ClientOption {
	return func(c *Client) error {
		if tp != nil {
			// Initialize the tracer immediately so it's available for JIT init later
			c.otelTracer = tp.Tracer(TracerName)
			c.tracer = telemetry.NewOTelTracer(c.otelTracer)
		}
		return nil
	}
}

// WithHistory configures a custom persistence store for chat history.
//
// Parameters:
//   - store: A core.HistoryStore implementation (e.g., Redis backed).
func WithHistory(store core.HistoryStore) ClientOption {
	return func(c *Client) error {
		if store == nil {
			return nil
		}
		// If agentMemory is already a HybridMemory, update its History component.
		if hm, ok := c.agentMemory.(*HybridMemory); ok {
			hm.History = store
		} else {
			// Otherwise wrap it in a new HybridMemory (losing previous non-hybrid memory if any)
			c.agentMemory = NewHybridMemory(store, core.NopVectorStore{}, core.NopEmbedder{})
		}
		return nil
	}
}

// WithMemory allows injecting a custom memory implementation (e.g., Hybrid HNSW).
func WithMemory(mem core.AgentMemory) ClientOption {
	return func(c *Client) error {
		c.agentMemory = mem
		return nil
	}
}

// WithLLM configures the AI backend for the client.
// This supports the "Explicit AI Adapter" pattern where the application initializes the model
// (e.g., via Genkit) and passes it to the SDK.
//
// Parameters:
//   - gen: A core.TextGenerator implementation (e.g., adapters.ai.NewGenkitAdapter(model)).
func WithLLM(gen core.TextGenerator) ClientOption {
	return func(c *Client) error {
		if gen != nil {
			c.llm = gen
		}
		return nil
	}
}

// WithConfig applies settings from a loaded configuration struct.
// It handles wiring of providers, actions, policy, knowledge, and memory.
func WithConfig(cfg *config.Config) ClientOption {
	return func(c *Client) error {
		if cfg == nil {
			return nil
		}

		// 1. Update Logger if specified
		if cfg.Observability.LogLevel != "" {
			c.logger = logger.New(cfg.Observability.LogLevel)
		}

		// Ensure dependencies (Engine/Tracer) now that logger is set
		if err := ensureDependencies(c); err != nil {
			return err
		}

		// 2. Load Policy (Blueprint)
		if cfg.Policy.Path != "" {
			content, err := os.ReadFile(cfg.Policy.Path)
			if err != nil {
				return fmt.Errorf("failed to read policy file %q: %w", cfg.Policy.Path, err)
			}
			// Use background context as option doesn't provide one
			if err := c.engine.LoadPolicy(context.Background(), string(content)); err != nil {
				return fmt.Errorf("failed to load policy from %q: %w", cfg.Policy.Path, err)
			}
		}

		// 3. Load Knowledge
		if cfg.Knowledge.Path != "" {
			path := cfg.Knowledge.Path
			var facts []string
			var err error

			if strings.HasSuffix(path, ".nt") {
				f, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open knowledge file %q: %w", path, err)
				}
				loader := knowledge.NewNTriplesLoader()
				facts, err = loader.Parse(f)
				f.Close()
			} else {
				loader := knowledge.NewRDFLoader()
				facts, err = loader.Parse(path)
			}

			if err != nil {
				return fmt.Errorf("failed to parse knowledge from %q: %w", path, err)
			}

			if err := c.engine.LoadFacts(facts); err != nil {
				return fmt.Errorf("failed to load knowledge facts from %q: %w", path, err)
			}
		}

		// 4. Hydrate Memory
		if cfg.Memory.Provider != "" {
			// Assuming createMemory helper exists or we use registry
			// We can use the MemoryRegistry from registry.go
			factory, err := MemoryProvider(cfg.Memory.Provider)
			if err != nil {
				return fmt.Errorf("memory provider %q not found: %w", cfg.Memory.Provider, err)
			}
			mem, err := factory(context.Background(), cfg.Memory)
			if err != nil {
				return fmt.Errorf("failed to create memory: %w", err)
			}
			c.agentMemory = mem
		}

		// 5. Hydrate Actions
		// Loops through cfg.Actions to register supervised actions using WithProviderConfig logic.
		// Note: The configuration schema does not currently have a separate "LLM" section.
		// LLMs are configured as actions with Type="llm". The WithProviderConfig option
		// automatically detects this and sets the client's default LLM if applicable.
		for name, actionCfg := range cfg.Actions {
			opt := WithProviderConfig(name, actionCfg)
			if err := opt(c); err != nil {
				return err
			}
		}

		// 6. Load MCP Actions
		if len(cfg.MCP) > 0 {
			for _, mcpCfg := range cfg.MCP {
				loader := mcpAdapter.NewLoader(mcpCfg).WithLogger(c.logger)
				actions, err := loader.Load(context.Background())
				if err != nil {
					if mcpCfg.FailOnStartup {
						return fmt.Errorf("critical tool '%s' failed to load: %w", mcpCfg.Name, err)
					}
					c.logger.Warn("MCP tool failed to load", "tool", mcpCfg.Name, "error", err)
					continue
				}

				for _, action := range actions {
					safeAction := c.Supervise(action)
					c.registry[safeAction.Metadata().Name] = safeAction
					c.logger.Info("Discovered MCP Tool", "name", safeAction.Metadata().Name)
				}
			}
		}

		// 7. Failure Mode
		if cfg.FailureMode != "" {
			c.failureMode = cfg.FailureMode
		}

		return nil
	}
}

// ExecutionParams holds the configuration for a specific execution run.
type ExecutionParams struct {
	// SessionID is the unique identifier for a conversation/session.
	SessionID string
	// MemoryMode determines how chat history is handled (None, Transient, Persist).
	MemoryMode core.MemoryMode
	// Metadata contains additional context to be injected into the execution envelope.
	Metadata map[string]string

	// State fields (Managed by ExecuteSingleStep/Loop)
	Store           core.HistoryStore `json:"-"` // Internal store reference
	CurrentHistory  []core.Message    `json:"history,omitempty"`
	FeedbackHistory []string          `json:"feedback_history,omitempty"`
	LastFeedback    string            `json:"last_feedback,omitempty"`
	RetryCount      int               `json:"retry_count,omitempty"`
}

// ExecuteOption configures a single execution call (e.g., ExecuteByName).
type ExecuteOption func(*ExecutionParams)

// WithSessionID activates persistent stateful mode for the execution.
// It links the execution to a specific session history.
//
// Parameters:
//   - id: The session identifier.
func WithSessionID(id string) ExecuteOption {
	return func(p *ExecutionParams) {
		p.SessionID = id
		p.MemoryMode = core.MemoryModePersist
	}
}

// WithTransientMemory activates in-memory stateful mode.
// History is tracked for the duration of the loop/process but not persisted.
func WithTransientMemory() ExecuteOption {
	return func(p *ExecutionParams) {
		p.MemoryMode = core.MemoryModeTransient
	}
}

// WithMetadata injects custom key-value pairs into the execution envelope's metadata.
// This is useful for passing context like "user_id" or "source" to the policy engine.
//
// Parameters:
//   - key: The metadata key.
//   - value: The metadata value.
func WithMetadata(key, value string) ExecuteOption {
	return func(p *ExecutionParams) {
		if p.Metadata == nil {
			p.Metadata = make(map[string]string)
		}
		p.Metadata[key] = value
	}
}

// WithTemperature sets the sampling temperature.
func WithTemperature(t float64) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.Temperature = t
	}
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(n int) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.MaxTokens = n
	}
}

// WithTopP sets the nucleus sampling probability.
func WithTopP(p float64) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.TopP = p
	}
}

// WithStopSequences sets the stop sequences.
func WithStopSequences(seqs []string) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.StopSequences = seqs
	}
}

// WithModel sets the model to use.
func WithModel(m string) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.Model = m
	}
}

// WithJSONMode enables JSON mode.
func WithJSONMode(enabled bool) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.JSONMode = enabled
	}
}

// WithStructuredOutput sets the output type for structured generation.
func WithStructuredOutput(schema any) core.GenerateOption {
	return func(cfg *core.GenerationConfig) {
		cfg.OutputType = schema
	}
}

// WithMetadataMap injects a map of custom key-value pairs into the execution envelope's metadata.
func WithMetadataMap(meta map[string]any) ExecuteOption {
	return func(p *ExecutionParams) {
		if p.Metadata == nil {
			p.Metadata = make(map[string]string)
		}
		for k, v := range meta {
			if s, ok := v.(string); ok {
				p.Metadata[k] = s
			} else {
				p.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}
	}
}

// ensureDependencies initializes defaults for the Engine and Tracer if they haven't been injected.
func ensureDependencies(c *Client) error {
	// 1. Ensure Tracer
	if c.tracer == nil {
		if c.otelTracer == nil {
			c.otelTracer = trace.NewNoopTracerProvider().Tracer(TracerName)
		}
		c.tracer = telemetry.NewOTelTracer(c.otelTracer)
	}

	// 2. Ensure Engine
	if c.engine == nil {
		eng, err := engine.NewWithObservability(c.tracer, c.logger)
		if err != nil {
			return fmt.Errorf("failed to initialize default engine: %w", err)
		}
		c.engine = eng
	}
	return nil
}
