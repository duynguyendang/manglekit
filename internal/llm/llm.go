// Package llm provides LLM client implementations for Manglekit.
// It implements the Gateway interface for generating responses using external LLM services.
package llm

import (
	"context"
	"fmt"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"ndduy.dev/manglekit/internal/types"
)

// LLM interface defines the basic LLM operations (legacy interface).
type LLM interface {
	Answer(ctx context.Context, question string, contextDocs []string) (string, error)
}

// OpenAIClient implements both LLM and Gateway interfaces for OpenAI API.
type OpenAIClient struct {
	api *openai.Client
}

// NewLLMFromEnv creates a new OpenAI client using environment variables.
func NewLLMFromEnv() *OpenAIClient {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Don't fail here, let it fail at runtime with a clear error
		apiKey = "missing-api-key"
	}

	return &OpenAIClient{
		api: openai.NewClient(apiKey),
	}
}

// Answer implements the legacy LLM interface for backward compatibility.
func (c *OpenAIClient) Answer(ctx context.Context, question string, contextDocs []string) (string, error) {
	// Convert to chunks for the new interface
	chunks := make([]*types.Chunk, len(contextDocs))
	for i, doc := range contextDocs {
		chunks[i] = &types.Chunk{
			ID:   fmt.Sprintf("doc_%d", i),
			Text: doc,
		}
	}

	// Build a simple prompt
	prompt := fmt.Sprintf("Use only the provided context to answer. If uncertain, say so.\n\nContext:\n%s\n\nQuestion: %s",
		strings.Join(contextDocs, "\n---\n"), question)

	response, err := c.Generate(ctx, prompt, chunks)
	if err != nil {
		return "", err
	}

	return response.Answer, nil
}

// Generate implements the Gateway interface for the new architecture.
func (c *OpenAIClient) Generate(ctx context.Context, prompt string, chunks []*types.Chunk) (*types.Response, error) {
	// Check for API key
	if os.Getenv("OPENAI_API_KEY") == "" || os.Getenv("OPENAI_API_KEY") == "missing-api-key" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	// Create the chat completion request
	resp, err := c.api.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens:   1000,
			Temperature: 0.1, // Low temperature for more deterministic responses
		},
	)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned from OpenAI")
	}

	// Extract citations from chunks
	citations := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.ID != "" {
			citations = append(citations, chunk.ID)
		}
	}

	// Build response
	response := &types.Response{
		Answer:    resp.Choices[0].Message.Content,
		Citations: citations,
		Metadata: map[string]interface{}{
			"model":         resp.Model,
			"usage_tokens":  resp.Usage.TotalTokens,
			"finish_reason": resp.Choices[0].FinishReason,
		},
	}

	return response, nil
}
