//go:build testhooks
// +build testhooks

package llm_test

import (
	"os"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestIntegration_LLMConfigParsing tests that GenkitLLMOptions are correctly parsed from YAML.
func TestIntegration_LLMConfigParsing(t *testing.T) {
	tests := []struct {
		name       string
		yamlConfig string
		wantErr    bool
		validate   func(t *testing.T, opts *llm.GenkitLLMOptions)
	}{
		{
			name: "OpenAI LLM configuration",
			yamlConfig: `
provider: openai
model: gpt-4-turbo
api_key: sk-test-key
base_url: https://api.openai.com
temperature: 0.7
max_output_tokens: 2048
skip_model_check: false
provider_config:
  top_p: 0.9
  frequency_penalty: 0.5
`,
			wantErr: false,
			validate: func(t *testing.T, opts *llm.GenkitLLMOptions) {
				assert.Equal(t, "openai", opts.Provider)
				assert.Equal(t, "gpt-4-turbo", opts.Model)
				assert.Equal(t, "sk-test-key", opts.APIKey)
				assert.Equal(t, "https://api.openai.com", opts.BaseURL)
				assert.Equal(t, float32(0.7), opts.Temperature)
				assert.Equal(t, 2048, opts.MaxOutputTokens)
				assert.Equal(t, false, opts.SkipModelCheck)
				assert.Equal(t, 0.9, opts.ProviderConfig["top_p"])
			},
		},
		{
			name: "Groq LLM configuration",
			yamlConfig: `
provider: groq
model: mixtral-8x7b-32768
api_key: gsk-test-key
base_url: https://api.groq.com/openai/v1
temperature: 0.5
max_output_tokens: 1024
`,
			wantErr: false,
			validate: func(t *testing.T, opts *llm.GenkitLLMOptions) {
				assert.Equal(t, "groq", opts.Provider)
				assert.Equal(t, "mixtral-8x7b-32768", opts.Model)
				assert.Equal(t, "gsk-test-key", opts.APIKey)
				assert.Equal(t, "https://api.groq.com/openai/v1", opts.BaseURL)
				assert.Equal(t, float32(0.5), opts.Temperature)
				assert.Equal(t, 1024, opts.MaxOutputTokens)
			},
		},
		{
			name: "Google Gemini LLM configuration",
			yamlConfig: `
provider: google
model: gemini-1.5-pro
api_key: test-key-google
skip_model_check: true
provider_config:
  safety_settings:
    - category: HARM_CATEGORY_HATE_SPEECH
      threshold: BLOCK_MEDIUM_AND_ABOVE
`,
			wantErr: false,
			validate: func(t *testing.T, opts *llm.GenkitLLMOptions) {
				assert.Equal(t, "google", opts.Provider)
				assert.Equal(t, "gemini-1.5-pro", opts.Model)
				assert.Equal(t, "test-key-google", opts.APIKey)
				assert.Equal(t, true, opts.SkipModelCheck)
			},
		},
		{
			name: "Vertex AI LLM configuration",
			yamlConfig: `
provider: vertex
model: gemini-pro
api_key: vertex-key
temperature: 0.3
`,
			wantErr: false,
			validate: func(t *testing.T, opts *llm.GenkitLLMOptions) {
				assert.Equal(t, "vertex", opts.Provider)
				assert.Equal(t, "gemini-pro", opts.Model)
				assert.Equal(t, "vertex-key", opts.APIKey)
				assert.Equal(t, float32(0.3), opts.Temperature)
			},
		},
		{
			name: "Minimal LLM configuration",
			yamlConfig: `
provider: openai
model: gpt-3.5-turbo
`,
			wantErr: false,
			validate: func(t *testing.T, opts *llm.GenkitLLMOptions) {
				assert.Equal(t, "openai", opts.Provider)
				assert.Equal(t, "gpt-3.5-turbo", opts.Model)
				assert.Equal(t, "", opts.APIKey)
				assert.Equal(t, "", opts.BaseURL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts llm.GenkitLLMOptions
			err := yaml.Unmarshal([]byte(tt.yamlConfig), &opts)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, &opts)
			}
		})
	}
}

