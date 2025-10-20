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
)

// BuilderAPI defines the fluent interface for the MangleKit orchestrator builder.
// It is the primary API for programmatically configuring and constructing a pipeline.
type BuilderAPI interface {
	With(opts any) BuilderAPI
	WithKind(kind core.Kind, name string, opts any) BuilderAPI
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

	// The single-instance fields hold the *last built* component of their kind.
	// This provides a default dependency for other components during the build process
	// without requiring a full dependency-by-name resolution system.
	embedder      ai.Embedder
	vectorStore   core.VectorStore
	retriever     core.Retriever
	reranker      core.Reranker
	ruleSet       core.RuleSet
	llmClient     core.LLMClient
	stateProvider core.StateProvider

	orchestratorName string
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
	name, ok := b.registry.optionsTypeToName[t]
	if !ok {
		err := fmt.Errorf("unregistered options type: %T", opts)
		b.errs = append(b.errs, err)
		b.opts.Obs.Logger.Errorf(err.Error())
		return b
	}
	kind, ok := b.registry.optionsTypeToKind[t]
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

// WithKind adds a component to the builder using an explicit kind, name, and options.
// This is primarily used by the config loader, which reads kind and name strings
// from a YAML file.
func (b *Builder) WithKind(kind core.Kind, name string, opts any) BuilderAPI {
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
				b.embedder = comp // Keep track of the last one for dependency injection.
			},
			registerCloser: func(b *Builder, v any) {
				if c, ok := v.(interface{ Close(context.Context) error }); ok {
					b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
				}
			},
		},
		core.KindVectorStore: {
			kind:     core.KindVectorStore,
			makeDeps: func(b *Builder) any { return diapi.VectorStoreDeps{Embedder: b.embedder} },
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.VectorStore)
				b.vectorStores[name] = comp
				b.vectorStore = comp
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
				return diapi.RetrieverDeps{
					Embedder:          b.embedder,
					VectorStore:       b.vectorStore,
					BuildSubRetriever: b.BuildRetriever,
				}
			},
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.Retriever)
				b.retrievers[name] = comp
				b.retriever = comp
			},
		},
		core.KindReranker: {
			kind:     core.KindReranker,
			makeDeps: func(b *Builder) any { return diapi.RerankerDeps{Embedder: b.embedder} },
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.Reranker)
				b.rerankers[name] = comp
				b.reranker = comp
			},
		},
		core.KindRules: {
			kind:     core.KindRules,
			makeDeps: func(b *Builder) any { return diapi.RuleSetDeps{} },
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.RuleSet)
				b.rules[name] = comp
				b.ruleSet = comp
			},
		},
		core.KindLLM: {
			kind:     core.KindLLM,
			makeDeps: func(b *Builder) any { return diapi.LLMDeps{Genkit: b.genkit} },
			assign: func(b *Builder, name string, v any) {
				comp := v.(core.LLMClient)
				b.llms[name] = comp
				b.llmClient = comp
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
				b.stateProvider = comp
			},
			registerCloser: func(b *Builder, v any) {
				if c, ok := v.(interface{ Close(context.Context) error }); ok {
					b.opts.ResourceClosers = append(b.opts.ResourceClosers, c.Close)
				}
			},
		},
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

			deps := spec.makeDeps(b)
			var cfg any
			if c.params != nil {
				cfg = c.params["typedConfig"]
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

// BuildRetriever constructs a retriever by name. This is used to support the
// hybrid retriever pattern, which needs to build its sub-retrievers.
func (b *Builder) BuildRetriever(ctx context.Context, name string, params map[string]any) (core.Retriever, error) {
	// This function is tricky. It needs to build a sub-component that might not
	// have been in the original config list. We look it up in the registry,
	// create its dependencies, and build it, but we *don't* add it to the main
	// builder's component maps, as it's a dependency of another component.
	b.opts.Obs.Logger.Debugf("building sub-retriever %q", name)
	factory, err := b.registry.Get(core.KindRetriever, name)
	if err != nil {
		return nil, err
	}

	// This is a critical assumption: the sub-retriever depends on the *last-built*
	// embedder and vector store.
	deps := diapi.RetrieverDeps{
		Embedder:    b.embedder,
		VectorStore: b.vectorStore,
	}

	var cfg any
	if params != nil {
		cfg = params["typedConfig"]
	}

	v, err := factory.Build(ctx, deps, cfg)
	if err != nil {
		return nil, err
	}
	return v.(core.Retriever), nil
}

// Build constructs the final Orchestrator.
func (b *Builder) Build(ctx context.Context) (core.Orchestrator, retrieve.Updatable, error) {
	orchestratorName := b.orchestratorName
	if orchestratorName == "" {
		orchestratorName = "sandwich" // Default orchestrator
	}
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

	// Check if the retriever is updatable and return a typed handle if so.
	// This now checks the *last-built* retriever for simplicity. A more advanced
	// system might need to return multiple updatable handles.
	var updatable retrieve.Updatable
	if b.retriever != nil {
		if u, ok := b.retriever.(retrieve.Updatable); ok {
			updatable = u
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