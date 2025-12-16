package openai

import (
	"context"
	"fmt"
	"os"

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

// generate handles the translation logic
func generate(ctx context.Context, client *openai.Client, explicitModelID string, req *ai.ModelRequest, cb func(context.Context, *ai.ModelResponseChunk) error) (*ai.ModelResponse, error) {
	// CONSTRAINT: No Streaming Support yet
	if cb != nil {
		return nil, fmt.Errorf("streaming is not supported by this provider yet")
	}

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

	// 4. Execute
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
