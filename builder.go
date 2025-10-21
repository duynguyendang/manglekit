package manglekit

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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
// It is the primary API for programmatically configuring and constructing a pipeline.
type BuilderAPI interface {
	With(opts any) BuilderAPI
	WithTopK(k int) BuilderAPI
	WithMaxTokens(n int) BuilderAPI
	WithObservability(obs core.Observability) BuilderAPI
	WithFallbackThreshold(f float64) BuilderAPI
	WithGenkit(g *genkit.Genkit) BuilderAPI
	WithOrchestrator(name string) BuilderAPI
	Build(ctx context.Context) (core.Orchestrator, retrieve.Updatable, error)
}

// configItem represents a single component configuration added to the builder.
type configItem struct {
	kind   core.Kind
	name   string
	params map[string]any // Holds {"typedConfig": opts}, matching the old structure.
}

// Builder provides a fluent, chainable interface for constructing a MangleKit
// Orchestrator. It is the recommended way to assemble a pipeline, as it handles
// dependency injection, component construction, and configuration resolution.
type Builder struct {
	registry *Registry
	genkit   *genkit.Genkit
	opts     core.OptionsLike
	errs     []error

	// cfgs stores the list of component configurations to be built.
	cfgs []configItem

	// Built components are stored here during the build process.
	// The maps store all named components.
	embedders      map[string]ai.Embedder
	vectorStores   map[string]core.VectorStore
	retrievers     map[string]core.Retriever
	rerankers      map[string]core.Reranker
	rules          map[string]core.RuleSet
	llms           map[string]core.LLMClient
	stateProviders map[string]core.StateProvider

	// Shared clients for dependency injection.
	openAIClient *openai.OpenAI

	orchestratorName string
}

// OpenAIClient implements the diapi.OpenAIClientProvider interface.
func (b *Builder) OpenAIClient() *openai.OpenAI {
	return b.openAIClient
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
	}
	b.opts.Obs.Logger = logger.NewStdLogger()
	return b
}

// With adds a component to the builder using its provider-specific options struct.
// It looks up the provider's kind and name from the registry using the options type.
// This is the primary, type-safe way to configure the builder.
func (b *Builder) With(opts any) BuilderAPI {
	if opts == nil {
		return b
	}
	t := reflect.TypeOf(opts)
	name, ok := b.registry.OptionsTypeToName[t]
	if !ok {
		err := fmt.Errorf("unregistered options type: %T", opts)
		b.errs = append(b.errs, err)
		b.opts.Obs.Logger.Errorf(err.Error())
		return b
	}
	kind, ok := b.registry.OptionsTypeToKind[t]
	if !ok {
		// This should be unreachable if the registry is populated correctly.
		err := fmt.Errorf("internal error: no kind registered for options type %T", opts)
		b.errs = append(b.errs, err)
		b.opts.Obs.Logger.Errorf(err.Error())
		return b
	}

	b.cfgs = append(b.cfgs, configItem{
		kind:   kind,
		name:   name,
		params: map[string]any{"typedConfig": opts},
	})
	return b
}

// compSpec defines the specification for building a single kind of component.
// It's the core of the data-driven, spec-based build process.
type compSpec struct {
	kind           core.Kind
	makeDeps       func(*Builder) any
	assign         func(*Builder, string, any)
	registerCloser func(*Builder, any)
}

