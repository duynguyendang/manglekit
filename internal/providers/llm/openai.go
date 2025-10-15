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
	manglekit.RegisterLLM("openai", NewOpenAI)
	manglekit.RegisterLLM("groq", NewOpenAI)
	manglekit.RegisterOptions("openai", (*llm.OpenAIOptions)(nil))
	manglekit.RegisterOptions("groq", (*llm.OpenAIOptions)(nil))
	manglekit.RegisterClientFactory("openai", openAIClientFactory)
	manglekit.RegisterClientFactory("groq", openAIClientFactory)
}

func openAIClientFactory(options any) (any, core.ResourceCloser, error) {
	var apiKey, baseURL string
	var apiKeyEnvVar string

	switch opts := options.(type) {
	case *manglekit.OpenAIConfig:
		apiKey = opts.APIKey
		apiKeyEnvVar = "OPENAI_API_KEY"
	case *manglekit.OpenAICompatibleConfig:
		apiKey = opts.APIKey
		baseURL = opts.BaseURL
		apiKeyEnvVar = "GROQ_API_KEY" // As an example for groq
	default:
		return nil, nil, fmt.Errorf("unsupported options type for openai client factory: %T", options)
	}

	if apiKey == "" {
		apiKey = os.Getenv(apiKeyEnvVar)
	}
	if apiKey == "" {
		return nil, nil, fmt.Errorf("missing apiKey for provider: please provide it via config or %s env var", apiKeyEnvVar)
	}

	if baseURL == "" {
		if _, ok := options.(*manglekit.OpenAICompatibleConfig); ok {
			// Default for groq
			baseURL = "https://api.groq.com/openai/v1"
		}
	}

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
