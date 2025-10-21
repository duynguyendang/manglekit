package manglekit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
	"github.com/openai/openai-go/option"
)

// BuilderAPI defines the fluent interface for the MangleKit orchestrator builder.
type BuilderAPI interface {
	With(name string, opts any) BuilderAPI
	WithTopK(k int) BuilderAPI
	WithMaxTokens(n int) BuilderAPI
	WithObservability(obs core.Observability) BuilderAPI
	WithFallbackThreshold(f float64) BuilderAPI
	WithGenkit(g *genkit.Genkit) BuilderAPI
	WithOrchestrator(name string) BuilderAPI
	WithUpdatable(name string) BuilderAPI
	Build(ctx context.Context) (core.Orchestrator, retrieve.Updatable, error)
}

type configItem struct {
	kind core.Kind
	name string
	cfg  core.ProviderOptions
}

// Builder provides a fluent, chainable interface for constructing a MangleKit Orchestrator.
type Builder struct {
	registry *Registry
	genkit   *genkit.Genkit
	opts     core.OptionsLike
	errs     []error
	cfgs     []configItem

	embedders      map[string]ai.Embedder
	vectorStores   map[string]core.VectorStore
	retrievers     map[string]core.Retriever
	rerankers      map[string]core.Reranker
	rules          map[string]core.RuleSet
	llms           map[string]core.LLMClient
	stateProviders map[string]core.StateProvider
	orchestrators  map[string]core.Orchestrator

	openAIClient *openai.OpenAI

	orchestratorName string
	updatableName    string
}

// NewBuilder returns a new, empty instance of the fluent builder.
func NewBuilder(r *Registry) *Builder {
	b := &Builder{
		registry:       r,
		embedders:      make(map[string]ai.Embedder),
		vectorStores:   make(map[string]core.VectorStore),
		retrievers:     make(map[string]core.Retriever),
		rerankers:      make(map[string]core.Reranker),
		rules:          make(map[string]core.RuleSet),
		llms:           make(map[string]core.LLMClient),
		stateProviders: make(map[string]core.StateProvider),
		orchestrators:  make(map[string]core.Orchestrator),
	}
	b.opts.Obs.Logger = logger.NewStdLogger()
	return b
}

// DI implementation
func (b *Builder) OpenAIClient() *openai.OpenAI    { return b.openAIClient }
func (b *Builder) Genkit() *genkit.Genkit          { return b.genkit }
func (b *Builder) GetEmbedder(n string) (ai.Embedder, error)   { return getComponent(b.embedders, n) }
func (b *Builder) GetVectorStore(n string) (core.VectorStore, error) { return getComponent(b.vectorStores, n) }
func (b *Builder) GetRetriever(n string) (core.Retriever, error)   { return getComponent(b.retrievers, n) }

func (b *Builder) With(name string, opts any) BuilderAPI {
	if opts == nil {
		return b
	}
	providerOpts, ok := opts.(core.ProviderOptions)
	if !ok {
		b.errs = append(b.errs, fmt.Errorf("type %T does not implement core.ProviderOptions", opts))
		return b
	}

	b.cfgs = append(b.cfgs, configItem{
		kind: providerOpts.ProviderKind(),
		name: name,
		cfg:  providerOpts,
	})
	return b
}

func (b *Builder) EnsureOpenAIClient(cfg core.ProviderOptions) error {
	if b.openAIClient != nil {
		return nil
	}

	provider, ok := cfg.(diapi.APIKeyProvider)
	if !ok {
		return fmt.Errorf("OpenAI provider config does not implement APIKeyProvider")
	}

	opts := []option.RequestOption{option.WithAPIKey(provider.GetAPIKey())}
	if baseURLProvider, ok := cfg.(diapi.BaseURLProvider); ok && baseURLProvider.GetBaseURL() != "" {
		opts = append(opts, option.WithBaseURL(baseURLProvider.GetBaseURL()))
	}
	b.openAIClient = &openai.OpenAI{APIKey: provider.GetAPIKey(), Opts: opts}
	b.opts.Obs.Logger.Infof("created shared openai client")
	return nil
}

