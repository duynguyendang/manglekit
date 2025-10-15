package llm

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func init() {
	factory := func(options any, deps manglekit.FactoryDeps) (llm.Client, error) {
		opts, ok := options.(llm.OpenAIOptions)
		if !ok {
			return nil, fmt.Errorf("invalid options type, expected llm.OpenAIOptions, got %T", options)
		}
		client, ok := deps["client"].(*openai.Client)
		if !ok {
			return nil, fmt.Errorf("invalid client type, expected *openai.Client, got %T", client)
		}
		return NewOpenAI(opts, client)
	}
	manglekit.RegisterLLM("openai", factory)
	manglekit.RegisterLLM("groq", factory)
	manglekit.RegisterOptions("openai", (*llm.OpenAIOptions)(nil))
	manglekit.RegisterOptions("groq", (*llm.OpenAIOptions)(nil))
	manglekit.RegisterClientFactory("openai", openAIClientFactory)
	manglekit.RegisterClientFactory("groq", groqClientFactory)
}

// newOpenAICompatibleClient is a helper that creates a client for any OpenAI-compatible API.
func newOpenAICompatibleClient(apiKey, baseURL string) (any, core.ResourceCloser, error) {
	transport := &http.Transport{
		IdleConnTimeout: 30 * time.Second,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   120 * time.Second,
	}

	var client openai.Client
	if baseURL != "" {
		client = openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseURL), option.WithHTTPClient(httpClient))
	} else {
		client = openai.NewClient(option.WithAPIKey(apiKey), option.WithHTTPClient(httpClient))
	}

	closer := func(ctx context.Context) error {
		transport.CloseIdleConnections()
		return nil
	}

	return &client, closer, nil
}

func openAIClientFactory(ctx context.Context, cfg *manglekit.Config) (any, core.ResourceCloser, error) {
	if cfg.Providers.OpenAI == nil {
		return nil, nil, fmt.Errorf("missing providers.openai config for openai client factory")
	}
	openAICfg := cfg.Providers.OpenAI

	apiKey := openAICfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, nil, fmt.Errorf("missing apiKey for provider 'openai': please provide it via config or OPENAI_API_KEY env var")
	}

	// OpenAI doesn't need a BaseURL unless it's a custom deployment, which is not handled here.
	return newOpenAICompatibleClient(apiKey, "")
}

func groqClientFactory(ctx context.Context, cfg *manglekit.Config) (any, core.ResourceCloser, error) {
	if cfg.Providers.Groq == nil {
		return nil, nil, fmt.Errorf("missing providers.groq config for groq client factory")
	}
	groqCfg := cfg.Providers.Groq

	apiKey := groqCfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("GROQ_API_KEY")
	}
	if apiKey == "" {
		return nil, nil, fmt.Errorf("missing apiKey for provider 'groq': please provide it via config or GROQ_API_KEY env var")
	}

	baseURL := groqCfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	return newOpenAICompatibleClient(apiKey, baseURL)
}

// openAIClient implements the llm.Client interface for OpenAI and other services
// that use an OpenAI-compatible API, such as Groq.
type openAIClient struct {
	client         *openai.Client
	modelName      string
	promptTemplate string // The user-provided template string.
	promptBuilder  *llm.PromptBuilder
}

// Close closes the idle connections on the underlying HTTP transport.
func (c *openAIClient) Close(ctx context.Context) error {
	// The underlying openai-go client does not expose a Close() method.
	// We are already closing the idle connections on the transport in the builder.
	return nil
}

// NewOpenAI is the constructor for an `llm.Client` that is compatible with
// the OpenAI API. It is registered for multiple providers (e.g., "openai", "groq")
// as they share the same interface. It relies on a pre-configured `openai.Client`
// which is injected by the MangleKit builder.
//
// opts provides configuration such as the model name (e.g., "gpt-4-turbo") and
// an optional custom prompt template.
// client is the initialized `openai-go` client instance, which should already
// be configured with the correct API key and base URL by the builder.
// It returns a configured `llm.Client` or an error if dependencies are missing.
func NewOpenAI(opts llm.OpenAIOptions, client *openai.Client) (llm.Client, error) {
	if client == nil {
		return nil, fmt.Errorf("openai client is required")
	}
	if opts.Model == "" {
		return nil, fmt.Errorf("model name is required for openai provider")
	}

	return &openAIClient{
		client:         client,
		modelName:      opts.Model,
		promptTemplate: opts.PromptTemplate,
		promptBuilder:  llm.NewPromptBuilder(llm.DefaultRAGTemplate),
	}, nil
}

// Complete generates a response from the configured OpenAI-compatible model.
// It uses the PromptBuilder to construct the final prompt and then calls the
// standard Chat Completions API. This method satisfies the `llm.Client` interface.
//
// ctx is the context for the API call.
// req is the request containing the prompt, context, and other parameters.
// It returns an `llm.Response` with the generated text and token usage, or an
// error if prompt building or the API call fails.
func (c *openAIClient) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	// Prepare the data for the template.
	data := map[string]any{
		"Context": req.Context,
		"Query":   req.Prompt,
	}
	// Merge any dynamic data from the request, overwriting defaults if needed.
	for k, v := range req.Data {
		data[k] = v
	}

	prompt, err := c.promptBuilder.Build(c.promptTemplate, data)
	if err != nil {
		return llm.Response{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: c.modelName,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})

	if err != nil {
		return llm.Response{}, fmt.Errorf("failed to generate response from openai-compatible provider: %w", err)
	}

	if len(resp.Choices) == 0 {
		return llm.Response{}, fmt.Errorf("openai-compatible provider returned no choices")
	}

	usage := make(map[string]int)
	// The Usage struct is not a pointer, so we can access it directly.
	usage["prompt"] = int(resp.Usage.PromptTokens)
	usage["completion"] = int(resp.Usage.CompletionTokens)

	return llm.Response{
		Text:  resp.Choices[0].Message.Content,
		Usage: usage,
	}, nil
}
