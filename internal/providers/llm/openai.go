package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func RegisterOpenAI(r *manglekit.Registry) {
	factory := func(ctx context.Context, deps diapi.LLMDeps, cfg any) (llm.Client, error) {
		opts, ok := cfg.(*llm.OpenAIOptions)
		if !ok {
			return nil, fmt.Errorf("invalid options type, expected *llm.OpenAIOptions, got %T", cfg)
		}
		if deps.Client == nil {
			return nil, fmt.Errorf("missing required dependency 'client'")
		}
		client, ok := deps.Client.(*openai.Client)
		if !ok {
			return nil, fmt.Errorf("dependency 'client' has the wrong type, expected *openai.Client, got %T", deps.Client)
		}
		return NewOpenAI(*opts, client), nil
	}
	r.RegisterLLM("openai", factory)
	r.RegisterLLM("groq", factory)
	if err := r.RegisterOptions("openai", (*llm.OpenAIOptions)(nil)); err != nil {
		panic(err)
	}
	if err := r.RegisterOptions("groq", (*llm.OpenAIOptions)(nil)); err != nil {
		panic(err)
	}
	r.RegisterClientFactory("openai", openAIClientFactory)
	r.RegisterClientFactory("groq", groqClientFactory)
}

// OpenAI is a wrapper around the OpenAI client.
type OpenAI struct {
	opts   llm.OpenAIOptions
	client *openai.Client
}

// NewOpenAI is the constructor for the OpenAI client wrapper.
func NewOpenAI(opts llm.OpenAIOptions, client *openai.Client) llm.Client {
	return &OpenAI{
		opts:   opts,
		client: client,
	}
}

// Complete implements the llm.Client interface.
func (o *OpenAI) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	resp, err := o.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: o.opts.Model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(req.Prompt),
			},
		},
	)

	if err != nil {
		return llm.Response{}, err
	}

	return llm.Response{Text: resp.Choices[0].Message.Content}, nil
}

func openAIClientFactory(ctx context.Context, cfg *config.Config) (any, core.ResourceCloser, error) {
	// This is a placeholder implementation. In the new world, the client factory
	// would receive its own specific options struct, not the entire config.
	// For now, we'll just return a basic client.
	apiKey := "dummy-key" // In a real scenario, this would come from cfg.Clients["openai"].APIKey
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return client, nil, nil
}

func groqClientFactory(ctx context.Context, cfg *config.Config) (any, core.ResourceCloser, error) {
	// This is a placeholder implementation.
	apiKey := "dummy-key"
	baseURL := "https://api.groq.com/openai/v1"
	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL))
	return client, nil, nil
}