// TestIntegration_LLMOptionsInterfaces tests that GenkitLLMOptions implements required interfaces.
func TestIntegration_LLMOptionsInterfaces(t *testing.T) {
	opts := &llm.GenkitLLMOptions{
		Provider:       "openai",
		Model:          "gpt-4-turbo",
		APIKey:         "sk-test",
		BaseURL:        "https://api.openai.com",
		Temperature:    0.7,
		SkipModelCheck: false,
	}

	// Test ProviderOptions interface
	assert.Equal(t, "genkit-llm", opts.ProviderName())
	assert.Equal(t, core.KindLLM, opts.ProviderKind())

	// Test APIKeyProvider interface
	assert.Equal(t, "sk-test", opts.GetAPIKey())

	// Test BaseURLProvider interface
	assert.Equal(t, "https://api.openai.com", opts.GetBaseURL())

	// Test SkipModelCheckProvider interface
	opts.SkipModelCheck = true
	assert.Equal(t, true, opts.ShouldSkipModelCheck())

	opts.SkipModelCheck = false
	assert.Equal(t, false, opts.ShouldSkipModelCheck())
}

// TestIntegration_LLMRegistryLoading tests that the factory can be registered.
func TestIntegration_LLMRegistryLoading(t *testing.T) {
	r := manglekit.NewRegistry()
	r.RegisterHandler(NewHandler())
	err := llm.RegisterGenkit(r)
	require.NoError(t, err, "failed to register genkit LLM factory")

	// Verify the handler is registered
	handler, err := r.GetHandler(core.KindLLM)
	require.NoError(t, err, "LLM handler not registered")
	assert.NotNil(t, handler)
}

// TestIntegration_LLMMultiProviderConfiguration tests simultaneous configuration of multiple LLM providers.
func TestIntegration_LLMMultiProviderConfiguration(t *testing.T) {
	configYAML := `
components:
  - name: openai-llm
    kind: llm
    type: genkit-llm
    params:
      provider: openai
      model: gpt-4-turbo
      api_key: sk-openai-key
  - name: groq-llm
    kind: llm
    type: genkit-llm
    params:
      provider: groq
      model: mixtral-8x7b-32768
      api_key: gsk-groq-key
  - name: google-llm
    kind: llm
    type: genkit-llm
    params:
      provider: google
      model: gemini-1.5-pro
      api_key: google-key
  - name: vertex-llm
    kind: llm
    type: genkit-llm
    params:
      provider: vertex
      model: gemini-pro
      api_key: vertex-key
`

	// Parse the config
	var config map[string]interface{}
	err := yaml.Unmarshal([]byte(configYAML), &config)
	require.NoError(t, err)

	// Extract components
	components, ok := config["components"].([]interface{})
	require.True(t, ok)
	require.Len(t, components, 4)

	// Verify each component
	expectedProviders := []string{"openai", "groq", "google", "vertex"}
	for i, comp := range components {
		compMap, ok := comp.(map[string]interface{})
		require.True(t, ok)

		params, ok := compMap["params"].(map[string]interface{})
		require.True(t, ok)

		paramsYAML, err := yaml.Marshal(params)
		require.NoError(t, err)

		var opts llm.GenkitLLMOptions
		err = yaml.Unmarshal(paramsYAML, &opts)
		require.NoError(t, err)

		assert.Equal(t, expectedProviders[i], opts.Provider)
		assert.Equal(t, "genkit-llm", compMap["type"])
	}
}

