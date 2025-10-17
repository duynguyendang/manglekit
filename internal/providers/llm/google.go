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
	manglekit.Register(r, llm.GoogleOptions{},
		func(ctx context.Context, deps diapi.LLMDeps, cfg llm.GoogleOptions) (llm.Client, error) {
			if deps.Genkit == nil {
				return nil, fmt.Errorf("missing required dependency 'genkit' of type *genkit.Genkit")
			}

			model := googlegenai.GoogleAIModel(deps.Genkit, cfg.Model)
			if model == nil {
				// Fallback for when the model isn't pre-registered in genkit init.
				return NewGoogle(cfg, nil, deps.Genkit)
			}

			return NewGoogle(cfg, model, deps.Genkit)
		},
	)
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
