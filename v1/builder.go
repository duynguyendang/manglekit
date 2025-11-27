package manglekit

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"reflect"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	"github.com/duynguyendang/manglekit/v1/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/mitchellh/mapstructure"
)

type configItem struct {
	kind core.Kind
	name string
	cfg  core.ProviderOptions
}

// builder provides a fluent, chainable interface for constructing a MangleKit Orchestrator.
//
// ⚠️  THREAD SAFETY WARNING: builder is NOT thread-safe and MUST be used by only one goroutine.
//
// The builder maintains 11 unprotected component maps (embedders, vectorStores, retrievers, etc.)
// and should only be accessed from a single goroutine. If you need to construct orchestrators
// from multiple goroutines, create separate builder instances for each goroutine.
//
// Correct Usage:
//
//	// Single goroutine usage (thread-safe)
//	b := NewBuilder(ctx, registry, obs, genkit)
//	b.WithOptions("component1", opts1)
//	b.WithOptions("component2", opts2)
//	orch, closer, err := b.Build(ctx, "orchestrator-name", "")
//
// Incorrect Usage (NOT thread-safe):
//
//	// ❌ DON'T do this - multiple goroutines accessing the same builder
//	b := NewBuilder(...)
//	go b.WithOptions("component1", opts1)  // Race condition!
//	go b.WithOptions("component2", opts2)  // Race condition!
//
// If you need concurrent construction:
//
//	// ✓ Create a new builder for each goroutine
//	go func() {
//	    b := NewBuilder(...)  // New builder instance
//	    b.WithOptions("component1", opts1)
//	    orch, _, _ := b.Build(...)
//	}()
type builder struct {
	registry *Registry
	genkit   *genkit.Genkit
	opts     core.OptionsLike
	errs     []error
	cfgs     []configItem

	embedders      map[string]ai.Embedder
	retrievers     map[string]core.Retriever
	rerankers      map[string]core.Reranker
	rules          map[string]core.RuleSet
	llms           map[string]core.LLMClient
	stateProviders map[string]core.StateProvider
	orchestrators  map[string]core.Orchestrator
	schemaParsers  map[string]core.SchemaParser
	tools          map[string]core.Tool
	reasoners      map[string]core.Reasoner
	planners       map[string]core.Planner

	// dependencyRegistry maps provider names to their required configurations
	dependencyRegistry *core.ProviderDependencyRegistry
}

// NewBuilder returns a new, empty instance of the fluent builder.
func NewBuilder(ctx context.Context, r *Registry, obs core.Observability, g *genkit.Genkit) (ProgrammaticBuilder, error) {
	if obs.Logger == nil {
		return nil, errors.New("observability logger cannot be nil")
	}
	if g == nil {
		return nil, errors.New("genkit cannot be nil")
	}
	b := &builder{
		registry:           r,
		genkit:             g,
		embedders:          make(map[string]ai.Embedder),
		retrievers:         make(map[string]core.Retriever),
		rerankers:          make(map[string]core.Reranker),
		rules:              make(map[string]core.RuleSet),
		llms:               make(map[string]core.LLMClient),
		stateProviders:     make(map[string]core.StateProvider),
		orchestrators:      make(map[string]core.Orchestrator),
		schemaParsers:      make(map[string]core.SchemaParser),
		tools:              make(map[string]core.Tool),
		reasoners:          make(map[string]core.Reasoner),
		planners:           make(map[string]core.Planner),
		dependencyRegistry: core.NewProviderDependencyRegistry(),
	}
	b.opts.Obs = obs
	return b, nil
}