// TestIntegration_LLMEnvironmentVariableExpansion tests environment variable expansion.
func TestIntegration_LLMEnvironmentVariableExpansion(t *testing.T) {
	// Set environment variables
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("LLM_MODEL", "gpt-4-turbo")
	t.Setenv("LLM_API_KEY", "sk-expanded-key")

	configYAML := `
provider: $LLM_PROVIDER
model: $LLM_MODEL
api_key: $LLM_API_KEY
`

	// Expand environment variables
	expandedConfig := os.ExpandEnv(configYAML)

	// Parse the config
	var opts llm.GenkitLLMOptions
	err := yaml.Unmarshal([]byte(expandedConfig), &opts)
	require.NoError(t, err)

	// Verify environment variables were expanded
	assert.Equal(t, "openai", opts.Provider)
	assert.Equal(t, "gpt-4-turbo", opts.Model)
	assert.Equal(t, "sk-expanded-key", opts.APIKey)
}

// TestIntegration_LLMProviderConfigExtensibility tests custom provider parameters.
func TestIntegration_LLMProviderConfigExtensibility(t *testing.T) {
	tests := []struct {
		name       string
		yamlConfig string
		validate   func(t *testing.T, opts *llm.GenkitLLMOptions)
	}{
		{
			name: "OpenAI with custom provider config",
			yamlConfig: `
provider: openai
model: gpt-4-turbo
provider_config:
  top_p: 0.9
  frequency_penalty: 0.5
  presence_penalty: 0.3
`,
			validate: func(t *testing.T, opts *llm.GenkitLLMOptions) {
				assert.Equal(t, 0.9, opts.ProviderConfig["top_p"])
				assert.Equal(t, 0.5, opts.ProviderConfig["frequency_penalty"])
				assert.Equal(t, 0.3, opts.ProviderConfig["presence_penalty"])
			},
		},
		{
			name: "Google with custom provider config",
			yamlConfig: `
provider: google
model: gemini-1.5-pro
provider_config:
  top_k: 40
  top_p: 0.95
  max_output_tokens: 2048
`,
			validate: func(t *testing.T, opts *llm.GenkitLLMOptions) {
				assert.Equal(t, 40, opts.ProviderConfig["top_k"])
				assert.Equal(t, 0.95, opts.ProviderConfig["top_p"])
				assert.Equal(t, 2048, opts.ProviderConfig["max_output_tokens"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts llm.GenkitLLMOptions
			err := yaml.Unmarshal([]byte(tt.yamlConfig), &opts)
			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, &opts)
			}
		})
	}
}

// TestIntegration_LLMInvalidConfiguration tests error handling.
func TestIntegration_LLMInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		yamlConfig  string
		wantErr     bool
		errContains string
	}{
		{
			name: "Empty provider",
			yamlConfig: `
provider: ""
model: gpt-4-turbo
`,
			wantErr: false,
		},
		{
			name: "Empty model",
			yamlConfig: `
provider: openai
model: ""
`,
			wantErr: false,
		},
		{
			name: "Invalid YAML",
			yamlConfig: `
provider: openai
model: [invalid: yaml: structure
`,
			wantErr:     true,
			errContains: "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts llm.GenkitLLMOptions
			err := yaml.Unmarshal([]byte(tt.yamlConfig), &opts)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIntegration_LLMOptionsTypeRetrieval verifies registry access patterns.
func TestIntegration_LLMOptionsTypeRetrieval(t *testing.T) {
	r := manglekit.NewRegistry()
	r.RegisterHandler(NewHandler())
	err := llm.RegisterGenkit(r)
	require.NoError(t, err)

	// Verify the handler is registered
	handler, err := r.GetHandler(core.KindLLM)
	require.NoError(t, err, "LLM handler not found in registry")
	require.NotNil(t, handler)
}

// NewHandler is a simple test handler for LLMs
func NewHandler() core.ComponentHandler {
	return llm.NewHandler()
}

// BenchmarkLLMConfigParsing benchmarks YAML parsing performance.
func BenchmarkLLMConfigParsing(b *testing.B) {
	yamlConfig := `
provider: openai
model: gpt-4-turbo
api_key: sk-test-key
base_url: https://api.openai.com
temperature: 0.7
max_output_tokens: 2048
skip_model_check: false
provider_config:
  top_p: 0.9
  frequency_penalty: 0.5
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var opts llm.GenkitLLMOptions
		_ = yaml.Unmarshal([]byte(yamlConfig), &opts)
	}
}
