package manglekit

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/pipeline"
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
	BuildRetriever(ctx context.Context, name string, params map[string]any) (retrieve.Retriever, error)
}

// Builder provides a fluent, chainable interface for constructing a MangleKit
// Orchestrator. It is the recommended way to assemble a pipeline, as it handles
// dependency injection, component construction, and configuration resolution
// from multiple sources (programmatic, YAML, environment variables).
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

// closer is a local interface used for type assertions.
type closer interface {
	Close(context.Context) error
}

// NewBuilder returns a new, empty instance of the fluent builder.
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

// WithConfig applies a configuration object to the builder.
func (b *Builder) WithConfig(cfg *Config) BuilderAPI {
	if cfg != nil {
		b.config = cfg
	}
	return b
}

// WithRetriever programmatically configures the retriever component.
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

// WithVectorStore programmatically configures the vector store.
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

// WithReranker programmatically configures the reranker component.
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

// WithRules programmatically configures the rules engine.
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

// WithLLM programmatically configures the language model client.
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
func (b *Builder) WithFlow(name string) BuilderAPI {
	b.flowName = name
	return b
}

// WithEmbedder programmatically configures the text embedding model.
func (b *Builder) WithEmbedder(opts any) BuilderAPI {
	if opts == nil {
		b.embedderName = ""
		b.embedderParams = nil
		b.embedder = nil // Clear pre-built embedder if opts is nil
		return b
	}

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

// WithTopK programmatically sets the default number of documents to retrieve.
func (b *Builder) WithTopK(k int) BuilderAPI {
	b.opts.TopK = k
	return b
}

// WithMaxTokens programmatically sets the default maximum number of tokens for the LLM response.
func (b *Builder) WithMaxTokens(n int) BuilderAPI {
	b.opts.MaxTokens = n
	return b
}

// WithObservability programmatically sets the observability hooks.
func (b *Builder) WithObservability(obs core.Observability) BuilderAPI {
	b.opts.Obs = obs
	return b
}

// WithFallbackThreshold programmatically sets the confidence score for fallback.
func (b *Builder) WithFallbackThreshold(f float64) BuilderAPI {
	b.opts.FallbackThreshold = f
	return b
}

// WithStateProvider programmatically configures the state provider.
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

// Build constructs the final Orchestrator.
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
		// This part remains complex for now and is out of scope for this refactoring
		b.opts.Obs.Logger.Debugf("building declarative orchestrator")
		// ... existing declarative logic ...
		return nil, errors.New("declarative orchestrator build not implemented in this refactoring")

	case "sandwich":
		b.opts.Obs.Logger.Debugf("building sandwich orchestrator")
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

// buildComponents calls the individual component builders in the correct order.
func (b *Builder) buildComponents(ctx context.Context) error {
	if err := b.resolveDependencies(ctx); err != nil { // Resolve dependencies first
		return err
	}
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

// buildEmbedder is now simple and type-safe.
func (b *Builder) buildEmbedder(ctx context.Context) error {
	if b.embedder != nil || b.embedderName == "" {
		return nil // Already built or not configured
	}
	b.opts.Obs.Logger.Debugf("building embedder %q", b.embedderName)

	factory, err := Get(b.registry.Embedders, b.embedderName)
	if err != nil {
		return err
	}

	deps := make(FactoryDeps)
	if client, ok := b.clients[b.embedderName]; ok { // Or provider family logic
		deps["client"] = client
	}

	embedder, err := factory(ctx, b.embedderParams["typedConfig"], deps)
	if err != nil {
		return fmt.Errorf("factory for embedder '%s' failed: %w", b.embedderName, err)
	}

	b.embedder = embedder // NO type assertion needed
	if c, ok := embedder.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}
	b.opts.Obs.Logger.Infof("initialized embedder: %s", b.embedderName)
	return nil
}

// buildRetriever is now simple and OCP-compliant.
func (b *Builder) buildRetriever(ctx context.Context) error {
	if b.retrieverName == "" {
		return nil // Not configured
	}
	b.opts.Obs.Logger.Debugf("building retriever %q", b.retrieverName)

	factory, err := Get(b.registry.Retrievers, b.retrieverName)
	if err != nil {
		return err
	}

	deps := make(FactoryDeps)
	deps["embedder"] = b.embedder
	deps["vectorStore"] = b.vectorStore
	deps["builder"] = b // For hybrid retriever callback

	retriever, err := factory(ctx, b.retrieverParams["typedConfig"], deps)
	if err != nil {
		return fmt.Errorf("factory for retriever '%s' failed: %w", b.retrieverName, err)
	}

	b.opts.Retriever = retriever // NO type assertion needed
	b.opts.Obs.Logger.Infof("initialized retriever: %s", b.retrieverName)
	return nil
}

// BuildRetriever is the public method for the hybrid factory to call back.
func (b *Builder) BuildRetriever(ctx context.Context, name string, params map[string]any) (retrieve.Retriever, error) {
	b.opts.Obs.Logger.Debugf("building sub-retriever %q for hybrid", name)
	factory, err := Get(b.registry.Retrievers, name)
	if err != nil {
		return nil, err
	}

	deps := make(FactoryDeps)
	deps["embedder"] = b.embedder
	deps["vectorStore"] = b.vectorStore
	// Do NOT pass the builder again to avoid infinite recursion.

	var opts any
	if params != nil {
		opts = params["typedConfig"]
	}

	return factory(ctx, opts, deps)
}


// buildVectorStore is now simple and type-safe.
func (b *Builder) buildVectorStore(ctx context.Context) error {
	if b.vectorStoreName == "" {
		return nil
	}
	b.opts.Obs.Logger.Debugf("building vector store %q", b.vectorStoreName)

	factory, err := Get(b.registry.VectorStores, b.vectorStoreName)
	if err != nil {
		return err
	}

	deps := make(FactoryDeps)
	deps["embedder"] = b.embedder

	vectorStore, err := factory(ctx, b.vectorStoreParams["typedConfig"], deps)
	if err != nil {
		return fmt.Errorf("factory for vector store '%s' failed: %w", b.vectorStoreName, err)
	}

	b.vectorStore = vectorStore
	if c, ok := vectorStore.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}
	b.opts.Obs.Logger.Infof("initialized vector store: %s", b.vectorStoreName)
	return nil
}

