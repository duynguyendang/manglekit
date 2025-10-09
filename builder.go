package manglekit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/google/generative-ai-go/genai"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	googleapi "google.golang.org/api/option"
)

// BuilderAPI provides a fluent, chainable interface for constructing a MangleKit
// Orchestrator. It is the recommended way to assemble a pipeline, as it handles
// dependency injection, component construction, and configuration resolution
// from multiple sources (programmatic, YAML, environment variables).
type BuilderAPI interface {
	// WithConfig sets the base configuration from a Config object. This is typically
	// used when loading settings from a YAML file. Settings provided by other
	// builder methods (e.g., `WithLLM`) will override values from this config.
	WithConfig(cfg *Config) BuilderAPI
	// WithRetriever configures the retriever component. The `opts` parameter should
	// be a pointer to a provider-specific options struct (e.g., `retrieve.BM25Options`).
	// The builder infers the provider name from the options type.
	WithRetriever(opts any) BuilderAPI
	// WithVectorStore configures the vector store component, which is a dependency
	// for dense and hybrid retrievers. The `opts` parameter should be a pointer to
	// a provider-specific options struct (e.g., `core.LocalvecOptions`).
	WithVectorStore(opts any) BuilderAPI
	// WithReranker configures the reranker component. The `opts` parameter should
	// be a pointer to a provider-specific options struct (e.g., `rerank.CosineOptions`).
	WithReranker(opts any) BuilderAPI
	// WithRules configures the rules engine component. The `opts` parameter should
	// be a pointer to a provider-specific options struct (e.g., `core.MangleOptions`).
	WithRules(opts any) BuilderAPI
	// WithLLM configures the language model client. The `opts` parameter should
	// be a pointer to a provider-specific options struct (e.g., `llm.OpenAIOptions`).
	WithLLM(opts any) BuilderAPI
	// WithEmbedder configures the text embedding model. The `opts` parameter can
	// be either a pointer to a provider-specific options struct (e.g., `embed.GoogleEmbedderOptions`)
	// or a pre-constructed `ai.Embedder` instance.
	WithEmbedder(opts any) BuilderAPI
	// WithFlow sets the name of the flow to be executed by the declarative
	// orchestrator. This is only used if the orchestrator type is "declarative".
	WithFlow(name string) BuilderAPI
	// WithTopK sets the default number of documents to retrieve. This can be
	// overridden by provider-specific settings.
	WithTopK(k int) BuilderAPI
	// WithMaxTokens sets the default maximum number of tokens for the LLM response.
	// This can be overridden by provider-specific settings.
	WithMaxTokens(n int) BuilderAPI
	// WithObservability sets the observability hooks (logger, tracer, meter) for
	// the entire pipeline.
	WithObservability(obs core.Observability) BuilderAPI
	// WithFallbackThreshold sets the confidence score below which the pipeline
	// may exit early and return a fallback response.
	WithFallbackThreshold(f float64) BuilderAPI
	// Build is the final step in the chain. It resolves all dependencies,
	// constructs the components based on the collected configuration, and assembles
	// the final, ready-to-use core.Orchestrator. It returns an error if any
	// part of the construction fails, such as missing dependencies or invalid configuration.
	Build() (core.Orchestrator, error)
}

// builder implements the BuilderAPI. It holds the state of the configuration
// as methods are chained and performs the final build process.
type builder struct {
	opts   core.Options
	config *Config
	errs   []error
	// configDir captures the directory where a YAML configuration file was loaded
	// from. It allows relative paths in tool params to be resolved lazily during
	// buildSingleTool.
	configDir string

	// Declarative flow fields
	flowName string
	tools    map[string]any // Holds built tool instances.

	retrieverName     string
	retrieverParams   map[string]any
	rerankerName      string
	rerankerParams    map[string]any
	rulesName         string
	rulesParams       map[string]any
	llmName           string
	llmParams         map[string]any
	embedderName      string
	embedderParams    map[string]any
	vectorStoreName   string
	vectorStoreParams map[string]any

	// Internal fields for dependency management.
	clients       map[string]any
	resolvedCfgs  map[string]any
	providerNames map[string]string

	// Built components are stored here during the build process.
	embedder    ai.Embedder
	vectorStore core.VectorStore
}

type googleClients struct {
	genkit *genkit.Genkit
	genai  *genai.Client
}

