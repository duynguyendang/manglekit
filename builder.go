package manglekit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

// BuilderAPI defines the fluent interface for the MangleKit orchestrator builder.
type BuilderAPI interface {
	With(name string, opts any) BuilderAPI
	WithHandlers(handlers ...core.ComponentHandler) BuilderAPI
	FromConfig(ctx context.Context, data []byte) (core.Orchestrator, retrieve.Updatable, error)
	Build(ctx context.Context, orchestratorName, updatableName string) (core.Orchestrator, retrieve.Updatable, error)
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
	schemaParsers  map[string]core.SchemaParser
}

// NewBuilder returns a new, empty instance of the fluent builder.
func NewBuilder(ctx context.Context, r *Registry, obs core.Observability, g *genkit.Genkit) (*Builder, error) {
	if obs.Logger == nil {
		return nil, errors.New("observability logger cannot be nil")
	}
	if g == nil {
		return nil, errors.New("genkit cannot be nil")
	}
	b := &Builder{
		registry:       r,
		genkit:         g,
		embedders:      make(map[string]ai.Embedder),
		vectorStores:   make(map[string]core.VectorStore),
		retrievers:     make(map[string]core.Retriever),
		rerankers:      make(map[string]core.Reranker),
		rules:          make(map[string]core.RuleSet),
		llms:           make(map[string]core.LLMClient),
		stateProviders: make(map[string]core.StateProvider),
		orchestrators:  make(map[string]core.Orchestrator),
		schemaParsers:  make(map[string]core.SchemaParser),
	}
	b.opts.Obs = obs
	return b, nil
}

// DI implementation
func (b *Builder) Genkit() *genkit.Genkit                        { return b.genkit }
func (b *Builder) GetEmbedder(n string) (ai.Embedder, error) { return getComponent(b.embedders, n) }
func (b *Builder) GetLLMClient(n string) (core.LLMClient, error) { return getComponent(b.llms, n) }
func (b *Builder) GetVectorStore(n string) (core.VectorStore, error) {
	return getComponent(b.vectorStores, n)
}
func (b *Builder) GetRetriever(n string) (core.Retriever, error) { return getComponent(b.retrievers, n) }
func (b *Builder) GetReranker(n string) (core.Reranker, error)   { return getComponent(b.rerankers, n) }
func (b *Builder) GetStateProvider(n string) (core.StateProvider, error) {
	return getComponent(b.stateProviders, n)
}
func (b *Builder) GetRuleSet(n string) (core.RuleSet, error) { return getComponent(b.rules, n) }
func (b *Builder) GetSchemaParser(n string) (core.SchemaParser, error) {
	return getComponent(b.schemaParsers, n)
}

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
		SchemaParsers:  b.schemaParsers,
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

func (b *Builder) Build(ctx context.Context, orchestratorName, updatableName string) (core.Orchestrator, retrieve.Updatable, error) {
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

func (b *Builder) WithHandlers(handlers ...core.ComponentHandler) BuilderAPI {
	for _, h := range handlers {
		b.registry.RegisterHandler(h)
	}
	return b
}

func (b *Builder) FromConfig(ctx context.Context, data []byte) (core.Orchestrator, retrieve.Updatable, error) {
	cfg, err := config.ParseConfig(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	for _, comp := range cfg.Components {
		if comp.Type == "" {
			return nil, nil, fmt.Errorf("component %q is missing required field 'type'", comp.Name)
		}
		var foundType reflect.Type
		for t, name := range b.registry.OptionsTypeToName {
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

		b.With(comp.Name, opts)
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