// buildReranker is now simple and type-safe.
func (b *Builder) buildReranker(ctx context.Context) error {
	if b.rerankerName == "" {
		return nil
	}
	b.opts.Obs.Logger.Debugf("building reranker %q", b.rerankerName)

	factory, err := Get(b.registry.Rerankers, b.rerankerName)
	if err != nil {
		return err
	}

	deps := make(FactoryDeps)
	deps["embedder"] = b.embedder

	reranker, err := factory(ctx, b.rerankerParams["typedConfig"], deps)
	if err != nil {
		return fmt.Errorf("factory for reranker '%s' failed: %w", b.rerankerName, err)
	}

	b.opts.Reranker = reranker
	b.opts.Obs.Logger.Infof("initialized reranker: %s", b.rerankerName)
	return nil
}

// buildRules is now simple and type-safe.
func (b *Builder) buildRules(ctx context.Context) error {
	if b.rulesName == "" {
		return nil
	}
	b.opts.Obs.Logger.Debugf("building rules engine %q", b.rulesName)

	factory, err := Get(b.registry.RuleSets, b.rulesName)
	if err != nil {
		return err
	}

	deps := make(FactoryDeps)
	deps["registry"] = b.registry

	ruleset, err := factory(ctx, b.rulesParams["typedConfig"], deps)
	if err != nil {
		return fmt.Errorf("factory for ruleset '%s' failed: %w", b.rulesName, err)
	}

	b.opts.Rules = ruleset
	b.opts.Obs.Logger.Infof("initialized rules engine: %s", b.rulesName)
	return nil
}

// buildLLM is now simple and type-safe.
func (b *Builder) buildLLM(ctx context.Context) error {
	if b.llmName == "" {
		return nil
	}
	b.opts.Obs.Logger.Debugf("building llm %q", b.llmName)

	factory, err := Get(b.registry.LLMs, b.llmName)
	if err != nil {
		return err
	}

	deps := make(FactoryDeps)
	if client, ok := b.clients[b.llmName]; ok {
		deps["client"] = client
	}

	llmClient, err := factory(ctx, b.llmParams["typedConfig"], deps)
	if err != nil {
		return fmt.Errorf("factory for llm '%s' failed: %w", b.llmName, err)
	}

	b.opts.LLM = llmClient
	if c, ok := llmClient.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}
	b.opts.Obs.Logger.Infof("initialized llm: %s", b.llmName)
	return nil
}

// buildStateProvider is now simple and type-safe.
func (b *Builder) buildStateProvider(ctx context.Context) error {
	if b.stateProviderName == "" {
		return nil
	}
	b.opts.Obs.Logger.Debugf("building state provider %q", b.stateProviderName)

	factory, err := Get(b.registry.StateProviders, b.stateProviderName)
	if err != nil {
		return err
	}

	provider, err := factory(ctx, b.stateProviderParams["typedConfig"], nil)
	if err != nil {
		return fmt.Errorf("factory for state provider '%s' failed: %w", b.stateProviderName, err)
	}

	b.stateProvider = provider
	if c, ok := provider.(closer); ok {
		b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
	}
	b.opts.Obs.Logger.Infof("initialized state provider: %s", b.stateProviderName)
	return nil
}

// closeResources attempts to release any provider clients that were opened during the build.
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

var embedderAlias = map[string]string{
	"google-embedder": "google",
	"openai-embedder": "openai",
}