// NewBuilder returns a new, empty instance of the fluent builder, ready to be
// configured. This is the entry point for programmatically constructing a
// MangleKit orchestrator.
func NewBuilder() BuilderAPI {
	return &builder{
		config:        &Config{},
		clients:       make(map[string]any),
		resolvedCfgs:  make(map[string]any),
		providerNames: make(map[string]string),
		tools:         make(map[string]any),
	}
}

// WithConfig applies a configuration object to the builder. This is the primary
// way to use settings loaded from a YAML file.
func (b *builder) WithConfig(cfg *Config) BuilderAPI {
	if cfg != nil {
		b.config = cfg
	}
	return b
}

// WithRetriever programmatically configures the retriever component for the pipeline.
// The provider name is inferred from the type of the `opts` struct.
func (b *builder) WithRetriever(opts any) BuilderAPI {
	if opts == nil {
		b.retrieverName = ""
		b.retrieverParams = nil
		return b
	}
	optsType := reflect.TypeOf(opts)
	name, ok := optionsTypeToName[optsType]
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("unregistered options type for retriever: %T", opts))
		return b
	}
	b.retrieverName = name
	b.retrieverParams = map[string]any{"typedConfig": opts}
	return b
}

// WithVectorStore programmatically configures the vector store, which is a dependency
// for dense and hybrid retrievers. The provider name is inferred from the `opts` type.
func (b *builder) WithVectorStore(opts any) BuilderAPI {
	if opts == nil {
		b.vectorStoreName = ""
		b.vectorStoreParams = nil
		return b
	}
	optsType := reflect.TypeOf(opts)
	name, ok := optionsTypeToName[optsType]
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("unregistered options type for vector store: %T", opts))
		return b
	}
	b.vectorStoreName = name
	b.vectorStoreParams = map[string]any{"typedConfig": opts}
	return b
}

// WithReranker programmatically configures the reranker component.
// The provider name is inferred from the type of the `opts` struct.
func (b *builder) WithReranker(opts any) BuilderAPI {
	if opts == nil {
		b.rerankerName = ""
		b.rerankerParams = nil
		return b
	}
	optsType := reflect.TypeOf(opts)
	name, ok := optionsTypeToName[optsType]
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("unregistered options type for reranker: %T", opts))
		return b
	}
	b.rerankerName = name
	b.rerankerParams = map[string]any{"typedConfig": opts}
	return b
}

// WithRules programmatically configures the rules engine.
// The provider name is inferred from the type of the `opts` struct.
func (b *builder) WithRules(opts any) BuilderAPI {
	if opts == nil {
		b.rulesName = ""
		b.rulesParams = nil
		return b
	}
	optsType := reflect.TypeOf(opts)
	name, ok := optionsTypeToName[optsType]
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("unregistered options type for rules engine: %T", opts))
		return b
	}
	b.rulesName = name
	b.rulesParams = map[string]any{"typedConfig": opts}
	return b
}

// WithLLM programmatically configures the language model client.
// The provider name is inferred from the type of the `opts` struct.
func (b *builder) WithLLM(opts any) BuilderAPI {
	if opts == nil {
		b.llmName = ""
		b.llmParams = nil
		return b
	}
	optsType := reflect.TypeOf(opts)
	name, ok := optionsTypeToName[optsType]
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("unregistered options type for LLM: %T", opts))
		return b
	}
	b.llmName = name
	b.llmParams = map[string]any{"typedConfig": opts}
	return b
}

// WithFlow programmatically sets the flow name for the declarative orchestrator.
func (b *builder) WithFlow(name string) BuilderAPI {
	b.flowName = name
	return b
}

// WithEmbedder programmatically configures the text embedding model.
// The provider name is inferred from the type of the `opts` struct.
// This method also accepts a pre-constructed `ai.Embedder` instance, allowing
// for custom or externally configured embedders to be injected.
func (b *builder) WithEmbedder(opts any) BuilderAPI {
	if opts == nil {
		b.embedderName = ""
		b.embedderParams = nil
		b.embedder = nil // Clear pre-built embedder if opts is nil
		return b
	}

	// Allow injecting a pre-constructed embedder.
	if emb, ok := opts.(ai.Embedder); ok {
		b.embedder = emb
		b.embedderName = "" // No name needed, it's already built.
		b.embedderParams = nil
		return b
	}

	optsType := reflect.TypeOf(opts)
	name, ok := optionsTypeToName[optsType]
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("unregistered options type for embedder: %T", opts))
		return b
	}
	b.embedderName = name
	b.embedderParams = map[string]any{"typedConfig": opts}
	b.embedder = nil // Clear pre-built embedder if using options.
	return b
}

