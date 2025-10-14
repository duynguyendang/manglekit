package manglekit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/pipeline"
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

// Builder provides a fluent, chainable interface for constructing a MangleKit
// Orchestrator. It is the recommended way to assemble a pipeline, as it handles
// dependency injection, component construction, and configuration resolution
// from multiple sources (programmatic, YAML, environment variables).
//
// The builder pattern allows for a clean and readable setup, where each component
// of the pipeline (e.g., Retriever, LLM, Rules Engine) is configured with a
// dedicated method. The final call to `Build()` consumes the configuration,
// resolves dependencies, and returns a fully initialized orchestrator.
//
// Example:
//
//	builder := NewBuilder()
//	orchestrator, err := builder.
//		WithRetriever(&retrieve.BM25Options{Path: "./docs"}).
//		WithLLM(&llm.GoogleOptions{Model: "gemini-1.5-flash"}).
//		Build()
//	if err != nil {
//		log.Fatalf("Failed to build orchestrator: %v", err)
//	}
type Builder struct {
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

// closer is a local interface used for type assertions. It ensures that any
// component that needs to be closed at shutdown has a `Close` method that
// adheres to the `ResourceCloser` function signature. This avoids the need for a
// public `Closer` interface in the `core` package, keeping the API clean.
type closer interface {
	Close(context.Context) error
}

// googleClients is a private struct to hold the initialized clients for Google services.
type googleClients struct {
	genkit *genkit.Genkit
	genai  *genai.Client
	cancel context.CancelFunc
}

// NewBuilder returns a new, empty instance of the fluent builder, ready to be
// configured. This is the primary entry point for programmatically constructing a
// MangleKit orchestrator.
func NewBuilder() *Builder {
	return &Builder{
		config:        &Config{},
		clients:       make(map[string]any),
		resolvedCfgs:  make(map[string]any),
		providerNames: make(map[string]string),
		tools:         make(map[string]any),
	}
}

// WithConfig applies a configuration object to the builder. This is the primary
// way to use settings loaded from a YAML file. Settings from the config object
// serve as a baseline, which can be overridden by individual `With...` calls.
func (b *Builder) WithConfig(cfg *Config) BuilderAPI {
	if cfg != nil {
		b.config = cfg
	}
	return b
}

// WithRetriever programmatically configures the retriever component for the pipeline.
// It accepts a pointer to a provider-specific options struct (e.g., `retrieve.BM25Options`),
// infers the provider's name from the type, and stores the configuration for later use
// during the `Build` process. If `opts` is nil, it clears any existing retriever configuration.
func (b *Builder) WithRetriever(opts any) BuilderAPI {
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
// for dense and hybrid retrievers. It accepts a pointer to a provider-specific options
// struct (e.g., `core.LocalvecOptions`), infers the provider's name from the type,
// and stores the configuration. If `opts` is nil, it clears any existing vector store configuration.
func (b *Builder) WithVectorStore(opts any) BuilderAPI {
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

// WithReranker programmatically configures the reranker component. It accepts a pointer
// to a provider-specific options struct (e.g., `rerank.CosineOptions`), infers the
// provider's name, and stores the configuration. If `opts` is nil, it clears any
// existing reranker configuration.
func (b *Builder) WithReranker(opts any) BuilderAPI {
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

// WithRules programmatically configures the rules engine. It accepts a pointer to a
// provider-specific options struct (e.g., `core.MangleOptions`), infers the provider's
// name, and stores the configuration. If `opts` is nil, it clears any existing rules
// engine configuration.
func (b *Builder) WithRules(opts any) BuilderAPI {
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

// WithLLM programmatically configures the language model client. It accepts a pointer
// to a provider-specific options struct (e.g., `llm.GoogleOptions`), infers the provider's
// name, and stores the configuration. If `opts` is nil, it clears any existing LLM
// configuration.
func (b *Builder) WithLLM(opts any) BuilderAPI {
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
// This is only used if the orchestrator type is set to "declarative" in the
// main configuration.
func (b *Builder) WithFlow(name string) BuilderAPI {
	b.flowName = name
	return b
}

var embedderAlias = map[string]string{
	"google-embedder": "google",
	"openai-embedder": "openai",
}

// WithEmbedder programmatically configures the text embedding model. It supports two
// modes of operation:
//  1. By provider options: Pass a pointer to a provider-specific options struct
//     (e.g., `embed.GoogleEmbedderOptions`). The builder infers the provider name
//     and constructs the embedder during the `Build` phase.
//  2. By pre-constructed instance: Pass an already-initialized object that implements
//     the `ai.Embedder` interface. This allows for injecting custom or externally
//     configured embedders.
//
// If `opts` is nil, any existing embedder configuration is cleared.
func (b *Builder) WithEmbedder(opts any) BuilderAPI {
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

// WithTopK programmatically sets the default number of documents to retrieve. This
// value acts as a fallback if a more specific value is not provided in a
// retriever's own configuration.
func (b *Builder) WithTopK(k int) BuilderAPI {
	b.opts.TopK = k
	return b
}

// WithMaxTokens programmatically sets the default maximum number of tokens for the
// LLM response. This value acts as a fallback if a more specific value is not
// provided in the LLM's own configuration.
func (b *Builder) WithMaxTokens(n int) BuilderAPI {
	b.opts.MaxTokens = n
	return b
}

// WithObservability programmatically sets the observability hooks (logger, tracer,
// meter) for the entire pipeline, enabling integration with monitoring and
// logging systems.
func (b *Builder) WithObservability(obs core.Observability) BuilderAPI {
	b.opts.Obs = obs
	return b
}

// WithFallbackThreshold programmatically sets the confidence score below which the
// pipeline may exit early and return a fallback response instead of calling the
// LLM. This is typically used with a reranker. A value of 0 disables this feature.
func (b *Builder) WithFallbackThreshold(f float64) BuilderAPI {
	b.opts.FallbackThreshold = f
	return b
}

// resolveProviderConfig finds the configuration for a given provider by checking params, config object, and env vars.
func (b *Builder) resolveProviderConfig(ctx context.Context, providerType, providerName string) error {
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
		if existing, ok := b.clients["google"].(googleClients); ok && existing.genkit != nil && existing.genai != nil {
			b.resolvedCfgs[key] = cfg
			return nil
		}
		gCtx, cancel := context.WithCancel(ctx)
		g, err := genai.NewClient(gCtx, googleapi.WithAPIKey(apiKey))
		if err != nil {
			cancel()
			return fmt.Errorf("failed to create genai client: %w", err)
		}
		gkit := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{APIKey: apiKey}))
		clients := googleClients{
			genkit: gkit,
			genai:  g,
			cancel: cancel,
		}
		var once sync.Once
		var closeErr error
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, func(closeCtx context.Context) error {
			once.Do(func() {
				if clients.cancel != nil {
					clients.cancel()
				}
				if clients.genai != nil {
					if err := clients.genai.Close(); err != nil {
						closeErr = errors.Join(closeErr, err)
					}
				}
			})
			return closeErr
		})
		b.clients["google"] = clients
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
		transport := &http.Transport{
			IdleConnTimeout: 30 * time.Second,
		}
		httpClient := &http.Client{
			Transport: transport,
			Timeout:   120 * time.Second,
		}
		client := openai.NewClient(option.WithAPIKey(apiKey), option.WithHTTPClient(httpClient))
		clientPtr := client
		b.clients["openai"] = &clientPtr
		b.resolvedCfgs[key] = cfg
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, func(ctx context.Context) error {
			transport.CloseIdleConnections()
			return nil
		})

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
		transport := &http.Transport{
			IdleConnTimeout: 30 * time.Second,
		}
		httpClient := &http.Client{
			Transport: transport,
			Timeout:   120 * time.Second,
		}
		client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL), option.WithHTTPClient(httpClient))
		clientPtr := client
		b.clients["groq"] = &clientPtr
		b.resolvedCfgs[key] = cfg
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, func(ctx context.Context) error {
			transport.CloseIdleConnections()
			return nil
		})
	}
	return nil
}