// specTable returns the map of all component specifications. The dependency
// graph and build logic are defined here, not in imperative code.
func (b *Builder) specTable() map[core.Kind]compSpec {
	return map[core.Kind]compSpec{
		core.KindEmbedder: {
			kind:     core.KindEmbedder,
			makeDeps: func(b *Builder) any { return diapi.EmbedderDeps{Genkit: b.genkit} },
			assign: func(b *Builder, name string, v any) {
				comp := v.(ai.Embedder)
				b.embedders[name] = comp
			},
			registerCloser: func(b *Builder, v any) {
				if c, ok := v.(interface{ Close(context.Context) error }); ok {
					b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
				}
			},
		},
		core.KindVectorStore: {
			kind: core.KindVectorStore,
			makeDeps: func(b *Builder) any {
				// This is a placeholder. The actual dependency resolution happens
				// inside the buildAll loop, which has access to the component's config.
				return diapi.VectorStoreDeps{}
			},
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.VectorStore)
				b.vectorStores[name] = comp
			},
			registerCloser: func(b *Builder, v any) {
				if c, ok := v.(interface{ Close(context.Context) error }); ok {
					b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
				}
			},
		},
		core.KindRetriever: {
			kind: core.KindRetriever,
			makeDeps: func(b *Builder) any {
				// Placeholder. Dependencies are resolved in the build loop.
				return diapi.RetrieverDeps{}
			},
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.Retriever)
				b.retrievers[name] = comp
			},
		},
		core.KindReranker: {
			kind:     core.KindReranker,
			makeDeps: func(b *Builder) any { return diapi.RerankerDeps{} },
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.Reranker)
				b.rerankers[name] = comp
			},
		},
		core.KindRules: {
			kind:     core.KindRules,
			makeDeps: func(b *Builder) any { return diapi.RuleSetDeps{} },
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.RuleSet)
				b.rules[name] = comp
			},
		},
		core.KindLLM: {
			kind: core.KindLLM,
			makeDeps: func(b *Builder) any {
				return struct {
					diapi.LLMDeps
					diapi.OpenAIClientProvider
				}{
					LLMDeps:              diapi.LLMDeps{Genkit: b.genkit},
					OpenAIClientProvider: b,
				}
			},
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.LLMClient)
				b.llms[name] = comp
			},
			registerCloser: func(b *Builder, v any) {
				if c, ok := v.(interface{ Close(context.Context) error }); ok {
					b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
				}
			},
		},
		core.KindStateProvider: {
			kind:     core.KindStateProvider,
			makeDeps: func(b *Builder) any { return diapi.StateProviderDeps{} },
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.StateProvider)
				b.stateProviders[name] = comp
			},
			registerCloser: func(b *Builder, v any) {
				if c, ok := v.(interface{ Close(context.Context) error }); ok {
					b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
				}
			},
		},
	}
}

// ensureOpenAIClient inspects the config for an LLM provider and creates the shared
// OpenAI client if it's needed and doesn't exist yet.
func (b *Builder) ensureOpenAIClient(c configItem) {
	// Only create the client for providers that are known to need it.
	if c.name != "openai" {
		return
	}
	// If the shared client already exists, do nothing.
	if b.openAIClient != nil {
		return
	}

	cfg := c.params["typedConfig"]
	if provider, ok := cfg.(diapi.APIKeyProvider); ok {
		opts := []option.RequestOption{option.WithAPIKey(provider.GetAPIKey())}
		if baseURLProvider, ok := cfg.(diapi.BaseURLProvider); ok && baseURLProvider.GetBaseURL() != "" {
			opts = append(opts, option.WithBaseURL(baseURLProvider.GetBaseURL()))
		}
		b.openAIClient = &openai.OpenAI{APIKey: provider.GetAPIKey(), Opts: opts}
		b.opts.Obs.Logger.Infof("created shared openai client for %s", c.name)
	}
}