// WithTopK programmatically sets the default number of documents to retrieve.
func (b *builder) WithTopK(k int) BuilderAPI {
	b.opts.TopK = k
	return b
}

// WithMaxTokens programmatically sets the default maximum number of tokens for the LLM response.
func (b *builder) WithMaxTokens(n int) BuilderAPI {
	b.opts.MaxTokens = n
	return b
}

// WithObservability programmatically sets the observability hooks (logger, tracer, meter).
func (b *builder) WithObservability(obs core.Observability) BuilderAPI {
	b.opts.Obs = obs
	return b
}

// WithFallbackThreshold programmatically sets the confidence score below which a fallback is triggered.
func (b *builder) WithFallbackThreshold(f float64) BuilderAPI {
	b.opts.FallbackThreshold = f
	return b
}

// resolveProviderConfig finds the configuration for a given provider by checking params, config object, and env vars.
func (b *builder) resolveProviderConfig(providerType, providerName string) error {
	key := fmt.Sprintf("%s.%s", providerType, providerName)
	if _, exists := b.resolvedCfgs[key]; exists {
		return nil // Already resolved.
	}

	switch providerName {
	case "google":
		cfg := b.config.Providers.Google
		if cfg == nil {
			cfg = &GoogleConfig{}
		}
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("GOOGLE_API_KEY")
		}
		if apiKey == "" {
			return fmt.Errorf("missing apiKey for provider 'google': please provide it via config or GOOGLE_API_KEY env var")
		}
		g, err := genai.NewClient(context.Background(), googleapi.WithAPIKey(apiKey))
		if err != nil {
			return fmt.Errorf("failed to create genai client: %w", err)
		}
		b.clients["google"] = googleClients{
			genkit: genkit.Init(context.Background(), genkit.WithPlugins(&googlegenai.GoogleAI{APIKey: apiKey})),
			genai:  g,
		}
		b.resolvedCfgs[key] = cfg

	case "openai":
		cfg := b.config.Providers.OpenAI
		if cfg == nil {
			cfg = &OpenAIConfig{}
		}
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		if apiKey == "" {
			return fmt.Errorf("missing apiKey for provider 'openai': please provide it via config or OPENAI_API_KEY env var")
		}
		client := openai.NewClient(option.WithAPIKey(apiKey))
		b.clients["openai"] = client
		b.resolvedCfgs[key] = cfg

	case "groq":
		cfg := b.config.Providers.Groq
		if cfg == nil {
			cfg = &OpenAICompatibleConfig{}
		}
		apiKey := cfg.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("GROQ_API_KEY")
		}
		if apiKey == "" {
			return fmt.Errorf("missing apiKey for provider 'groq': please provide it via config or GROQ_API_KEY env var")
		}
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.groq.com/openai/v1"
		}
		client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL))
		b.clients["groq"] = client
		b.resolvedCfgs[key] = cfg
	}
	return nil
}

// Build constructs the final Orchestrator by resolving all dependencies and
// building the components in the correct order. It is the terminal method in
// the builder chain.
func (b *builder) Build() (core.Orchestrator, error) {
	orchestratorType := "sandwich" // default
	if b.config.Orchestrator.Type != "" {
		orchestratorType = b.config.Orchestrator.Type
	}

	if len(b.errs) > 0 {
		return nil, errors.Join(b.errs...)
	}

	switch orchestratorType {
	case "declarative":
		// The declarative orchestrator requires a rules engine.
		// Note: The main rules engine for the orchestrator is built here,
		// separate from any "rules" tools that might be defined.
		if err := b.buildRules(); err != nil {
			return nil, fmt.Errorf("failed to build rules for declarative orchestrator: %w", err)
		}
		if b.opts.Rules == nil {
			return nil, errors.New("declarative orchestrator requires a rules engine, but none was configured")
		}

		// Build all the tools defined in the config.
		if err := b.buildTools(); err != nil {
			return nil, err
		}

		// Get the flow name from the builder or the config.
		flowName := b.flowName
		if flowName == "" {
			flowName = b.config.Orchestrator.FlowName
		}

		flowController, ok := b.opts.Rules.(core.FlowController)
		if !ok {
			return nil, fmt.Errorf("rules engine for declarative orchestrator must be a FlowController, but got %T", b.opts.Rules)
		}

		return declarative.New(flowController, b.tools, flowName)

	case "sandwich":
		if err := b.resolveDependencies(); err != nil {
			return nil, err
		}
		if err := b.buildComponents(); err != nil {
			return nil, err
		}
		return New(b.opts)

	default:
		return nil, fmt.Errorf("unknown orchestrator type: %q", orchestratorType)
	}
}

