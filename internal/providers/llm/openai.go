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
	// Factory function for OpenAI
	openAIFactory := func(ctx context.Context, deps diapi.LLMDeps, cfg llm.OpenAIOptions) (llm.Client, error) {
		if deps.Genkit == nil {
			return nil, fmt.Errorf("missing required dependency 'genkit'")
		}
		oai := &openai.OpenAI{APIKey: cfg.APIKey}
		model := oai.Model(deps.Genkit, cfg.Model)
		if model == nil {
			return nil, fmt.Errorf("failed to get openai model %q from genkit", cfg.Model)
		}
		return NewOpenAI(cfg, model, deps.Genkit), nil
	}
	manglekit.Register(r, llm.OpenAIOptions{}, openAIFactory)

	// Factory function for Groq (reuses OpenAI logic but with GroqOptions)
	groqFactory := func(ctx context.Context, deps diapi.LLMDeps, cfg llm.GroqOptions) (llm.Client, error) {
		if deps.Genkit == nil {
			return nil, fmt.Errorf("missing required dependency 'genkit'")
		}
		oai := &openai.OpenAI{APIKey: cfg.APIKey}
		model := oai.Model(deps.Genkit, cfg.Model)
		if model == nil {
			return nil, fmt.Errorf("failed to get groq model %q from genkit", cfg.Model)
		}
		return NewOpenAI(cfg.OpenAIOptions, model, deps.Genkit), nil
	}
	manglekit.Register(r, llm.GroqOptions{}, groqFactory)
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

	// Prepare the generation configuration.
	config := &ai.GenerationCommonConfig{
		Temperature:     float64(o.opts.Temperature),
		MaxOutputTokens: o.opts.MaxOutputTokens, // Default from options
	}

	// Override MaxOutputTokens if provided in the request.
	if req.MaxTokens > 0 {
		config.MaxOutputTokens = req.MaxTokens
	}

	// Use the standard genkit.Generate function.
	res, err := genkit.Generate(ctx, o.genkit,
		ai.WithModel(o.model),
		ai.WithPrompt(req.Prompt),
		ai.WithConfig(config),
	)
	if err != nil {
		return llm.Response{}, err
	}

	// Extract token usage from the response metadata.
	var usage *llm.TokenUsage
	if res.Usage != nil {
		usage = &llm.TokenUsage{
			Provider:   o.opts.ProviderName(),
			Prompt:     res.Usage.InputTokens,
			Completion: res.Usage.OutputTokens,
			Total:      res.Usage.TotalTokens,
		}
	}

	return llm.Response{
		Text:  res.Text(),
		Usage: usage,
	}, nil
}