// Build constructs the final Orchestrator by resolving all dependencies and
// building the components in the correct order. It is the terminal method in
// the builder chain.
//
// The build process differs based on the orchestrator type ("sandwich" or "declarative"):
//
//   - For the "sandwich" orchestrator (default), it follows a linear dependency
//     resolution: Embedder -> VectorStore -> Retriever/Reranker -> LLM -> Rules.
//
//   - For the "declarative" orchestrator, it first builds a required FlowController
//     (a special rules engine) and then iteratively builds all "tools" defined in
//     the configuration, respecting inter-tool dependencies.
//
// In both cases, it handles the initialization of API clients (e.g., Google,
// OpenAI), manages their lifecycle, and injects them into the components that
// need them. If any step fails, it attempts to clean up any resources that were
// allocated before returning an error.
func (b *Builder) Build(ctx context.Context) (core.Orchestrator, error) {
	orchestratorType := "sandwich" // default
	if b.config.Orchestrator.Type != "" {
		orchestratorType = b.config.Orchestrator.Type
	}

	b.ensureObservabilityDefaults()

	if len(b.errs) > 0 {
		return nil, errors.Join(b.errs...)
	}

	switch orchestratorType {
	case "declarative":
		// The declarative orchestrator requires a rules engine.
		// Note: The main rules engine for the orchestrator is built here,
		// separate from any "rules" tools that might be defined.
		if err := b.buildRules(ctx); err != nil {
			closeErr := b.closeResources(ctx)
			return nil, errors.Join(fmt.Errorf("failed to build rules for declarative orchestrator: %w", err), closeErr)
		}
		if b.opts.Rules == nil {
			closeErr := b.closeResources(ctx)
			return nil, errors.Join(errors.New("declarative orchestrator requires a rules engine, but none was configured"), closeErr)
		}

		// Build all the tools defined in the config.
		if err := b.buildTools(ctx); err != nil {
			closeErr := b.closeResources(ctx)
			return nil, errors.Join(err, closeErr)
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
		if _, ok := b.tools["mangle_rules"]; !ok {
			b.tools["mangle_rules"] = flowController
		}

		orchestrator, err := declarative.New(flowController, b.tools, flowName, b.opts.Obs, b.opts.ResourceClosers)
		if err != nil {
			closeErr := b.closeResources(ctx)
			return nil, errors.Join(err, closeErr)
		}
		return orchestrator, nil

	case "sandwich":
		if err := b.resolveDependencies(ctx); err != nil {
			closeErr := b.closeResources(ctx)
			return nil, errors.Join(err, closeErr)
		}
		if err := b.buildComponents(ctx); err != nil {
			closeErr := b.closeResources(ctx)
			return nil, errors.Join(err, closeErr)
		}
		orchestrator, err := pipeline.NewSandwich(b.opts)
		if err != nil {
			closeErr := b.closeResources(ctx)
			return nil, errors.Join(err, closeErr)
		}
		return orchestrator, nil

	default:
		closeErr := b.closeResources(ctx)
		return nil, errors.Join(fmt.Errorf("unknown orchestrator type: %q", orchestratorType), closeErr)
	}
}

func (b *Builder) ensureObservabilityDefaults() {
	if b.opts.Obs.Logger == nil {
		b.opts.Obs.Logger = logger.NewStdLogger()
	}
}

// buildTools iterates through the tools defined in the config, respects dependencies,
// and builds each one. It uses an iterative approach to handle the dependency graph.
func (b *Builder) buildTools(ctx context.Context) error {
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
				tool, err := b.buildSingleTool(ctx, name, cfg)
				if err != nil {
					return fmt.Errorf("failed to build tool %q: %w", name, err)
				}
				if c, ok := tool.(closer); ok {
					b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
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
func (b *Builder) buildSingleTool(ctx context.Context, name string, cfg ToolConfig) (any, error) {
	// 1. Resolve provider-level config (e.g., API keys)
	providerFamily := cfg.Provider
	if family, ok := providerToFamily[providerFamily]; ok {
		providerFamily = family
	}

	if norm, ok := embedderAlias[providerFamily]; ok {
		providerFamily = norm
	}

	// The resolveProviderConfig function will only return an error if a provider
	// that requires external configuration (e.g., an API key) is missing it.
	// For other providers (like bm25), it will do nothing and return nil.
	if err := b.resolveProviderConfig(ctx, "tool", providerFamily); err != nil {
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
			newFn, ok := constructor.(func(embed.GoogleEmbedderOptions, *genai.Client) (ai.Embedder, error))
			if !ok {
				return nil, fmt.Errorf("invalid constructor type for google-embedder")
			}
			client, ok := b.clients["google"].(googleClients)
			if !ok {
				return nil, fmt.Errorf("google client not initialized")
			}
			return newFn(*optsPtr.(*embed.GoogleEmbedderOptions), client.genai)
		case "openai-embedder":
			newFn, ok := constructor.(func(embed.OpenAIEmbedderOptions, *openai.Client) (ai.Embedder, error))
			if !ok {
				return nil, fmt.Errorf("invalid constructor type for openai-embedder")
			}
			client, ok := b.clients["openai"].(*openai.Client)
			if !ok {
				return nil, fmt.Errorf("invalid client type for openai-embedder")
			}
			return newFn(*optsPtr.(*embed.OpenAIEmbedderOptions), client)
		}

	case "localvec":
		constructor, _ := Get(Registry.Component, cfg.Provider)
		newFn := constructor.(func(context.Context, core.LocalvecOptions, ai.Embedder) (core.VectorStore, error))
		embedderToolName := cfg.Params["embedder"].(string)
		embedder, ok := b.tools[embedderToolName].(ai.Embedder)
		if !ok {
			return nil, fmt.Errorf("dependency '%s' for tool '%s' is not a valid embedder", embedderToolName, name)
		}
		// We pass a background context here because the tool's lifecycle is managed
		// by the main application context, which is not available here.
		return newFn(ctx, *optsPtr.(*core.LocalvecOptions), embedder)

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
			client := b.clients[cfg.Provider].(*openai.Client)
			return newFn(*optsPtr.(*llm.OpenAIOptions), client)
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
func (b *Builder) resolveDependencies(ctx context.Context) error {
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
		if norm, ok := embedderAlias[b.embedderName]; ok {
			b.embedderName = norm
		}
		b.providerNames["embedder"] = b.embedderName
	}

	for compType, providerName := range b.providerNames {
		if err := b.resolveProviderConfig(ctx, compType, providerName); err != nil {
			return err
		}
	}
	return nil
}

// buildComponents calls the individual component builders in the correct order.
func (b *Builder) buildComponents(ctx context.Context) error {
	// The order is important due to dependencies:
	// Embedder -> VectorStore -> Retriever
	// Embedder -> Reranker
	if err := b.buildEmbedder(); err != nil {
		return err
	}
	if err := b.buildVectorStore(ctx); err != nil {
		return err
	}
	if err := b.buildRetriever(); err != nil {
		return err
	}
	if err := b.buildReranker(); err != nil {
		return err
	}
	if err := b.buildRules(ctx); err != nil {
		return err
	}
	if err := b.buildLLM(); err != nil {
		return err
	}
	return nil
}

func (b *Builder) buildEmbedder() error {
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
		clientVal, ok := b.clients[b.embedderName].(*openai.Client)
		if !ok {
			return fmt.Errorf("invalid client type for %s embedder, expected *openai.Client but got %T", b.embedderName, b.clients[b.embedderName])
		}
		b.embedder, err = newFn(opts, clientVal)

	default:
		return fmt.Errorf("unsupported embedder type in builder: %s", b.embedderName)
	}

	if err != nil {
		return fmt.Errorf("failed to build embedder '%s': %w", b.embedderName, err)
	}
	if c, ok := b.embedder.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}
	return nil
}

func (b *Builder) buildVectorStore(ctx context.Context) error {
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
	newFn, ok := constructor.(func(context.Context, core.LocalvecOptions, ai.Embedder) (core.VectorStore, error))
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

	// We pass a background context here because the vector store's lifecycle is managed
	// by the main application context, which is not available here. The Close()
	// method on the vector store will handle the shutdown.
	b.vectorStore, err = newFn(ctx, opts, b.embedder)
	if err != nil {
		return fmt.Errorf("failed to build vector store '%s': %w", b.vectorStoreName, err)
	}
	if c, ok := b.vectorStore.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}
	return nil
}