// buildTools iterates through the tools defined in the config, respects dependencies,
// and builds each one. It uses an iterative approach to handle the dependency graph.
func (b *builder) buildTools() error {
	toBuild := make(map[string]ToolConfig)
	for name, cfg := range b.config.Tools {
		toBuild[name] = cfg
	}

	// Create a set of all tool names for efficient lookup.
	allToolNames := make(map[string]struct{})
	for name := range toBuild {
		allToolNames[name] = struct{}{}
	}

	for len(toBuild) > 0 {
		builtInThisPass := 0
		for name, cfg := range toBuild {
			deps := getToolDependencies(cfg, allToolNames)
			allDepsMet := true
			for _, dep := range deps {
				if _, ok := b.tools[dep]; !ok {
					allDepsMet = false
					break
				}
			}

			if allDepsMet {
				tool, err := b.buildSingleTool(name, cfg)
				if err != nil {
					return fmt.Errorf("failed to build tool %q: %w", name, err)
				}
				b.tools[name] = tool
				delete(toBuild, name)
				builtInThisPass++
			}
		}

		if builtInThisPass == 0 && len(toBuild) > 0 {
			// If we went through a whole pass without building anything, there's a circular dependency.
			var remaining []string
			for name := range toBuild {
				remaining = append(remaining, name)
			}
			return fmt.Errorf("circular dependency detected or missing tools. cannot build: %v", remaining)
		}
	}
	return nil
}

// getToolDependencies inspects a tool's parameters to find its dependencies on other tools.
// A dependency is assumed to be any string value in the first level of the params map that
// corresponds to another tool's name.
func getToolDependencies(cfg ToolConfig, allToolNames map[string]struct{}) []string {
	var deps []string
	if cfg.Params == nil {
		return deps
	}
	// This simple heuristic assumes any string value in the params map could be a dependency.
	// The build loop is robust enough to ignore strings that aren't actual tool names.
	for _, param := range cfg.Params {
		if depName, ok := param.(string); ok {
			if _, isTool := allToolNames[depName]; isTool {
				deps = append(deps, depName)
			}
		}
	}
	return deps
}