// buildAll constructs all configured components in the correct dependency order.
func (b *Builder) buildAll(ctx context.Context) error {
	specs := b.specTable()

	// Define the explicit build order to manage dependencies.
	order := []core.Kind{
		core.KindEmbedder,
		core.KindVectorStore,
		core.KindRetriever,
		core.KindReranker,
		core.KindRules,
		core.KindLLM,
		core.KindStateProvider,
		core.KindSchemaParser,
	}

	// Group configurations by kind for ordered processing.
	groups := make(map[core.Kind][]configItem)
	for _, c := range b.cfgs {
		groups[c.kind] = append(groups[c.kind], c)
	}

	// Iterate through the build order and construct components.
	for _, k := range order {
		spec, ok := specs[k]
		if !ok {
			continue // No spec for this kind.
		}
		for _, c := range groups[k] {
			b.opts.Obs.Logger.Debugf("building %s %q", k, c.name)
			factory, err := b.registry.Get(k, c.name)
			if err != nil {
				return err
			}

			// Pre-build step: Handle shared client creation.
			if k == core.KindLLM {
				b.ensureOpenAIClient(c)
			}

			var cfg any
			if c.params != nil {
				cfg = c.params["typedConfig"]
			}

			deps, err := b.resolveDeps(ctx, k, cfg)
			if err != nil {
				return err
			}

			built, err := factory.Build(ctx, deps, cfg)
			if err != nil {
				return fmt.Errorf("factory for %s '%s' failed: %w", k, c.name, err)
			}

			spec.assign(b, c.name, built)
			if spec.registerCloser != nil {
				spec.registerCloser(b, built)
			}
			b.opts.Obs.Logger.Infof("initialized %s: %s", k, c.name)
		}
	}
	return nil
}

// resolveDeps is the core of the dependency injection system. It inspects the
// configuration `cfg` for a component of `kind`, determines its dependencies
// by name, looks them up in the builder's maps of already-built components,
// and returns a fully populated `diapi` struct.
func (b *Builder) resolveDeps(ctx context.Context, kind core.Kind, cfg any) (any, error) {
	switch kind {
	case core.KindEmbedder:
		return diapi.EmbedderDeps{Genkit: b.genkit}, nil

	case core.KindVectorStore:
		if typedCfg, ok := cfg.(diapi.EmbedderDep); ok {
			embedder, err := b.getEmbedder(typedCfg.GetEmbedder())
			if err != nil {
				return nil, err
			}
			return diapi.VectorStoreDeps{Embedder: embedder}, nil
		}
		return nil, fmt.Errorf("vector store config does not declare an embedder dependency")

	case core.KindRetriever:
		deps := diapi.RetrieverDeps{}
		if typedCfg, ok := cfg.(diapi.EmbedderDep); ok {
			embedder, err := b.getEmbedder(typedCfg.GetEmbedder())
			if err != nil {
				return nil, err
			}
			deps.Embedder = embedder
		}
		if typedCfg, ok := cfg.(diapi.VectorStoreDep); ok {
			vs, err := b.getVectorStore(typedCfg.GetVectorStore())
			if err != nil {
				return nil, err
			}
			deps.VectorStore = vs
		}
		if typedCfg, ok := cfg.(diapi.SubRetrieversDep); ok {
			for _, name := range typedCfg.GetSubRetrievers() {
				r, err := b.getRetriever(name)
				if err != nil {
					return nil, err
				}
				deps.SubRetrievers = append(deps.SubRetrievers, r)
			}
		}
		return deps, nil

	case core.KindReranker:
		// Currently, the main reranker (Cohere) does not have dependencies.
		return diapi.RerankerDeps{}, nil

	case core.KindRules:
		return diapi.RuleSetDeps{}, nil

	case core.KindLLM:
		return struct {
			diapi.LLMDeps
			diapi.OpenAIClientProvider
		}{
			LLMDeps:              diapi.LLMDeps{Genkit: b.genkit},
			OpenAIClientProvider: b,
		}, nil

	case core.KindStateProvider:
		return diapi.StateProviderDeps{}, nil

	default:
		return nil, nil // No deps
	}
}

func (b *Builder) getEmbedder(name string) (ai.Embedder, error) {
	e, ok := b.embedders[name]
	if !ok {
		return nil, fmt.Errorf("dependency not found: embedder %q", name)
	}
	return e, nil
}