func (b *Builder) buildAll(ctx context.Context) error {
	order := []core.Kind{
		core.KindEmbedder,
		core.KindVectorStore,
		core.KindRetriever,
		core.KindReranker,
		core.KindRules,
		core.KindLLM,
		core.KindStateProvider,
		core.KindSchemaParser,
		core.KindOrchestrator,
	}

	groups := make(map[core.Kind][]configItem)
	for _, c := range b.cfgs {
		groups[c.kind] = append(groups[c.kind], c)
	}

	resolved := &core.Resolved{
		Retrievers:     b.retrievers,
		VectorStores:   b.vectorStores,
		Rerankers:      b.rerankers,
		Rules:          b.rules,
		LLMs:           b.llms,
		Embedders:      b.embedders,
		StateProviders: b.stateProviders,
		Orchestrators:  b.orchestrators,
	}

	for _, k := range order {
		handler, err := b.registry.GetHandler(k)
		if err != nil {
			continue // No handler for this kind.
		}

		for _, c := range groups[k] {
			b.opts.Obs.Logger.Debugf("building %s %q", k, c.name)
			// Use the provider name from the config to look up the factory.
			factory, err := b.registry.Get(k, c.cfg.ProviderName())
			if err != nil {
				return err
			}

			closer, err := handler.BuildComponent(ctx, b, factory, resolved, c.cfg, c.name)
			if err != nil {
				return err
			}
			if closer != nil {
				b.opts.ResourceClosers = append(b.opts.ResourceClosers, closer)
			}
			b.opts.Obs.Logger.Infof("initialized %s: %s", k, c.name)
		}
	}
	return nil
}

func (b *Builder) Build(ctx context.Context) (core.Orchestrator, retrieve.Updatable, error) {
	if b.orchestratorName == "" {
		return nil, nil, errors.New("no orchestrator specified in configuration")
	}
	b.opts.Obs.Logger.Infof("starting build for orchestrator type %q", b.orchestratorName)

	if len(b.errs) > 0 {
		return nil, nil, errors.Join(b.errs...)
	}

	if err := b.buildAll(ctx); err != nil {
		closeErr := b.closeResources(ctx)
		return nil, nil, errors.Join(err, closeErr)
	}

	orchestrator, ok := b.orchestrators[b.orchestratorName]
	if !ok {
		return nil, nil, fmt.Errorf("orchestrator %q not found", b.orchestratorName)
	}

	var updatable retrieve.Updatable
	if b.updatableName != "" {
		r, ok := b.retrievers[b.updatableName]
		if !ok {
			return nil, nil, fmt.Errorf("updatable component %q not found", b.updatableName)
		}

		u, ok := r.(retrieve.Updatable)
		if !ok {
			return nil, nil, fmt.Errorf("component %q was found, but it does not implement retrieve.Updatable", b.updatableName)
		}
		updatable = u
	}

	b.opts.Obs.Logger.Infof("successfully built %s orchestrator", b.orchestratorName)
	return orchestrator, updatable, nil
}

func (b *Builder) closeResources(ctx context.Context) error {
	closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var combined error
	for i := len(b.opts.ResourceClosers) - 1; i >= 0; i-- {
		if err := b.opts.ResourceClosers[i](closeCtx); err != nil {
			combined = errors.Join(combined, err)
		}
	}
	return combined
}

func (b *Builder) WithTopK(k int) BuilderAPI              { b.opts.TopK = k; return b }
func (b *Builder) WithMaxTokens(n int) BuilderAPI         { b.opts.MaxTokens = n; return b }
func (b *Builder) WithObservability(obs core.Observability) BuilderAPI { b.opts.Obs = obs; return b }
func (b *Builder) WithFallbackThreshold(f float64) BuilderAPI { b.opts.FallbackThreshold = f; return b }
func (b *Builder) WithGenkit(g *genkit.Genkit) BuilderAPI    { b.genkit = g; return b }
func (b *Builder) WithOrchestrator(name string) BuilderAPI  { b.orchestratorName = name; return b }
func (b *Builder) WithUpdatable(name string) BuilderAPI    { b.updatableName = name; return b }

func getComponent[T any](m map[string]T, name string) (T, error) {
	c, ok := m[name]
	if !ok {
		var zero T
		return zero, fmt.Errorf("dependency not found: %s", name)
	}
	return c, nil
}