// buildSingleTool is a dispatcher that constructs a single tool instance based on its provider type.
func (b *builder) buildSingleTool(name string, cfg ToolConfig) (any, error) {
	// 1. Resolve provider-level config (e.g., API keys)
	providerFamily := cfg.Provider
	if family, ok := providerToFamily[providerFamily]; ok {
		providerFamily = family
	}
	// The resolveProviderConfig function will only return an error if a provider
	// that requires external configuration (e.g., an API key) is missing it.
	// For other providers (like bm25), it will do nothing and return nil.
	if err := b.resolveProviderConfig("tool", providerFamily); err != nil {
		return nil, fmt.Errorf("failed to resolve provider config for %q: %w", providerFamily, err)
	}

	// 2. Unmarshal the tool's parameters into the correct options struct
	optsType, hasOpts := nameToOptionsType[cfg.Provider]
	var optsPtr any
	if hasOpts {
		optsPtr = reflect.New(optsType.Elem()).Interface()
		jsonParams, err := json.Marshal(cfg.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params for %q: %w", name, err)
		}
		if err := json.Unmarshal(jsonParams, optsPtr); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params for %q: %w", name, err)
		}
		if err := resolvePathsInStruct(optsPtr, b.configDir); err != nil {
			return nil, fmt.Errorf("failed to resolve paths for tool %q: %w", name, err)
		}
	}

	// 3. Get the constructor from the registry and call it with dependencies
	switch cfg.Provider {
	case "openai-embedder", "google-embedder":
		constructor, err := Get(Registry.Embedder, cfg.Provider)
		if err != nil {
			return nil, err
		}
		switch cfg.Provider {
		case "google-embedder":
			newFn := constructor.(func(embed.GoogleEmbedderOptions, *genai.Client) (ai.Embedder, error))
			client := b.clients["google"].(googleClients)
			return newFn(*optsPtr.(*embed.GoogleEmbedderOptions), client.genai)
		case "openai-embedder":
			newFn := constructor.(func(embed.OpenAIEmbedderOptions, *openai.Client) (ai.Embedder, error))
			client := b.clients["openai"].(openai.Client)
			return newFn(*optsPtr.(*embed.OpenAIEmbedderOptions), &client)
		}

	case "localvec":
		constructor, _ := Get(Registry.Component, cfg.Provider)
		newFn := constructor.(func(core.LocalvecOptions, ai.Embedder) (core.VectorStore, error))
		embedderToolName := cfg.Params["embedder"].(string)
		embedder, ok := b.tools[embedderToolName].(ai.Embedder)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' for tool '%s' is not a valid embedder", embedderToolName, name)
		}
		return newFn(*optsPtr.(*core.LocalvecOptions), embedder)

	case "dense":
		constructor, _ := Get(Registry.Retriever, "dense")
		newFn := constructor.(func(ai.Embedder, core.VectorStore) (retrieve.Retriever, error))
		embedderToolName := cfg.Params["embedder"].(string)
		embedder := b.tools[embedderToolName].(ai.Embedder)
		vsToolName := cfg.Params["vectorStore"].(string)
		vs := b.tools[vsToolName].(core.VectorStore)
		return newFn(embedder, vs)

	case "bm25":
		constructor, _ := Get(Registry.Retriever, "bm25")
		newFn := constructor.(func(retrieve.BM25Options) (retrieve.Retriever, error))
		return newFn(*optsPtr.(*retrieve.BM25Options))

	case "hybrid":
		constructor, _ := Get(Registry.Retriever, "hybrid")
		newFn := constructor.(func(retrieve.HybridOptions) (retrieve.Retriever, error))
		bm25ToolName := cfg.Params["bm25"].(string)
		bm25Retriever, ok := b.tools[bm25ToolName].(retrieve.Retriever)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' for tool '%s' is not a valid retriever", bm25ToolName, name)
		}
		denseToolName := cfg.Params["dense"].(string)
		denseRetriever, ok := b.tools[denseToolName].(retrieve.Retriever)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' for tool '%s' is not a valid retriever", denseToolName, name)
		}
		return newFn(retrieve.HybridOptions{BM25Retriever: bm25Retriever, DenseRetriever: denseRetriever})

	case "openai", "google", "groq":
		constructor, _ := Get(Registry.LLM, cfg.Provider)
		switch cfg.Provider {
		case "google":
			newFn := constructor.(func(llm.GoogleOptions, *genkit.Genkit) (llm.Client, error))
			client := b.clients["google"].(googleClients)
			return newFn(*optsPtr.(*llm.GoogleOptions), client.genkit)
		case "openai", "groq":
			newFn := constructor.(func(llm.OpenAIOptions, *openai.Client) (llm.Client, error))
			client := b.clients[cfg.Provider].(openai.Client)
			return newFn(*optsPtr.(*llm.OpenAIOptions), &client)
		}

	default:
		return nil, fmt.Errorf("unsupported provider type for tool building: %q", cfg.Provider)
	}
	return nil, fmt.Errorf("unhandled provider in buildSingleTool: %s", cfg.Provider)
}

// providerToFamily maps a specific provider name (like openai-embedder) to its
// configuration family (like openai) for resolving API keys.
var providerToFamily = map[string]string{
	"openai-embedder": "openai",
	"google-embedder": "google",
}

// resolveDependencies handles identifying required providers and initializing API clients.
func (b *builder) resolveDependencies() error {
	// Infer embedder from components that need it, only if one is not already provided or configured.
	if b.embedder == nil && b.embedderName == "" {
		if b.vectorStoreName != "" {
			if name, ok := b.vectorStoreParams["embedder"].(string); ok {
				b.embedderName = name
			}
		} else if b.retrieverName == "dense" {
			if name, ok := b.retrieverParams["embedder"].(string); ok {
				b.embedderName = name
			}
		} else if b.retrieverName == "hybrid" {
			if denseParams, ok := b.retrieverParams["dense"].(map[string]any); ok {
				if name, ok := denseParams["embedder"].(string); ok {
					b.embedderName = name
				}
			}
		} else if b.rerankerName == "cosine" {
			if name, ok := b.rerankerParams["embedder"].(string); ok {
				b.embedderName = name
			}
		}
	}

	// Collect all required provider names to resolve their configs (e.g., API keys).
	if b.llmName != "" {
		b.providerNames["llm"] = b.llmName
	}
	// Only resolve embedder provider if we are building it from a name, not if it's pre-built.
	if b.embedder == nil && b.embedderName != "" {
		b.providerNames["embedder"] = b.embedderName
	}

	for compType, providerName := range b.providerNames {
		if err := b.resolveProviderConfig(compType, providerName); err != nil {
			return err
		}
	}
	return nil
}

