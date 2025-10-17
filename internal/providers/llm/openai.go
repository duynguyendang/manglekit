package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/compat_oai/openai"
)

func RegisterOpenAI(r *manglekit.Registry) {
	factory := func(ctx context.Context, deps diapi.LLMDeps, cfg any) (llm.Client, error) {
		opts, ok := cfg.(*llm.OpenAIOptions)
		if !ok {
			return nil, fmt.Errorf("invalid options type, expected *llm.OpenAIOptions, got %T", cfg)
		}
		if deps.Genkit == nil {
			return nil, fmt.Errorf("missing required dependency 'genkit'")
		}

		// The Genkit OpenAI plugin is initialized differently. We create the plugin instance
		// and then get the model from it. The plugin itself is not registered with Genkit
		// in the same way as other plugins.
		oai := &openai.OpenAI{
			APIKey: opts.APIKey,
		}
		model := oai.Model(deps.Genkit, opts.Model)
		if model == nil {
			return nil, fmt.Errorf("failed to get openai model %q from genkit", opts.Model)
		}

		return NewOpenAI(*opts, model, deps.Genkit), nil
	}
	r.RegisterLLM("openai", factory)
	r.RegisterLLM("groq", factory) // Groq uses the same machinery
	if err := r.RegisterOptions("openai", (*llm.OpenAIOptions)(nil)); err != nil {
		panic(err)
	}
	if err := r.RegisterOptions("groq", (*llm.OpenAIOptions)(nil)); err != nil {
		panic(err)
	}
}

// OpenAI is a wrapper around a genkit AI model from the OpenAI plugin.
type OpenAI struct {
	opts   llm.OpenAIOptions
	model  ai.Model
	genkit *genkit.Genkit
}

// NewOpenAI is the constructor for the OpenAI client wrapper.
func NewOpenAI(opts llm.OpenAIOptions, model ai.Model, g *genkit.Genkit) llm.Client {
	return &OpenAI{
		opts:   opts,
		model:  model,
		genkit: g,
	}
}

// Complete implements the llm.Client interface.
func (o *OpenAI) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	if o.model == nil {
		return llm.Response{}, fmt.Errorf("openai llm client not initialized with a model")
	}

	// Use the standard genkit.Generate function.
	res, err := genkit.Generate(ctx, o.genkit,
		ai.WithModel(o.model),
		ai.WithPrompt(req.Prompt),
		ai.WithConfig(&ai.GenerationCommonConfig{
			Temperature: float64(o.opts.Temperature),
			MaxOutputTokens: o.opts.MaxOutputTokens,
		}),
	)
	if err != nil {
		return llm.Response{}, err
	}

	return llm.Response{Text: res.Text()}, nil
}
