package llm

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/openai/openai-go"
)

func init() {
	manglekit.RegisterLLM("openai", NewOpenAI)
	manglekit.RegisterLLM("groq", NewOpenAI)
}

// openAIClient implements the llm.Client interface for OpenAI and other services
// that use an OpenAI-compatible API, such as Groq.
type openAIClient struct {
	client         *openai.Client
	modelName      string
	promptTemplate string // The user-provided template string.
	promptBuilder  *llm.PromptBuilder
}

// NewOpenAI creates a new OpenAI-compatible llm.Client. It is the constructor
// function registered with the MangleKit registry for the "openai" and "groq"
// LLM providers. It requires a pre-configured OpenAI client, which is handled
// by the builder.
//
// opts provides configuration such as the model name and an optional prompt template.
// client is the initialized OpenAI Go client.
// It returns a configured llm.Client or an error if dependencies are missing.
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
// chat completions API.
// This method satisfies the llm.Client interface.
//
// req is the request containing the prompt, context, and other parameters.
// It returns an llm.Response with the generated text and token usage, or an
// error if prompt building or the API call fails.
func (c *openAIClient) Complete(req llm.Request) (llm.Response, error) {
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

	resp, err := c.client.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
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