// buildComponents calls the individual component builders in the correct order.
func (b *builder) buildComponents() error {
	// The order is important due to dependencies:
	// Embedder -> VectorStore -> Retriever
	// Embedder -> Reranker
	if err := b.buildEmbedder(); err != nil {
		return err
	}
	if err := b.buildVectorStore(); err != nil {
		return err
	}
	if err := b.buildRetriever(); err != nil {
		return err
	}
	if err := b.buildReranker(); err != nil {
		return err
	}
	if err := b.buildRules(); err != nil {
		return err
	}
	if err := b.buildLLM(); err != nil {
		return err
	}
	return nil
}

func (b *builder) buildEmbedder() error {
	// If an embedder is already built and provided, do nothing.
	if b.embedder != nil {
		return nil
	}
	if b.embedderName == "" {
		return nil
	}
	constructor, err := Get(Registry.Embedder, b.embedderName)
	if err != nil {
		return err
	}

	switch b.embedderName {
	case "google":
		newFn, ok := constructor.(func(embed.GoogleEmbedderOptions, *genai.Client) (ai.Embedder, error))
		if !ok {
			return fmt.Errorf("invalid constructor type for embedder '%s'", b.embedderName)
		}
		var opts embed.GoogleEmbedderOptions
		if o, ok := b.embedderParams["typedConfig"].(*embed.GoogleEmbedderOptions); ok && o != nil {
			opts = *o
		} else if o, ok := b.embedderParams["typedConfig"].(embed.GoogleEmbedderOptions); ok {
			opts = o
		}
		client, ok := b.clients["google"].(googleClients)
		if !ok {
			return fmt.Errorf("invalid client type for google embedder")
		}
		b.embedder, err = newFn(opts, client.genai)

	case "openai", "groq":
		newFn, ok := constructor.(func(embed.OpenAIEmbedderOptions, *openai.Client) (ai.Embedder, error))
		if !ok {
			return fmt.Errorf("invalid constructor type for embedder '%s'", b.embedderName)
		}
		var opts embed.OpenAIEmbedderOptions
		if o, ok := b.embedderParams["typedConfig"].(*embed.OpenAIEmbedderOptions); ok && o != nil {
			opts = *o
		} else if o, ok := b.embedderParams["typedConfig"].(embed.OpenAIEmbedderOptions); ok {
			opts = o
		}
		clientVal, ok := b.clients[b.embedderName].(openai.Client)
		if !ok {
			return fmt.Errorf("invalid client type for %s embedder, expected openai.Client but got %T", b.embedderName, b.clients[b.embedderName])
		}
		b.embedder, err = newFn(opts, &clientVal)

	default:
		return fmt.Errorf("unsupported embedder type in builder: %s", b.embedderName)
	}

	if err != nil {
		return fmt.Errorf("failed to build embedder '%s': %w", b.embedderName, err)
	}
	return nil
}

func (b *builder) buildVectorStore() error {
	if b.vectorStoreName == "" {
		return nil
	}
	if b.embedder == nil {
		return fmt.Errorf("vector store '%s' requires an embedder, but none was configured", b.vectorStoreName)
	}

	constructor, err := Get(Registry.Component, b.vectorStoreName)
	if err != nil {
		return err
	}
	newFn, ok := constructor.(func(core.LocalvecOptions, ai.Embedder) (core.VectorStore, error))
	if !ok {
		return fmt.Errorf("invalid constructor type for vector store '%s'", b.vectorStoreName)
	}

	var opts core.LocalvecOptions
	if o, ok := b.vectorStoreParams["typedConfig"].(*core.LocalvecOptions); ok && o != nil {
		opts = *o
	} else if o, ok := b.vectorStoreParams["typedConfig"].(core.LocalvecOptions); ok {
		opts = o
	} else {
		if path, ok := b.vectorStoreParams["path"].(string); ok {
			opts.Path = path
		}
	}
	if opts.Path == "" && b.config.VectorStore.Name == b.vectorStoreName {
		if path, ok := b.config.VectorStore.Params["path"].(string); ok {
			opts.Path = path
		}
	}

	b.vectorStore, err = newFn(opts, b.embedder)
	if err != nil {
		return fmt.Errorf("failed to build vector store '%s': %w", b.vectorStoreName, err)
	}
	return nil
}

