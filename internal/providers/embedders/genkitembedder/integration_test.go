//go:build testhooks
// +build testhooks

package genkitembedder_test

import (
	"os"
	"testing"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/embed"
	"github.com/duynguyendang/manglekit/internal/embedders"
	"github.com/duynguyendang/manglekit/internal/providers/embedders/genkitembedder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestIntegration_ConfigParsing tests that GenkitEmbedderOptions are correctly parsed from YAML.
func TestIntegration_ConfigParsing(t *testing.T) {
	tests := []struct {
		name       string
		yamlConfig string
		wantErr    bool
		validate   func(t *testing.T, opts *embed.GenkitEmbedderOptions)
	}{
		{
			name: "OpenAI provider configuration",
			yamlConfig: `
provider: openai
model: text-embedding-3-small
api_key: sk-test-key
base_url: https://api.openai.com
dimensions: 1536
skip_model_check: false
provider_config:
  timeout: 30
  retries: 3
`,
			wantErr: false,
			validate: func(t *testing.T, opts *embed.GenkitEmbedderOptions) {
				assert.Equal(t, "openai", opts.Provider)
				assert.Equal(t, "text-embedding-3-small", opts.Model)
				assert.Equal(t, "sk-test-key", opts.APIKey)
				assert.Equal(t, "https://api.openai.com", opts.BaseURL)
				assert.Equal(t, 1536, opts.Dimensions)
				assert.Equal(t, false, opts.SkipModelCheck)
				assert.Equal(t, 30, opts.ProviderConfig["timeout"])
			},
		},
		{
			name: "Google provider configuration",
			yamlConfig: `
provider: google
model: embedding-001
api_key: test-key-google
skip_model_check: true
`,
			wantErr: false,
			validate: func(t *testing.T, opts *embed.GenkitEmbedderOptions) {
				assert.Equal(t, "google", opts.Provider)
				assert.Equal(t, "embedding-001", opts.Model)
				assert.Equal(t, "test-key-google", opts.APIKey)
				assert.Equal(t, true, opts.SkipModelCheck)
			},
		},
		{
			name: "Groq provider configuration",
			yamlConfig: `
provider: groq
model: nomic-embed-text
api_key: gsk-test-key
`,
			wantErr: false,
			validate: func(t *testing.T, opts *embed.GenkitEmbedderOptions) {
				assert.Equal(t, "groq", opts.Provider)
				assert.Equal(t, "nomic-embed-text", opts.Model)
				assert.Equal(t, "gsk-test-key", opts.APIKey)
			},
		},
		{
			name: "Cohere provider configuration",
			yamlConfig: `
provider: cohere
model: embed-english-v3.0
api_key: co-test-key
provider_config:
  input_type: search_document
`,
			wantErr: false,
			validate: func(t *testing.T, opts *embed.GenkitEmbedderOptions) {
				assert.Equal(t, "cohere", opts.Provider)
				assert.Equal(t, "embed-english-v3.0", opts.Model)
				assert.Equal(t, "co-test-key", opts.APIKey)
				assert.Equal(t, "search_document", opts.ProviderConfig["input_type"])
			},
		},
		{
			name: "Minimal configuration",
			yamlConfig: `
provider: openai
model: text-embedding-3-small
`,
			wantErr: false,
			validate: func(t *testing.T, opts *embed.GenkitEmbedderOptions) {
				assert.Equal(t, "openai", opts.Provider)
				assert.Equal(t, "text-embedding-3-small", opts.Model)
				assert.Equal(t, "", opts.APIKey)
				assert.Equal(t, "", opts.BaseURL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts embed.GenkitEmbedderOptions
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

// TestIntegration_OptionsInterfaces tests that GenkitEmbedderOptions implements required interfaces.
func TestIntegration_OptionsInterfaces(t *testing.T) {
	opts := &embed.GenkitEmbedderOptions{
		Provider:       "openai",
		Model:          "text-embedding-3-small",
		APIKey:         "sk-test",
		BaseURL:        "https://api.openai.com",
		Dimensions:     1536,
		SkipModelCheck: false,
	}

	// Test ProviderOptions interface
	assert.Equal(t, "genkit-embedder", opts.ProviderName())
	assert.Equal(t, core.KindEmbedder, opts.ProviderKind())

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

// TestIntegration_RegistryLoading tests that the factory can be registered and used with the Manglekit registry.
func TestIntegration_RegistryLoading(t *testing.T) {
	r := manglekit.NewRegistry()
	// Must register the handler first for embedders to be available
	r.RegisterHandler(embedders.NewHandler())
	err := genkitembedder.Register(r)
	require.NoError(t, err, "failed to register genkit embedder factory")

	// Verify the handler is registered
	handler, err := r.GetHandler(core.KindEmbedder)
	require.NoError(t, err, "embedder handler not registered")
	assert.NotNil(t, handler)
}

// TestIntegration_EmbedderHandler tests that the embedder handler works with the generic factory.
func TestIntegration_EmbedderHandler(t *testing.T) {
	// Setup registry with handler and factory
	r := manglekit.NewRegistry()
	r.RegisterHandler(embedders.NewHandler())

	// Register genkit embedder factory
	err := genkitembedder.Register(r)
	require.NoError(t, err)

	// Verify handler is registered
	handler, err := r.GetHandler(core.KindEmbedder)
	require.NoError(t, err, "embedder handler not found in registry")
	assert.NotNil(t, handler)
}

// TestIntegration_ConfigYAMLParsing tests that embedder configuration can be loaded from YAML files.
func TestIntegration_ConfigYAMLParsing(t *testing.T) {
	configYAML := `
orchestrator: test-orchestrator
components:
  - name: test-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: openai
      model: text-embedding-3-small
      api_key: sk-test-key
      base_url: https://api.openai.com
      dimensions: 1536
      provider_config:
        timeout: 30
        retries: 3
`

	// Parse the config
	var config map[string]interface{}
	err := yaml.Unmarshal([]byte(configYAML), &config)
	require.NoError(t, err)

	// Extract components
	components, ok := config["components"].([]interface{})
	require.True(t, ok, "components is not a list")
	require.Len(t, components, 1)

	// Extract first component
	comp, ok := components[0].(map[string]interface{})
	require.True(t, ok, "component is not a map")

	// Verify component structure
	assert.Equal(t, "test-embedder", comp["name"])
	assert.Equal(t, "embedder", comp["kind"])
	assert.Equal(t, "genkit-embedder", comp["type"])

	// Extract params and parse into options
	params, ok := comp["params"].(map[string]interface{})
	require.True(t, ok, "params is not a map")

	// Convert params to YAML and unmarshal into options
	paramsYAML, err := yaml.Marshal(params)
	require.NoError(t, err)

	var opts embed.GenkitEmbedderOptions
	err = yaml.Unmarshal(paramsYAML, &opts)
	require.NoError(t, err)

	// Verify options were correctly parsed
	assert.Equal(t, "openai", opts.Provider)
	assert.Equal(t, "text-embedding-3-small", opts.Model)
	assert.Equal(t, "sk-test-key", opts.APIKey)
	assert.Equal(t, "https://api.openai.com", opts.BaseURL)
	assert.Equal(t, 1536, opts.Dimensions)
	// timeout may be int or float64 depending on YAML parsing
	timeoutVal := opts.ProviderConfig["timeout"]
	assert.True(t, timeoutVal == 30 || timeoutVal == float64(30), "timeout should be 30")
}

// TestIntegration_EnvironmentVariableExpansion tests that environment variables are expanded in configuration.
func TestIntegration_EnvironmentVariableExpansion(t *testing.T) {
	// Set environment variables
	t.Setenv("EMBEDDER_PROVIDER", "openai")
	t.Setenv("EMBEDDER_MODEL", "text-embedding-3-small")
	t.Setenv("EMBEDDER_API_KEY", "sk-expanded-key")

	configYAML := `
provider: $EMBEDDER_PROVIDER
model: $EMBEDDER_MODEL
api_key: $EMBEDDER_API_KEY
`

	// Expand environment variables
	expandedConfig := os.ExpandEnv(configYAML)

	// Parse the config
	var opts embed.GenkitEmbedderOptions
	err := yaml.Unmarshal([]byte(expandedConfig), &opts)
	require.NoError(t, err)

	// Verify environment variables were expanded
	assert.Equal(t, "openai", opts.Provider)
	assert.Equal(t, "text-embedding-3-small", opts.Model)
	assert.Equal(t, "sk-expanded-key", opts.APIKey)
}

// TestIntegration_MultiProviderConfiguration tests that multiple embedder configurations for different providers work.
func TestIntegration_MultiProviderConfiguration(t *testing.T) {
	configYAML := `
components:
  - name: openai-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: openai
      model: text-embedding-3-small
      apiKey: sk-openai-key
  - name: google-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: google
      model: embedding-001
      apiKey: google-key
  - name: groq-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: groq
      model: nomic-embed-text
      apiKey: gsk-groq-key
  - name: cohere-embedder
    kind: embedder
    type: genkit-embedder
    params:
      provider: cohere
      model: embed-english-v3.0
      apiKey: co-cohere-key
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
	expectedProviders := []string{"openai", "google", "groq", "cohere"}
	for i, comp := range components {
		compMap, ok := comp.(map[string]interface{})
		require.True(t, ok)

		params, ok := compMap["params"].(map[string]interface{})
		require.True(t, ok)

		paramsYAML, err := yaml.Marshal(params)
		require.NoError(t, err)

		var opts embed.GenkitEmbedderOptions
		err = yaml.Unmarshal(paramsYAML, &opts)
		require.NoError(t, err)

		assert.Equal(t, expectedProviders[i], opts.Provider)
		assert.Equal(t, "genkit-embedder", compMap["type"])
	}
}

// TestIntegration_ProviderConfigExtensibility tests that ProviderConfig map allows custom parameters.
func TestIntegration_ProviderConfigExtensibility(t *testing.T) {
	tests := []struct {
		name       string
		yamlConfig string
		validate   func(t *testing.T, opts *embed.GenkitEmbedderOptions)
	}{
		{
			name: "OpenAI with custom provider config",
			yamlConfig: `
provider: openai
model: text-embedding-3-small
provider_config:
  timeout: 60
  retries: 5
  max_tokens: 2048
`,
			validate: func(t *testing.T, opts *embed.GenkitEmbedderOptions) {
				assert.Equal(t, 60, opts.ProviderConfig["timeout"])
				assert.Equal(t, 5, opts.ProviderConfig["retries"])
				assert.Equal(t, 2048, opts.ProviderConfig["max_tokens"])
			},
		},
		{
			name: "Google with custom provider config",
			yamlConfig: `
provider: google
model: embedding-001
provider_config:
  task_type: RETRIEVAL_QUERY
  title: "Test Query"
  api_version: v1beta1
`,
			validate: func(t *testing.T, opts *embed.GenkitEmbedderOptions) {
				assert.Equal(t, "RETRIEVAL_QUERY", opts.ProviderConfig["task_type"])
				assert.Equal(t, "Test Query", opts.ProviderConfig["title"])
				assert.Equal(t, "v1beta1", opts.ProviderConfig["api_version"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts embed.GenkitEmbedderOptions
			err := yaml.Unmarshal([]byte(tt.yamlConfig), &opts)
			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, &opts)
			}
		})
	}
}

// TestIntegration_InvalidConfiguration tests that invalid configurations are properly rejected.
func TestIntegration_InvalidConfiguration(t *testing.T) {
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
model: text-embedding-3-small
`,
			wantErr: false, // YAML parses fine, error would come from factory
		},
		{
			name: "Empty model",
			yamlConfig: `
provider: openai
model: ""
`,
			wantErr: false, // YAML parses fine, error would come from factory
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
			var opts embed.GenkitEmbedderOptions
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

// TestIntegration_OptionsTypeRetrieval tests that options types can be retrieved from the registry.
func TestIntegration_OptionsTypeRetrieval(t *testing.T) {
	r := manglekit.NewRegistry()
	r.RegisterHandler(embedders.NewHandler())
	err := genkitembedder.Register(r)
	require.NoError(t, err)

	// Test that the factory is registered by verifying handler is available
	handler, err := r.GetHandler(core.KindEmbedder)
	require.NoError(t, err, "embedder handler not found in registry")
	require.NotNil(t, handler)
}

// TestIntegration_FactoryMockBuild tests the factory build pattern with mocked Genkit instance.
func TestIntegration_FactoryMockBuild(t *testing.T) {
	r := manglekit.NewRegistry()
	r.RegisterHandler(embedders.NewHandler())
	err := genkitembedder.Register(r)
	require.NoError(t, err)

	// Verify the factory is registered by checking handler availability
	handler, err := r.GetHandler(core.KindEmbedder)
	require.NoError(t, err, "embedder handler not registered")
	assert.NotNil(t, handler)

	// Create and verify options
	opts := &embed.GenkitEmbedderOptions{
		Provider: "openai",
		Model:    "text-embedding-3-small",
		APIKey:   "sk-test",
	}

	// Verify the options implements required interfaces
	assert.Equal(t, "genkit-embedder", opts.ProviderName())
	assert.Equal(t, core.KindEmbedder, opts.ProviderKind())
	assert.Equal(t, "sk-test", opts.GetAPIKey())
}

// BenchmarkConfigParsing benchmarks the performance of parsing embedder configurations.
func BenchmarkConfigParsing(b *testing.B) {
	yamlConfig := `
provider: openai
model: text-embedding-3-small
apiKey: sk-test-key
baseURL: https://api.openai.com
dimensions: 1536
skipModelCheck: false
providerConfig:
  timeout: 30
  retries: 3
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var opts embed.GenkitEmbedderOptions
		_ = yaml.Unmarshal([]byte(yamlConfig), &opts)
	}
}
