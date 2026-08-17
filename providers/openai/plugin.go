package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	openai "github.com/sashabaranov/go-openai"
)

// Config holds connection parameters.
type Config struct {
	APIKey  string
	BaseURL string // Crucial for Ollama/LocalAI/OpenRouter support
}

// Init registers a specific model (e.g., "gpt-4o") into Genkit's global registry.
// It maps "openai/{modelID}" in Genkit to "{modelID}" in OpenAI API.
func Init(g *genkit.Genkit, modelID string, cfg Config) error {
	// 1. Env Var Fallback
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	// Allow empty Key only if BaseURL is set (e.g. Local Ollama)
	if apiKey == "" && cfg.BaseURL == "" {
		return fmt.Errorf("openai provider: API Key is missing and no BaseURL provided")
	}

	// 2. Setup Client
	c := openai.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		c.BaseURL = cfg.BaseURL
	}
	client := openai.NewClientWithConfig(c)

	// 3. Define Metadata
	meta := &ai.ModelOptions{
		Label: modelID,
		Supports: &ai.ModelSupports{
			Multiturn: true, SystemRole: true, Tools: false, Media: false,
		},
	}

	// 4. Register
	// Genkit Name: "openai/gpt-4o"
	// We use genkit.DefineModel to register it to the instance.
	// Name: "openai/" + modelID
	fullName := "openai/" + modelID
	genkit.DefineModel(g, fullName, meta, func(ctx context.Context, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
		// Pass the explicit 'modelID' to ensure we don't send "openai/gpt-4o" to the API
		return generate(ctx, client, modelID, req, cb)
	})

	return nil
}

// generate handles the translation logic. When cb is non-nil the response is
// streamed incrementally through cb and the assembled full response is
// returned once the stream completes (Genkit ai.ModelFunc contract).
func generate(ctx context.Context, client *openai.Client, explicitModelID string, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	oaiReq := buildChatCompletionRequest(explicitModelID, req)

	if cb != nil {
		return generateStream(ctx, client, oaiReq, cb)
	}

	resp, err := client.CreateChatCompletion(ctx, oaiReq)
	if err != nil {
		return nil, fmt.Errorf("openai completion error: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	// 5. Return Response
	return &ai.ModelResponse{
		Message: ai.NewModelTextMessage(resp.Choices[0].Message.Content),
		// Fill Usage if available
		Usage: &ai.GenerationUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}, nil
}

// buildChatCompletionRequest maps a Genkit ModelRequest to an OpenAI
// ChatCompletionRequest.
func buildChatCompletionRequest(explicitModelID string, req *ai.ModelRequest) openai.ChatCompletionRequest {
	// 1. Map Messages
	msgs := []openai.ChatCompletionMessage{}
	for _, m := range req.Messages {
		role := openai.ChatMessageRoleUser
		switch m.Role {
		case ai.RoleModel:
			role = openai.ChatMessageRoleAssistant
		case ai.RoleSystem:
			role = openai.ChatMessageRoleSystem
		}

		content := ""
		for _, p := range m.Content {
			if p.Text != "" {
				content += p.Text
			}
		}
		msgs = append(msgs, openai.ChatCompletionMessage{Role: role, Content: content})
	}

	// 2. Prepare Request
	// IMPORTANT: Use explicitModelID
	oaiReq := openai.ChatCompletionRequest{
		Model:    explicitModelID,
		Messages: msgs,
	}

	// 3. Map Config
	// Config is 'any' in ModelRequest. We need to cast it.
	// Genkit passes 'GenerationCommonConfig' usually.
	// Fields are values (not pointers) in this version.
	if cfg, ok := req.Config.(ai.GenerationCommonConfig); ok {
		if cfg.Temperature != 0 {
			oaiReq.Temperature = float32(cfg.Temperature)
		}
		if cfg.MaxOutputTokens != 0 {
			oaiReq.MaxTokens = cfg.MaxOutputTokens
		}
		if cfg.TopP != 0 {
			oaiReq.TopP = float32(cfg.TopP)
		}
	}

	return oaiReq
}

// generateStream executes the request against OpenAI's streaming API. Each
// delta is forwarded to the Genkit streaming callback as an incremental
// ModelResponseChunk; the assembled full response (with token usage, when the
// server reports it) is returned after the stream ends.
func generateStream(ctx context.Context, client *openai.Client, oaiReq openai.ChatCompletionRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	// Request token usage on the final streamed chunk.
	oaiReq.StreamOptions = &openai.StreamOptions{IncludeUsage: true}

	stream, err := client.CreateChatCompletionStream(ctx, oaiReq)
	if err != nil {
		return nil, fmt.Errorf("openai stream request error: %w", err)
	}
	defer stream.Close()

	var aggregated strings.Builder
	var usage openai.Usage

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("openai stream error: %w", err)
		}
		if len(resp.Choices) > 0 {
			if delta := resp.Choices[0].Delta.Content; delta != "" {
				aggregated.WriteString(delta)
				chunk := &ai.ModelResponseChunk{
					Role:    ai.RoleModel,
					Content: []*ai.Part{ai.NewTextPart(delta)},
				}
				if err := cb(ctx, chunk); err != nil {
					return nil, fmt.Errorf("openai stream callback error: %w", err)
				}
			}
		}
		if resp.Usage != nil {
			usage = *resp.Usage
		}
	}

	return &ai.ModelResponse{
		Message: ai.NewModelTextMessage(aggregated.String()),
		Usage: &ai.GenerationUsage{
			InputTokens:  usage.PromptTokens,
			OutputTokens: usage.CompletionTokens,
			TotalTokens:  usage.TotalTokens,
		},
	}, nil
}