func (b *builder) buildRetriever() error {
	if b.retrieverName == "" {
		return nil
	}
	var retriever retrieve.Retriever
	var err error

	switch b.retrieverName {
	case "dense":
		retriever, err = b.buildDenseRetriever()
	case "hybrid":
		retriever, err = b.buildHybridRetriever()
	case "bm25", "in-memory":
		retriever, err = b.buildMapBasedRetriever(b.retrieverName)
	default:
		return fmt.Errorf("unsupported retriever type in builder: %s", b.retrieverName)
	}

	if err != nil {
		return fmt.Errorf("failed to build retriever '%s': %w", b.retrieverName, err)
	}
	b.opts.Retriever = retriever
	return nil
}

func (b *builder) buildDenseRetriever() (retrieve.Retriever, error) {
	if b.embedder == nil || b.vectorStore == nil {
		return nil, fmt.Errorf("retriever 'dense' requires an embedder and a vector store")
	}
	constructor, err := Get(Registry.Retriever, "dense")
	if err != nil {
		return nil, err
	}
	newFn, ok := constructor.(func(ai.Embedder, core.VectorStore) (retrieve.Retriever, error))
	if !ok {
		return nil, fmt.Errorf("invalid constructor type for retriever 'dense'")
	}
	return newFn(b.embedder, b.vectorStore)
}

func (b *builder) buildHybridRetriever() (retrieve.Retriever, error) {
	constructor, err := Get(Registry.Retriever, "hybrid")
	if err != nil {
		return nil, err
	}
	newFn, ok := constructor.(func(retrieve.HybridOptions) (retrieve.Retriever, error))
	if !ok {
		return nil, fmt.Errorf("invalid constructor type for hybrid retriever")
	}

	bm25Retriever, err := b.buildMapBasedRetriever("bm25")
	if err != nil {
		return nil, fmt.Errorf("failed to build bm25 child for hybrid: %w", err)
	}

	var denseRetriever retrieve.Retriever
	if _, hasDense := b.retrieverParams["dense"]; hasDense {
		denseRetriever, err = b.buildDenseRetriever()
		if err != nil {
			return nil, fmt.Errorf("failed to build dense child for hybrid: %w", err)
		}
	}

	hybridOpts := retrieve.HybridOptions{
		BM25Retriever:  bm25Retriever,
		DenseRetriever: denseRetriever,
	}
	return newFn(hybridOpts)
}

func (b *builder) buildMapBasedRetriever(name string) (retrieve.Retriever, error) {
	constructor, err := Get(Registry.Retriever, name)
	if err != nil {
		return nil, err
	}

	// Logic to extract the correct parameters whether we are building a standalone retriever
	// or a child of a hybrid retriever.
	var sourceParams map[string]any
	if b.retrieverName == name && b.retrieverParams != nil {
		sourceParams = b.retrieverParams
	} else if b.retrieverName == "hybrid" {
		if childParams, ok := b.retrieverParams[name].(map[string]any); ok {
			sourceParams = childParams
		}
	}

	var retriever retrieve.Retriever
	switch name {
	case "bm25":
		newFn, ok := constructor.(func(retrieve.BM25Options) (retrieve.Retriever, error))
		if !ok {
			return nil, fmt.Errorf("invalid constructor type for retriever '%s'", name)
		}
		var opts retrieve.BM25Options
		if o, ok := sourceParams["typedConfig"].(*retrieve.BM25Options); ok && o != nil {
			opts = *o
		} else if o, ok := sourceParams["typedConfig"].(retrieve.BM25Options); ok {
			opts = o
		} else if p, ok := sourceParams["path"].(string); ok {
			opts.Path = p
		}
		retriever, err = newFn(opts)

	case "in-memory":
		newFn, ok := constructor.(func(retrieve.InMemoryOptions) (retrieve.Retriever, error))
		if !ok {
			return nil, fmt.Errorf("invalid constructor type for retriever '%s'", name)
		}
		var opts retrieve.InMemoryOptions
		if o, ok := sourceParams["typedConfig"].(*retrieve.InMemoryOptions); ok && o != nil {
			opts = *o
		} else if o, ok := sourceParams["typedConfig"].(retrieve.InMemoryOptions); ok {
			opts = o
		} else if d, ok := sourceParams["documents"].([]core.Doc); ok {
			opts.Documents = d
		}
		retriever, err = newFn(opts)

	default:
		return nil, fmt.Errorf("unsupported map-based retriever in builder: %s", name)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build retriever '%s': %w", name, err)
	}
	return retriever, nil
}

