package manglekit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/pipeline/declarative"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
)

// BuilderAPI defines the fluent interface for the MangleKit orchestrator builder.
// It is the primary API for programmatically configuring and constructing a pipeline.
// This interface is returned by all chained `With...` methods to allow for a
// seamless and readable configuration flow.
type BuilderAPI interface {
	WithConfig(cfg *Config) BuilderAPI
	WithRetriever(opts any) BuilderAPI
	WithVectorStore(opts any) BuilderAPI
	WithReranker(opts any) BuilderAPI
	WithRules(opts any) BuilderAPI
	WithLLM(opts any) BuilderAPI
	WithFlow(name string) BuilderAPI
	WithEmbedder(opts any) BuilderAPI
	WithTopK(k int) BuilderAPI
	WithMaxTokens(n int) BuilderAPI
	WithObservability(obs core.Observability) BuilderAPI
	WithFallbackThreshold(f float64) BuilderAPI
	WithStateProvider(opts any) BuilderAPI
	Build(ctx context.Context) (core.Orchestrator, error)
	BuildRetrieverComponent(ctx context.Context, name string, params map[string]any) (retrieve.Retriever, error)
}

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
	opts      core.Options
	config    *Config
	errs      []error
	registry  *Registry
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
	embedder      ai.Embedder
	vectorStore   core.VectorStore
	stateProvider core.StateProvider

	stateProviderName   string
	stateProviderParams map[string]any
}

// closer is a local interface used for type assertions. It ensures that any
// component that needs to be closed at shutdown has a `Close` method that
// adheres to the `ResourceCloser` function signature. This avoids the need for a
// public `Closer` interface in the `core` package, keeping the API clean.
type closer interface {
	Close(context.Context) error
}

