package openai

import (
	"context"
	"io"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/sashabaranov/go-openai"
)

// LLM is an OpenAI-compatible LLM implementation.
// It wraps the Genkit plugin to provide core.TextGenerator interface.
type LLM struct {
	client *openai.Client
	model  string
}

// NewLLM creates a new OpenAI LLM client.
// If baseURL is empty, it defaults to OpenAI's API.
// If modelName is empty, it defaults to "gpt-3.5-turbo".
func NewLLM(apiKey, baseURL, modelName string) (*LLM, error) {
	if modelName == "" {
		modelName = "gpt-3.5-turbo"
	}
	config := openai.DefaultConfig(apiKey)
	if baseURL != "" {
		config.BaseURL = baseURL
	}
	return &LLM{
		client: openai.NewClientWithConfig(config),
		model:  modelName,
	}, nil
}

// Complete generates text from a prompt.
func (l *LLM) Complete(ctx context.Context, prompt string) (string, error) {
	resp, err := l.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: l.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", nil
	}
	return resp.Choices[0].Message.Content, nil
}

// Generate generates text with options.
func (l *LLM) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	cfg := &core.GenerationConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	req := openai.ChatCompletionRequest{
		Model: l.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Apply options
	if cfg.Temperature > 0 {
		req.Temperature = float32(cfg.Temperature)
	}
	if cfg.MaxTokens > 0 {
		req.MaxTokens = cfg.MaxTokens
	}

	resp, err := l.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	text := ""
	if len(resp.Choices) > 0 {
		text = resp.Choices[0].Message.Content
	}

	usage := make(map[string]int)
	if resp.Usage.PromptTokens > 0 {
		usage["prompt"] = resp.Usage.PromptTokens
	}
	if resp.Usage.CompletionTokens > 0 {
		usage["completion"] = resp.Usage.CompletionTokens
	}
	if resp.Usage.TotalTokens > 0 {
		usage["total"] = resp.Usage.TotalTokens
	}

	return &core.LLMResponse{
		Text:  text,
		Usage: usage,
	}, nil
}

// Stream generates a stream of text.
func (l *LLM) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	stream, err := l.client.CreateChatCompletionStream(
		ctx,
		openai.ChatCompletionRequest{
			Model: l.model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		return nil, err
	}

	ch := make(chan string)
	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			resp, err := stream.Recv()
			if err != nil {
				if err != io.EOF {
					// Error occurred, but we can't send it on the channel
					// The channel only sends strings
				}
				return
			}
			if len(resp.Choices) > 0 {
				ch <- resp.Choices[0].Delta.Content
			}
		}
	}()

	return ch, nil
}