func (b *builder) buildReranker() error {
	if b.rerankerName == "" {
		return nil
	}
	if b.embedder == nil {
		return fmt.Errorf("reranker '%s' requires an embedder", b.rerankerName)
	}
	constructor, err := Get(Registry.Reranker, b.rerankerName)
	if err != nil {
		return err
	}
	newFn, ok := constructor.(func(rerank.CosineOptions, ai.Embedder) (rerank.Reranker, error))
	if !ok {
		return fmt.Errorf("invalid constructor type for reranker '%s'", b.rerankerName)
	}

	var opts rerank.CosineOptions
	if o, ok := b.rerankerParams["typedConfig"].(*rerank.CosineOptions); ok && o != nil {
		opts = *o
	} else if o, ok := b.rerankerParams["typedConfig"].(rerank.CosineOptions); ok {
		opts = o
	} else {
		if topK, ok := b.rerankerParams["topK"].(int); ok {
			opts.TopK = topK
		}
	}
	if opts.TopK == 0 && b.config.Reranker.Name == b.rerankerName {
		if topK, ok := b.config.Reranker.Params["topK"].(int); ok {
			opts.TopK = topK
		}
	}

	b.opts.Reranker, err = newFn(opts, b.embedder)
	if err != nil {
		return fmt.Errorf("failed to build reranker '%s': %w", b.rerankerName, err)
	}
	return nil
}

func (b *builder) buildRules() error {
	if b.rulesName == "" {
		// If a declarative orchestrator is being used, a rules engine is required.
		// This check is handled in the main Build method.
		return nil
	}
	if b.rulesName != "mangle" {
		return fmt.Errorf("rules engine '%s' not supported in this builder path", b.rulesName)
	}

	constructor, err := Get(Registry.Rules, b.rulesName)
	if err != nil {
		return err
	}
	newFn, ok := constructor.(func(context.Context, core.MangleOptions) (core.RuleSet, error))
	if !ok {
		return fmt.Errorf("invalid constructor type for rules '%s'", b.rulesName)
	}

	var opts core.MangleOptions
	if o, ok := b.rulesParams["typedConfig"].(*core.MangleOptions); ok && o != nil {
		opts = *o
	} else if o, ok := b.rulesParams["typedConfig"].(core.MangleOptions); ok {
		opts = o
	}
	if len(opts.Path) == 0 && b.config.Rules.Name == b.rulesName {
		if m, ok := b.config.Rules.Params["typedConfig"].(core.MangleOptions); ok {
			opts = m
		}
	}

	ruleset, err := newFn(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("failed to build rules '%s': %w", b.rulesName, err)
	}
	b.opts.Rules = ruleset // Store it in the options for sandwich mode.
	return nil
}

func (b *builder) buildLLM() error {
	if b.llmName == "" {
		return nil
	}
	constructor, err := Get(Registry.LLM, b.llmName)
	if err != nil {
		return err
	}

	switch b.llmName {
	case "google":
		newFn, ok := constructor.(func(llm.GoogleOptions, *genkit.Genkit) (llm.Client, error))
		if !ok {
			return fmt.Errorf("invalid constructor type for llm '%s'", b.llmName)
		}
		var opts llm.GoogleOptions
		if o, ok := b.llmParams["typedConfig"].(*llm.GoogleOptions); ok && o != nil {
			opts = *o
		} else if o, ok := b.llmParams["typedConfig"].(llm.GoogleOptions); ok {
			opts = o
		}
		client, ok := b.clients["google"].(googleClients)
		if !ok {
			return fmt.Errorf("invalid client type for google llm")
		}
		b.opts.LLM, err = newFn(opts, client.genkit)

	case "openai", "groq":
		newFn, ok := constructor.(func(llm.OpenAIOptions, *openai.Client) (llm.Client, error))
		if !ok {
			return fmt.Errorf("invalid constructor type for llm '%s'", b.llmName)
		}
		var opts llm.OpenAIOptions
		if o, ok := b.llmParams["typedConfig"].(*llm.OpenAIOptions); ok && o != nil {
			opts = *o
		} else if o, ok := b.llmParams["typedConfig"].(llm.OpenAIOptions); ok {
			opts = o
		}
		clientVal, ok := b.clients[b.llmName].(openai.Client)
		if !ok {
			return fmt.Errorf("invalid client type for %s llm, expected openai.Client but got %T", b.llmName, b.clients[b.llmName])
		}
		b.opts.LLM, err = newFn(opts, &clientVal)

	default:
		return fmt.Errorf("unsupported llm type in builder: %s", b.llmName)
	}

	if err != nil {
		return fmt.Errorf("failed to build llm '%s': %w", b.llmName, err)
	}
	return nil
}