// DI implementation
func (b *builder) Registry() any                                 { return b.registry }
func (b *builder) Genkit() *genkit.Genkit                        { return b.genkit }
func (b *builder) GetEmbedder(n string) (ai.Embedder, error)     { return getComponent(b.embedders, n) }
func (b *builder) GetLLMClient(n string) (core.LLMClient, error) { return getComponent(b.llms, n) }
func (b *builder) GetRetriever(n string) (core.Retriever, error) {
	return getComponent(b.retrievers, n)
}
func (b *builder) GetReranker(n string) (core.Reranker, error) { return getComponent(b.rerankers, n) }
func (b *builder) GetStateProvider(n string) (core.StateProvider, error) {
	return getComponent(b.stateProviders, n)
}
func (b *builder) GetRuleSet(n string) (core.RuleSet, error) { return getComponent(b.rules, n) }
func (b *builder) GetSchemaParser(n string) (core.SchemaParser, error) {
	return getComponent(b.schemaParsers, n)
}
func (b *builder) GetReasoner(n string) (core.Reasoner, error) { return getComponent(b.reasoners, n) }
func (b *builder) GetPlanner(n string) (core.Planner, error)   { return getComponent(b.planners, n) }

func (b *builder) SetRetriever(name string, retriever core.Retriever) error {
	b.retrievers[name] = retriever
	return nil
}

func (b *builder) GetCoreDeps() diapi.CoreDeps {
	return diapi.CoreDeps{
		Obs: b.opts.Obs,
	}
}

func (b *builder) buildAll(ctx context.Context) error {
	order := []core.Kind{
		core.KindEmbedder,
		core.KindRetriever,
		core.KindReranker,
		core.KindRules,
		core.KindLLM,
		core.KindStateProvider,
		core.KindSchemaParser,
		core.KindTool,
		core.KindReasoner,
		core.KindPlanner,
		core.KindOrchestrator,
	}

	groups := make(map[core.Kind][]configItem)
	for _, c := range b.cfgs {
		groups[c.kind] = append(groups[c.kind], c)
	}

	resolved := &core.Resolved{
		Retrievers:     b.retrievers,
		Rerankers:      b.rerankers,
		Rules:          b.rules,
		LLMs:           b.llms,
		Embedders:      b.embedders,
		StateProviders: b.stateProviders,
		Orchestrators:  b.orchestrators,
		SchemaParsers:  b.schemaParsers,
		Tools:          b.tools,
		Reasoners:      b.reasoners,
		Planners:       b.planners,
		Obs:            b.opts.Obs,
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
				return fmt.Errorf("failed to get factory for %s %q: %w", k, c.cfg.ProviderName(), err)
			}

			closer, err := handler.BuildComponent(ctx, b, factory, resolved, c.cfg, c.name)
			if err != nil {
				return fmt.Errorf("failed to build component %s %q: %w", k, c.name, err)
			}
			if closer != nil {
				b.opts.ResourceClosers = append(b.opts.ResourceClosers, closer)
			}
			b.opts.Obs.Logger.Infof("initialized %s: %s", k, c.name)
		}
	}
	return nil
}

func (b *builder) WithOptions(name string, opts core.ProviderOptions) ProgrammaticBuilder {
	b.cfgs = append(b.cfgs, configItem{
		kind: opts.ProviderKind(),
		name: name,
		cfg:  opts,
	})

	// Validate that provider has required environment variables
	// This provides early feedback during configuration rather than at build time
	if err := b.dependencyRegistry.ValidateProvider(name); err != nil {
		b.errs = append(b.errs, err)
	}

	return b
}

func (b *builder) Build(ctx context.Context, orchestratorName, updatableName string) (core.Orchestrator, retrieve.Updatable, error) {
	if orchestratorName == "" {
		return nil, nil, errors.New("no orchestrator specified in configuration")
	}
	b.opts.Obs.Logger.Infof("starting build for orchestrator type %q", orchestratorName)

	if len(b.errs) > 0 {
		return nil, nil, errors.Join(b.errs...)
	}

	if err := b.buildAll(ctx); err != nil {
		closeErr := b.closeResources(ctx)
		return nil, nil, errors.Join(err, closeErr)
	}

	orchestrator, ok := b.orchestrators[orchestratorName]
	if !ok {
		return nil, nil, fmt.Errorf("orchestrator %q not found", orchestratorName)
	}

	var updatable retrieve.Updatable
	if updatableName != "" {
		r, ok := b.retrievers[updatableName]
		if !ok {
			return nil, nil, fmt.Errorf("updatable component %q not found", updatableName)
		}

		u, ok := r.(retrieve.Updatable)
		if !ok {
			return nil, nil, fmt.Errorf("component %q was found, but it does not implement retrieve.Updatable", updatableName)
		}
		updatable = u
	}

	b.opts.Obs.Logger.Infof("successfully built %s orchestrator", orchestratorName)
	return orchestrator, updatable, nil
}