func (b *Builder) getVectorStore(name string) (core.VectorStore, error) {
	vs, ok := b.vectorStores[name]
	if !ok {
		return nil, fmt.Errorf("dependency not found: vectorStore %q", name)
	}
	return vs, nil
}

func (b *Builder) getRetriever(name string) (core.Retriever, error) {
	r, ok := b.retrievers[name]
	if !ok {
		return nil, fmt.Errorf("dependency not found: retriever %q", name)
	}
	return r, nil
}

// Build constructs the final Orchestrator.
func (b *Builder) Build(ctx context.Context) (core.Orchestrator, retrieve.Updatable, error) {
	if b.orchestratorName == "" {
		return nil, nil, errors.New("no orchestrator specified in configuration")
	}
	orchestratorName := b.orchestratorName
	b.opts.Obs.Logger.Infof("starting build for orchestrator type %q", orchestratorName)

	if len(b.errs) > 0 {
		err := errors.Join(b.errs...)
		b.opts.Obs.Logger.Errorf("pre-build validation failed: %v", err)
		return nil, nil, err
	}

	if err := b.buildAll(ctx); err != nil {
		closeErr := b.closeResources(ctx)
		b.opts.Obs.Logger.Errorf("failed to build components: %v", err)
		return nil, nil, errors.Join(err, closeErr)
	}

	// Assemble the strongly-typed Resolved struct for the orchestrator.
	resolved := core.Resolved{
		Retrievers:        b.retrievers,
		VectorStores:      b.vectorStores,
		Rerankers:         b.rerankers,
		Rules:             b.rules,
		LLMs:              b.llms,
		Embedders:         b.embedders,
		StateProviders:    b.stateProviders,
		Obs:               b.opts.Obs,
		TopK:              b.opts.TopK,
		MaxTokens:         b.opts.MaxTokens,
		FallbackThreshold: b.opts.FallbackThreshold,
		Closers:           b.opts.ResourceClosers,
	}

	// Build the orchestrator itself.
	factory, err := b.registry.Get(core.KindOrchestrator, orchestratorName)
	if err != nil {
		closeErr := b.closeResources(ctx)
		err = fmt.Errorf("unknown orchestrator %q", orchestratorName)
		b.opts.Obs.Logger.Errorf(err.Error())
		return nil, nil, errors.Join(err, closeErr)
	}

	// The orchestrator factory expects `core.Resolved` as its dependency.
	// Orchestrators typically don't have their own config, so `cfg` is nil.
	orchAny, err := factory.Build(ctx, resolved, nil)
	if err != nil {
		closeErr := b.closeResources(ctx)
		b.opts.Obs.Logger.Errorf("factory for orchestrator %q failed: %v", orchestratorName, err)
		return nil, nil, errors.Join(err, closeErr)
	}
	orchestrator := orchAny.(core.Orchestrator)

	// Check if any retriever is updatable and return a typed handle to the first one found.
	var updatable retrieve.Updatable
	for _, r := range b.retrievers {
		if u, ok := r.(retrieve.Updatable); ok {
			updatable = u
			b.opts.Obs.Logger.Infof("found updatable retriever")
			break
		}
	}

	b.opts.Obs.Logger.Infof("successfully built %s orchestrator", orchestratorName)
	return orchestrator, updatable, nil
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

// Fluent setters for global options.
func (b *Builder) WithTopK(k int) BuilderAPI                           { b.opts.TopK = k; return b }
func (b *Builder) WithMaxTokens(n int) BuilderAPI                      { b.opts.MaxTokens = n; return b }
func (b *Builder) WithObservability(obs core.Observability) BuilderAPI { b.opts.Obs = obs; return b }
func (b *Builder) WithFallbackThreshold(f float64) BuilderAPI {
	b.opts.FallbackThreshold = f
	return b
}
func (b *Builder) WithGenkit(g *genkit.Genkit) BuilderAPI { b.genkit = g; return b }
func (b *Builder) WithOrchestrator(name string) BuilderAPI { b.orchestratorName = name; return b }