// NewBuilder returns a new, empty instance of the fluent builder, ready to be
// configured. This is the primary entry point for programmatically constructing a
// MangleKit orchestrator.
func NewBuilder(r *Registry) *Builder {
	b := &Builder{
		config:        &Config{},
		registry:      r,
		clients:       make(map[string]any),
		resolvedCfgs:  make(map[string]any),
		providerNames: make(map[string]string),
		tools:         make(map[string]any),
	}
	b.opts.Obs.Logger = logger.NewStdLogger()
	return b
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
		err := fmt.Errorf("unregistered options type for retriever: %T", opts)
		b.opts.Obs.Logger.Errorf(err.Error())
		b.errs = append(b.errs, err)
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
		err := fmt.Errorf("unregistered options type for vector store: %T", opts)
		b.opts.Obs.Logger.Errorf(err.Error())
		b.errs = append(b.errs, err)
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
		err := fmt.Errorf("unregistered options type for reranker: %T", opts)
		b.opts.Obs.Logger.Errorf(err.Error())
		b.errs = append(b.errs, err)
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
		err := fmt.Errorf("unregistered options type for rules engine: %T", opts)
		b.opts.Obs.Logger.Errorf(err.Error())
		b.errs = append(b.errs, err)
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
		err := fmt.Errorf("unregistered options type for LLM: %T", opts)
		b.opts.Obs.Logger.Errorf(err.Error())
		b.errs = append(b.errs, err)
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
		err := fmt.Errorf("unregistered options type for embedder: %T", opts)
		b.opts.Obs.Logger.Errorf(err.Error())
		b.errs = append(b.errs, err)
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

// WithStateProvider programmatically configures the state provider. It accepts a
// pointer to a provider-specific options struct (e.g., `redis.Options`), infers
// the provider's name from the type, and stores the configuration for later use
// during the `Build` process.
func (b *Builder) WithStateProvider(opts any) BuilderAPI {
	if opts == nil {
		b.stateProviderName = ""
		b.stateProviderParams = nil
		return b
	}
	optsType := reflect.TypeOf(opts)
	name, ok := optionsTypeToName[optsType]
	if !ok {
		err := fmt.Errorf("unregistered options type for state provider: %T", opts)
		b.opts.Obs.Logger.Errorf(err.Error())
		b.errs = append(b.errs, err)
		return b
	}
	b.stateProviderName = name
	b.stateProviderParams = map[string]any{"typedConfig": opts}
	return b
}

// resolveProviderConfig finds the configuration for a given provider by checking params, config object, and env vars.
func (b *Builder) resolveProviderConfig(ctx context.Context, providerType, providerName string) error {
	if _, exists := b.clients[providerName]; exists {
		return nil // Already resolved.
	}

	factory, err := Get(b.registry.ClientFactories, providerName)
	if err != nil {
		// Not all providers have client factories, so this is not an error.
		return nil
	}

	clientFactory, ok := factory.(ClientFactory)
	if !ok {
		return fmt.Errorf("invalid client factory type for provider '%s'", providerName)
	}

	client, closer, err := clientFactory(ctx, b.config)
	if err != nil {
		return fmt.Errorf("failed to create client for provider '%s': %w", providerName, err)
	}

	b.clients[providerName] = client
	if closer != nil {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, closer)
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
	b.opts.Obs.Logger.Infof("starting build for orchestrator type %q", orchestratorType)

	if len(b.errs) > 0 {
		err := errors.Join(b.errs...)
		b.opts.Obs.Logger.Errorf("pre-build validation failed: %v", err)
		return nil, err
	}

	switch orchestratorType {
	case "declarative":
		b.opts.Obs.Logger.Debugf("building declarative orchestrator")
		if err := b.buildRules(ctx); err != nil {
			closeErr := b.closeResources(ctx)
			err = fmt.Errorf("failed to build rules for declarative orchestrator: %w", err)
			b.opts.Obs.Logger.Errorf(err.Error())
			return nil, errors.Join(err, closeErr)
		}
		if b.opts.Rules == nil {
			closeErr := b.closeResources(ctx)
			err := errors.New("declarative orchestrator requires a rules engine, but none was configured")
			b.opts.Obs.Logger.Errorf(err.Error())
			return nil, errors.Join(err, closeErr)
		}

		b.opts.Obs.Logger.Debugf("building tools for declarative orchestrator")
		if err := b.buildTools(ctx); err != nil {
			closeErr := b.closeResources(ctx)
			b.opts.Obs.Logger.Errorf("failed to build tools: %v", err)
			return nil, errors.Join(err, closeErr)
		}

		flowName := b.flowName
		if flowName == "" {
			flowName = b.config.Orchestrator.FlowName
		}
		b.opts.Obs.Logger.Debugf("using flow %q", flowName)

		flowController, ok := b.opts.Rules.(core.FlowController)
		if !ok {
			err := fmt.Errorf("rules engine for declarative orchestrator must be a FlowController, but got %T", b.opts.Rules)
			b.opts.Obs.Logger.Errorf(err.Error())
			return nil, err
		}
		if _, ok := b.tools["mangle_rules"]; !ok {
			b.tools["mangle_rules"] = flowController
		}

		orchestrator, err := declarative.New(flowController, b.tools, flowName, b.stateProvider, b.opts.Obs, b.opts.ResourceClosers)
		if err != nil {
			closeErr := b.closeResources(ctx)
			b.opts.Obs.Logger.Errorf("failed to create new declarative orchestrator: %v", err)
			return nil, errors.Join(err, closeErr)
		}
		b.opts.Obs.Logger.Infof("successfully built declarative orchestrator")
		return orchestrator, nil

	case "sandwich":
		b.opts.Obs.Logger.Debugf("building sandwich orchestrator")
		if err := b.resolveDependencies(ctx); err != nil {
			closeErr := b.closeResources(ctx)
			b.opts.Obs.Logger.Errorf("failed to resolve dependencies: %v", err)
			return nil, errors.Join(err, closeErr)
		}
		if err := b.buildComponents(ctx); err != nil {
			closeErr := b.closeResources(ctx)
			b.opts.Obs.Logger.Errorf("failed to build components: %v", err)
			return nil, errors.Join(err, closeErr)
		}
		b.opts.StateProvider = b.stateProvider
		orchestrator, err := pipeline.NewSandwich(b.opts)
		if err != nil {
			closeErr := b.closeResources(ctx)
			b.opts.Obs.Logger.Errorf("failed to create new sandwich pipeline: %v", err)
			return nil, errors.Join(err, closeErr)
		}
		b.opts.Obs.Logger.Infof("successfully built sandwich orchestrator")
		return orchestrator, nil

	default:
		closeErr := b.closeResources(ctx)
		err := fmt.Errorf("unknown orchestrator type: %q", orchestratorType)
		b.opts.Obs.Logger.Errorf(err.Error())
		return nil, errors.Join(err, closeErr)
	}
}

// buildTools iterates through the tools defined in the config, respects dependencies,
// and builds each one. It uses an iterative approach to handle the dependency graph.
func (b *Builder) buildTools(ctx context.Context) error {
	toBuild := make(map[string]ToolConfig)
	for name, cfg := range b.config.Tools {
		toBuild[name] = cfg
	}

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
				b.opts.Obs.Logger.Debugf("building tool %q", name)
				tool, err := b.buildSingleTool(ctx, name, cfg)
				if err != nil {
					b.opts.Obs.Logger.Errorf("failed to build tool %q: %v", name, err)
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
			var remaining []string
			for name := range toBuild {
				remaining = append(remaining, name)
			}
			err := fmt.Errorf("circular dependency detected or missing tools. cannot build: %v", remaining)
			b.opts.Obs.Logger.Errorf(err.Error())
			return err
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
	providerFamily := cfg.Provider
	if family, ok := providerToFamily[providerFamily]; ok {
		providerFamily = family
	}

	if norm, ok := embedderAlias[providerFamily]; ok {
		providerFamily = norm
	}
	if err := b.resolveProviderConfig(ctx, "tool", providerFamily); err != nil {
		err = fmt.Errorf("failed to resolve provider config for %q: %w", providerFamily, err)
		b.opts.Obs.Logger.Errorf(err.Error())
		return nil, err
	}

	optsType, hasOpts := nameToOptionsType[cfg.Provider]
	var optsPtr any
	if hasOpts {
		optsPtr = reflect.New(optsType.Elem()).Interface()
		jsonParams, err := json.Marshal(cfg.Params)
		if err != nil {
			err = fmt.Errorf("failed to marshal params for %q: %w", name, err)
			b.opts.Obs.Logger.Errorf(err.Error())
			return nil, err
		}
		if err := json.Unmarshal(jsonParams, optsPtr); err != nil {
			err = fmt.Errorf("failed to unmarshal params for %q: %w", name, err)
			b.opts.Obs.Logger.Errorf(err.Error())
			return nil, err
		}
		if err := resolvePathsInStruct(optsPtr, b.configDir); err != nil {
			err = fmt.Errorf("failed to resolve paths for tool %q: %w", name, err)
			b.opts.Obs.Logger.Errorf(err.Error())
			return nil, err
		}
	}

	deps := make(FactoryDeps)
	if client, ok := b.clients[providerFamily]; ok {
		deps["client"] = client
	}
	if embedder, ok := b.tools[cfg.Params["embedder"].(string)].(ai.Embedder); ok {
		deps["embedder"] = embedder
	}
	if vs, ok := b.tools[cfg.Params["vectorStore"].(string)].(core.VectorStore); ok {
		deps["vectorStore"] = vs
	}
	if bm25, ok := b.tools[cfg.Params["bm25"].(string)].(retrieve.Retriever); ok {
		deps["bm25"] = bm25
	}
	if dense, ok := b.tools[cfg.Params["dense"].(string)].(retrieve.Retriever); ok {
		deps["dense"] = dense
	}

	var component any
	var err error

	switch cfg.Type {
	case "retriever":
		factory, err := Get(b.registry.Retrievers, cfg.Provider)
		if err != nil {
			return nil, err
		}
		component, err = factory(ctx, optsPtr, deps)
	case "llm":
		factory, err := Get(b.registry.LLMs, cfg.Provider)
		if err != nil {
			return nil, err
		}
		component, err = factory(ctx, optsPtr, deps)
	case "embedder":
		factory, err := Get(b.registry.Embedders, cfg.Provider)
		if err != nil {
			return nil, err
		}
		component, err = factory(ctx, optsPtr, deps)
	case "reranker":
		factory, err := Get(b.registry.Rerankers, cfg.Provider)
		if err != nil {
			return nil, err
		}
		component, err = factory(ctx, optsPtr, deps)
	case "vectorStore":
		factory, err := Get(b.registry.VectorStores, cfg.Provider)
		if err != nil {
			return nil, err
		}
		component, err = factory(ctx, optsPtr, deps)
	case "rules":
		factory, err := Get(b.registry.RuleSets, cfg.Provider)
		if err != nil {
			return nil, err
		}
		component, err = factory(ctx, optsPtr, deps)
	case "stateProvider":
		factory, err := Get(b.registry.StateProviders, cfg.Provider)
		if err != nil {
			return nil, err
		}
		component, err = factory(ctx, optsPtr, deps)
	default:
		return nil, fmt.Errorf("unsupported tool type: %q", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build tool %q of type %q: %w", name, cfg.Type, err)
	}
	return component, nil
}

// providerToFamily maps a specific provider name (like openai-embedder) to its
// configuration family (like openai) for resolving API keys.
var providerToFamily = map[string]string{
	"openai-embedder": "openai",
	"google-embedder": "google",
}

// resolveDependencies handles identifying required providers and initializing API clients.
func (b *Builder) resolveDependencies(ctx context.Context) error {
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

	if b.llmName != "" {
		b.providerNames["llm"] = b.llmName
	}
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
	if err := b.buildEmbedder(ctx); err != nil {
		return err
	}
	if err := b.buildVectorStore(ctx); err != nil {
		return err
	}
	if err := b.buildRetriever(ctx); err != nil {
		return err
	}
	if err := b.buildReranker(ctx); err != nil {
		return err
	}
	if err := b.buildRules(ctx); err != nil {
		return err
	}
	if err := b.buildLLM(ctx); err != nil {
		return err
	}
	if err := b.buildStateProvider(ctx); err != nil {
		return err
	}
	return nil
}

func (b *Builder) buildEmbedder(ctx context.Context) error {
	if b.embedder != nil {
		return nil
	}
	if b.embedderName == "" {
		return nil
	}

	b.opts.Obs.Logger.Debugf("building embedder %q", b.embedderName)

	factory, err := Get(b.registry.Embedders, b.embedderName)
	if err != nil {
		return fmt.Errorf("failed to get factory for embedder '%s': %w", b.embedderName, err)
	}

	var opts any
	if o, ok := b.embedderParams["typedConfig"]; ok {
		opts = o
	}

	deps := make(FactoryDeps)
	if client, ok := b.clients[b.embedderName]; ok {
		deps["client"] = client
	} else if client, ok := b.clients[providerToFamily[b.embedderName]]; ok {
		deps["client"] = client
	}

	embedder, err := factory(ctx, opts, deps)
	if err != nil {
		return fmt.Errorf("failed to build embedder '%s': %w", b.embedderName, err)
	}

	b.embedder = embedder
	if c, ok := b.embedder.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}

	b.opts.Obs.Logger.Infof("initialized component embedder with provider %s", b.embedderName)
	return nil
}

func (b *Builder) buildVectorStore(ctx context.Context) error {
	if b.vectorStoreName == "" {
		return nil
	}
	if b.embedder == nil {
		return fmt.Errorf("vector store '%s' requires an embedder, but none was configured", b.vectorStoreName)
	}

	b.opts.Obs.Logger.Debugf("building vector store %q", b.vectorStoreName)

	factory, err := Get(b.registry.VectorStores, b.vectorStoreName)
	if err != nil {
		return fmt.Errorf("failed to get factory for vector store '%s': %w", b.vectorStoreName, err)
	}

	var opts any
	if o, ok := b.vectorStoreParams["typedConfig"]; ok {
		opts = o
	}

	deps := make(FactoryDeps)
	deps["embedder"] = b.embedder

	vectorStore, err := factory(ctx, opts, deps)
	if err != nil {
		return fmt.Errorf("failed to build vector store '%s': %w", b.vectorStoreName, err)
	}

	b.vectorStore = vectorStore
	if c, ok := b.vectorStore.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}

	b.opts.Obs.Logger.Infof("initialized component vectorStore with provider %s", b.vectorStoreName)
	return nil
}

func (b *Builder) buildRetriever(ctx context.Context) error {
	if b.retrieverName == "" {
		return nil
	}

	b.opts.Obs.Logger.Debugf("building retriever %q", b.retrieverName)

	factory, err := Get(b.registry.Retrievers, b.retrieverName)
	if err != nil {
		return fmt.Errorf("failed to get factory for retriever '%s': %w", b.retrieverName, err)
	}

	var opts any
	if o, ok := b.retrieverParams["typedConfig"]; ok {
		opts = o
	}

	deps := make(FactoryDeps)
	deps["embedder"] = b.embedder
	deps["vectorStore"] = b.vectorStore
	deps["builder"] = b // Pass the builder itself for callbacks.

	retriever, err := factory(ctx, opts, deps)
	if err != nil {
		return fmt.Errorf("failed to build retriever '%s': %w", b.retrieverName, err)
	}

	b.opts.Retriever = retriever
	b.opts.Obs.Logger.Infof("initialized component retriever with provider %s", b.retrieverName)
	return nil
}

func (b *Builder) BuildRetrieverComponent(ctx context.Context, name string, params map[string]any) (retrieve.Retriever, error) {
	factory, err := Get(b.registry.Retrievers, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get factory for retriever '%s': %w", name, err)
	}

	var opts any
	if o, ok := params["typedConfig"]; ok {
		opts = o
	}

	deps := make(FactoryDeps)
	deps["embedder"] = b.embedder
	deps["vectorStore"] = b.vectorStore

	retriever, err := factory(ctx, opts, deps)
	if err != nil {
		return nil, err
	}
	return retriever, nil
}

func (b *Builder) buildReranker(ctx context.Context) error {
	if b.rerankerName == "" {
		return nil
	}

	b.opts.Obs.Logger.Debugf("building reranker %q", b.rerankerName)

	factory, err := Get(b.registry.Rerankers, b.rerankerName)
	if err != nil {
		return fmt.Errorf("failed to get factory for reranker '%s': %w", b.rerankerName, err)
	}

	var opts any
	if o, ok := b.rerankerParams["typedConfig"]; ok {
		opts = o
	}

	deps := make(FactoryDeps)
	deps["embedder"] = b.embedder

	reranker, err := factory(ctx, opts, deps)
	if err != nil {
		return fmt.Errorf("failed to build reranker '%s': %w", b.rerankerName, err)
	}

	b.opts.Reranker = reranker
	b.opts.Obs.Logger.Infof("initialized component reranker with provider %s", b.rerankerName)
	return nil
}

func (b *Builder) buildRules(ctx context.Context) error {
	if b.rulesName == "" {
		return nil
	}
	b.opts.Obs.Logger.Debugf("building rules engine %q", b.rulesName)

	factory, err := Get(b.registry.RuleSets, b.rulesName)
	if err != nil {
		return err
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
	deps := make(FactoryDeps)
	deps["registry"] = b.registry

	ruleset, err := factory(ctx, &opts, deps)
	if err != nil {
		err = fmt.Errorf("failed to build rules '%s': %w", b.rulesName, err)
		b.opts.Obs.Logger.Errorf(err.Error())
		return err
	}
	b.opts.Rules = ruleset
	b.opts.Obs.Logger.Infof("initialized component rules with provider %s", b.rulesName)
	return nil
}

func (b *Builder) buildLLM(ctx context.Context) error {
	if b.llmName == "" {
		return nil
	}

	b.opts.Obs.Logger.Debugf("building llm %q", b.llmName)

	factory, err := Get(b.registry.LLMs, b.llmName)
	if err != nil {
		return fmt.Errorf("failed to get factory for llm '%s': %w", b.llmName, err)
	}

	var opts any
	if o, ok := b.llmParams["typedConfig"]; ok {
		opts = o
	}

	deps := make(FactoryDeps)
	if client, ok := b.clients[b.llmName]; ok {
		deps["client"] = client
	}

	llmClient, err := factory(ctx, opts, deps)
	if err != nil {
		return fmt.Errorf("failed to build llm '%s': %w", b.llmName, err)
	}

	b.opts.LLM = llmClient
	if c, ok := b.opts.LLM.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}

	b.opts.Obs.Logger.Infof("initialized component llm with provider %s", b.llmName)
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

func (b *Builder) buildStateProvider(ctx context.Context) error {
	if b.stateProviderName == "" {
		return nil
	}

	b.opts.Obs.Logger.Debugf("building state provider %q", b.stateProviderName)

	factory, err := Get(b.registry.StateProviders, b.stateProviderName)
	if err != nil {
		return fmt.Errorf("failed to get factory for state provider '%s': %w", b.stateProviderName, err)
	}

	var opts any
	if o, ok := b.stateProviderParams["typedConfig"]; ok {
		opts = o
	}

	provider, err := factory(ctx, opts, nil) // State providers currently don't have dependencies.
	if err != nil {
		return fmt.Errorf("failed to build state provider '%s': %w", b.stateProviderName, err)
	}

	b.stateProvider = provider
	if c, ok := provider.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}

	b.opts.Obs.Logger.Infof("initialized component stateProvider with provider %s", b.stateProviderName)
	return nil
}
