package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
)

func RegisterGoogle(r *manglekit.Registry) {
	// The new pattern: we don't register a client factory anymore.
	// We register the LLM factory directly.
	// The Genkit plugin handles the client lifecycle.
	r.RegisterLLM("google", func(ctx context.Context, deps diapi.LLMDeps, cfg any) (llm.Client, error) {
		opts, ok := cfg.(*llm.GoogleOptions)
		if !ok {
			return nil, fmt.Errorf("invalid options type for google llm, expected *llm.GoogleOptions, got %T", cfg)
		}

		if deps.Genkit == nil {
			return nil, fmt.Errorf("missing required dependency 'genkit' of type *genkit.Genkit")
		}

		// In the new genkit model, models are defined and then retrieved.
		// We will define a model based on the options.
		model := googlegenai.GoogleAIModel(deps.Genkit, opts.Model)
		if model == nil {
			// If the model is not already defined (e.g. by a user's explicit genkit.Init),
			// we can try to define it.
			// This part is complex and depends on how Manglekit wants to integrate with Genkit's initialization.
			// For now, we'll assume the model must be predefined in the user's main.go
			// by initializing the genkit with the googleai plugin.
			// A simpler approach for now is to just use the model name.
			return NewGoogle(*opts, nil, deps.Genkit)
		}

		return NewGoogle(*opts, model, deps.Genkit)
	})

	if err := r.RegisterOptions("google", (*llm.GoogleOptions)(nil)); err != nil {
		panic(err)
	}
}

// Google is a wrapper around a genkit AI model.
type Google struct {
	opts   llm.GoogleOptions
	model  ai.Model
	genkit *genkit.Genkit
}

// NewGoogle creates a new Google LLM client.
func NewGoogle(opts llm.GoogleOptions, model ai.Model, g *genkit.Genkit) (llm.Client, error) {
	return &Google{opts: opts, model: model, genkit: g}, nil
}

// Complete implements the llm.Client interface.
func (g *Google) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	if g.model == nil {
		return llm.Response{}, fmt.Errorf("google llm client not initialized with a model")
	}

	// Use the functional options pattern from the new genkit API.
	res, err := genkit.Generate(ctx, g.genkit,
		ai.WithModel(g.model),
		ai.WithPrompt(req.Prompt),
		ai.WithConfig(&ai.GenerationCommonConfig{
			Temperature: float64(g.opts.Temperature),
		}),
	)

	if err != nil {
		return llm.Response{}, err
	}

	return llm.Response{Text: res.Text()}, nil
}

func (g *Google) Model() string {
	return g.opts.Model
}

func (g *Google) GetName() string {
	return "google"
}