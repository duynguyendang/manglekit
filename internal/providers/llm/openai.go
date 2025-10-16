package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func RegisterOpenAI(r *manglekit.Registry) {
	factory := func(ctx context.Context, options any, deps manglekit.FactoryDeps) (llm.Client, error) {
		opts, ok := options.(*llm.OpenAIOptions)
		if !ok {
			return nil, fmt.Errorf("invalid options type, expected *llm.OpenAIOptions, got %T", options)
		}
		client, ok := deps["client"].(*openai.Client)
		if !ok {
			return nil, fmt.Errorf("missing required dependency 'client' of type *openai.Client")
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

func openAIClientFactory(ctx context.Context, cfg *manglekit.Config) (any, core.ResourceCloser, error) {
	if cfg.Providers.OpenAI == nil {
		return nil, nil, fmt.Errorf("openai provider config not found")
	}
	apiKey := cfg.Providers.OpenAI.APIKey
	if apiKey == "" {
		return nil, nil, fmt.Errorf("openai provider requires 'apiKey'")
	}
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return client, nil, nil
}

func groqClientFactory(ctx context.Context, cfg *manglekit.Config) (any, core.ResourceCloser, error) {
	if cfg.Providers.Groq == nil {
		return nil, nil, fmt.Errorf("groq provider config not found")
	}
	apiKey := cfg.Providers.Groq.APIKey
	if apiKey == "" {
		return nil, nil, fmt.Errorf("groq provider requires 'apiKey'")
	}
	baseURL := cfg.Providers.Groq.BaseURL
	if baseURL == "" {
		return nil, nil, fmt.Errorf("groq provider requires 'baseURL'")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL))

	closer := func(ctx context.Context) error {
		// The new client doesn't expose the http.Client, so we can't close idle connections.
		return nil
	}

	return client, closer, nil
}