func (b *Builder) buildRetriever() error {
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

func (b *Builder) buildDenseRetriever() (retrieve.Retriever, error) {
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

func (b *Builder) buildHybridRetriever() (retrieve.Retriever, error) {
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

func (b *Builder) buildMapBasedRetriever(name string) (retrieve.Retriever, error) {
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
		if opts.Logger == nil {
			opts.Logger = b.opts.Obs.Logger.With("component", "retriever", "provider", name)
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

func (b *Builder) buildReranker() error {
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

func (b *Builder) buildRules(ctx context.Context) error {
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
	if opts.Logger == nil {
		opts.Logger = b.opts.Obs.Logger.With("component", "rules", "provider", b.rulesName)
	}

	ruleset, err := newFn(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to build rules '%s': %w", b.rulesName, err)
	}
	b.opts.Rules = ruleset // Store it in the options for sandwich mode.
	return nil
}

func (b *Builder) buildLLM() error {
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
		clientVal, ok := b.clients[b.llmName].(*openai.Client)
		if !ok {
			return fmt.Errorf("invalid client type for %s llm, expected *openai.Client but got %T", b.llmName, b.clients[b.llmName])
		}
		b.opts.LLM, err = newFn(opts, clientVal)

	default:
		return fmt.Errorf("unsupported llm type in builder: %s", b.llmName)
	}

	if err != nil {
		return fmt.Errorf("failed to build llm '%s': %w", b.llmName, err)
	}
	if c, ok := b.opts.LLM.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}
	return nil
}

// closeResources attempts to release any provider clients that were opened during the build.
// It is primarily used on error paths to avoid leaking background goroutines or connections
// when the build cannot be completed.
func (b *Builder) closeResources(ctx context.Context) error {
	if len(b.opts.ResourceClosers) == 0 {
		return nil
	}
	closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var combined error
	for i := len(b.opts.ResourceClosers) - 1; i >= 0; i-- {
		if b.opts.ResourceClosers[i] == nil {
			continue
		}
		if err := b.opts.ResourceClosers[i](closeCtx); err != nil {
			combined = errors.Join(combined, err)
		}
	}
	return combined
}
