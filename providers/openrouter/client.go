package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// options holds configuration for the OpenRouter provider.
type options struct {
	apiKey  string
	model   string
	baseURL string
}

// Option is a functional option for configuration.
type Option func(*options)

// WithAPIKey sets the OpenRouter API Key.
func WithAPIKey(key string) Option {
	return func(o *options) {
		o.apiKey = key
	}
}

// WithModel sets the model name (e.g., "liquid/lfm-40b").
func WithModel(model string) Option {
	return func(o *options) {
		o.model = model
	}
}

// WithBaseURL sets the base URL (default: "https://openrouter.ai/api/v1").
func WithBaseURL(url string) Option {
	return func(o *options) {
		o.baseURL = url
	}
}

// Client implements core.TextGenerator for OpenRouter/OpenAI compatible APIs.
type Client struct {
	opts       options
	httpClient *http.Client
}

// New initializes the OpenRouter client using Functional Options.
func New(ctx context.Context, opts ...Option) (core.TextGenerator, error) {
	o := &options{
		model:   "liquid/lfm-40b",
		baseURL: "https://openrouter.ai/api/v1",
	}

	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		o.apiKey = key
	}

	for _, opt := range opts {
		opt(o)
	}

	if o.apiKey == "" {
		return nil, fmt.Errorf("openrouter requires API Key (OPENROUTER_API_KEY env or WithAPIKey)")
	}

	return &Client{
		opts:       *o,
		httpClient: &http.Client{},
	}, nil
}

// Complete implements core.TextGenerator.
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	resp, err := c.Generate(ctx, prompt)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

type openAIRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []choice  `json:"choices"`
	Usage   usage     `json:"usage"`
	Error   *apiError `json:"error,omitempty"`
}

type choice struct {
	Message message `json:"message"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type apiError struct {
	Message string `json:"message"`
}

// Generate implements core.TextGenerator.
func (c *Client) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	// 1. Prepare Request
	reqBody := openAIRequest{
		Model: c.opts.model,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	// Dynamic Prompt Configuration via Context Facts
	facts := core.ContextFacts(ctx)
	if facts != nil {
		systemPrompt := ""
		if val, ok := facts[core.PrefixPromptConfig+"tone"]; ok {
			systemPrompt += "Tone: " + val + ". "
		}
		if systemPrompt != "" {
			reqBody.Messages = append([]message{{Role: "system", Content: systemPrompt}}, reqBody.Messages...)
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.opts.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.opts.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/duynguyendang/manglekit")
	req.Header.Set("X-Title", "Manglekit")

	// 2. Execute
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(body))
	}

	// 3. Parse Response
	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Error != nil {
		return nil, fmt.Errorf("api error: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	return &core.LLMResponse{
		Text: result.Choices[0].Message.Content,
		Usage: map[string]int{
			"prompt":     result.Usage.PromptTokens,
			"completion": result.Usage.CompletionTokens,
			"total":      result.Usage.TotalTokens,
		},
	}, nil
}

// Stream implements core.TextGenerator.
func (c *Client) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	return nil, fmt.Errorf("streaming not implemented for openrouter yet")
}

// Register installs the OpenRouter provider into the Manglekit SDK registry.
func Register() {
	sdk.RegisterProvider("openrouter", func(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error) {
		var opts []Option

		if m, ok := cfg.Options["model"].(string); ok {
			opts = append(opts, WithModel(m))
		}
		if k, ok := cfg.Options["api_key"].(string); ok {
			opts = append(opts, WithAPIKey(k))
		}
		if u, ok := cfg.Options["base_url"].(string); ok {
			opts = append(opts, WithBaseURL(u))
		}

		gen, err := New(ctx, opts...)
		if err != nil {
			return nil, err
		}
		return ai.NewLLMAction(name, gen)
	})
}