func (b *builder) closeResources(ctx context.Context) error {
	timeout := b.opts.ResourceCleanupTimeout
	if timeout == 0 {
		timeout = 5 * time.Second // Default to 5 seconds if not configured
	}
	closeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var combined error
	for i := len(b.opts.ResourceClosers) - 1; i >= 0; i-- {
		if err := b.opts.ResourceClosers[i](closeCtx); err != nil {
			b.opts.Obs.Logger.Warnf("resource cleanup failed",
				"closer_index", i,
				"total_closers", len(b.opts.ResourceClosers),
				"error", err.Error())
			combined = errors.Join(combined, err)
		} else {
			b.opts.Obs.Logger.Debugf("resource closed successfully",
				"closer_index", i,
				"total_closers", len(b.opts.ResourceClosers))
		}
	}
	if combined != nil {
		b.opts.Obs.Logger.Errorf("resource cleanup completed with errors",
			"error", combined.Error())
	}
	return combined
}

func (b *builder) fromConfig(ctx context.Context, data []byte) (core.Orchestrator, retrieve.Updatable, error) {
	cfg, err := config.ParseConfig(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	for _, comp := range cfg.Components {
		if comp.Type == "" {
			return nil, nil, fmt.Errorf("component %q is missing required field 'type'", comp.Name)
		}
		var foundType reflect.Type

		// Get all types and sort them for deterministic iteration.
		var types []reflect.Type
		for t := range b.registry.OptionsTypeToName {
			types = append(types, t)
		}
		sort.Slice(types, func(i, j int) bool {
			return types[i].String() < types[j].String()
		})

		for _, t := range types {
			name := b.registry.OptionsTypeToName[t]
			if name == comp.Type && b.registry.OptionsTypeToKind[t] == comp.Kind {
				foundType = t
				break
			}
		}

		if foundType == nil {
			return nil, nil, fmt.Errorf("could not find options type for kind=%s, type=%s", comp.Kind, comp.Type)
		}

		// Create a new instance of the options struct.
		var optsPtr reflect.Value
		if foundType.Kind() == reflect.Ptr {
			optsPtr = reflect.New(foundType.Elem())
		} else {
			optsPtr = reflect.New(foundType)
		}
		opts := optsPtr.Interface()

		// Unmarshal the YAML params into the new options struct.
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:           opts,
			WeaklyTypedInput: true,
			TagName:          "yaml",
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create mapstructure decoder: %w", err)
		}
		if err := decoder.Decode(comp.Params); err != nil {
			return nil, nil, fmt.Errorf("failed to decode params for %s '%s': %w", comp.Kind, comp.Name, err)
		}

		providerOpts, ok := opts.(core.ProviderOptions)
		if !ok {
			b.errs = append(b.errs, fmt.Errorf("type %T does not implement core.ProviderOptions", opts))
		} else {
			b.cfgs = append(b.cfgs, configItem{
				kind: providerOpts.ProviderKind(),
				name: comp.Name,
				cfg:  providerOpts,
			})
		}
	}

	return b.Build(ctx, cfg.Orchestrator, cfg.Updatable)
}

func getComponent[T any](m map[string]T, name string) (T, error) {
	c, ok := m[name]
	if !ok {
		var zero T
		return zero, fmt.Errorf("dependency not found: %s", name)
	}
	return c, nil
}
