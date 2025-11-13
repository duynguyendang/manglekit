package llm

import (
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenkitRegister_Success(t *testing.T) {
	r := manglekit.NewRegistry()
	err := RegisterGenkit(r)
	require.NoError(t, err, "RegisterGenkit should not fail")
}

func TestGenkitLLMOptions_ProviderName(t *testing.T) {
	opts := &GenkitLLMOptions{
		Provider: "openai",
		Model:    "gpt-4-turbo",
	}

	assert.Equal(t, "genkit-llm", opts.ProviderName())
}

func TestGenkitLLMOptions_ProviderKind(t *testing.T) {
	opts := &GenkitLLMOptions{
		Provider: "openai",
		Model:    "gpt-4-turbo",
	}

	assert.Equal(t, core.KindLLM, opts.ProviderKind())
}

func TestGenkitLLMOptions_Fields(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		apiKey   string
		baseURL  string
	}{
		{
			name:     "OpenAI",
			provider: "openai",
			model:    "gpt-4-turbo",
			apiKey:   "sk-test",
			baseURL:  "https://api.openai.com",
		},
		{
			name:     "Groq",
			provider: "groq",
			model:    "mixtral-8x7b-32768",
			apiKey:   "gsk-test",
			baseURL:  "https://api.groq.com/openai/v1",
		},
		{
			name:     "Google",
			provider: "google",
			model:    "gemini-1.5-pro",
			apiKey:   "test-key",
			baseURL:  "",
		},
		{
			name:     "Vertex",
			provider: "vertex",
			model:    "gemini-pro",
			apiKey:   "test-key",
			baseURL:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &GenkitLLMOptions{
				Provider: tt.provider,
				Model:    tt.model,
				APIKey:   tt.apiKey,
				BaseURL:  tt.baseURL,
			}

			assert.Equal(t, tt.provider, opts.Provider)
			assert.Equal(t, tt.model, opts.Model)
			assert.Equal(t, tt.apiKey, opts.APIKey)
			assert.Equal(t, tt.baseURL, opts.BaseURL)
		})
	}
}

func TestGenkitLLMOptions_GetAPIKey(t *testing.T) {
	opts := &GenkitLLMOptions{
		Provider: "openai",
		Model:    "gpt-4-turbo",
		APIKey:   "sk-test-key",
	}

	assert.Equal(t, "sk-test-key", opts.GetAPIKey())
}

func TestGenkitLLMOptions_GetBaseURL(t *testing.T) {
	opts := &GenkitLLMOptions{
		Provider: "groq",
		Model:    "mixtral-8x7b-32768",
		BaseURL:  "https://api.groq.com/openai/v1",
	}

	assert.Equal(t, "https://api.groq.com/openai/v1", opts.GetBaseURL())
}

func TestGenkitLLMOptions_ShouldSkipModelCheck(t *testing.T) {
	opts := &GenkitLLMOptions{
		Provider:       "openai",
		Model:          "gpt-4-turbo",
		SkipModelCheck: true,
	}

	assert.Equal(t, true, opts.ShouldSkipModelCheck())

	opts.SkipModelCheck = false
	assert.Equal(t, false, opts.ShouldSkipModelCheck())
}

func TestGenkitLLMOptions_ProviderConfig_Map(t *testing.T) {
	opts := &GenkitLLMOptions{
		Provider: "openai",
		Model:    "gpt-4-turbo",
		ProviderConfig: map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  2048,
		},
	}

	assert.Equal(t, 0.7, opts.ProviderConfig["temperature"])
	assert.Equal(t, 2048, opts.ProviderConfig["max_tokens"])
}

func TestGenkitLLMOptions_AllFields(t *testing.T) {
	opts := &GenkitLLMOptions{
		Provider:        "openai",
		Model:           "gpt-4-turbo",
		APIKey:          "sk-test",
		BaseURL:         "https://api.openai.com",
		Temperature:     0.5,
		MaxOutputTokens: 1024,
		PromptTemplate:  "custom template",
		SkipModelCheck:  true,
		ProviderConfig: map[string]interface{}{
			"custom": "value",
		},
	}

	assert.Equal(t, "openai", opts.Provider)
	assert.Equal(t, "gpt-4-turbo", opts.Model)
	assert.Equal(t, "sk-test", opts.APIKey)
	assert.Equal(t, "https://api.openai.com", opts.BaseURL)
	assert.Equal(t, float32(0.5), opts.Temperature)
	assert.Equal(t, 1024, opts.MaxOutputTokens)
	assert.Equal(t, "custom template", opts.PromptTemplate)
	assert.Equal(t, true, opts.SkipModelCheck)
	assert.Equal(t, "value", opts.ProviderConfig["custom"])
}
