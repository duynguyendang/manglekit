package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromYAML(t *testing.T) {
	yamlContent := `
llm:
  provider: google
  options:
    model: gemini-pro
topK: 20
`
	reader := strings.NewReader(yamlContent)
	cfg, err := LoadFromYAML(reader)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "google", cfg.LLM.Provider)
	assert.Equal(t, "gemini-pro", cfg.LLM.Options["model"])
	assert.Equal(t, 20, cfg.TopK)
}

func TestLoadFromYAMLWithEnvVar(t *testing.T) {
	os.Setenv("TEST_MODEL_NAME", "test-model-from-env")
	defer os.Unsetenv("TEST_MODEL_NAME")

	yamlContent := `
llm:
  provider: openai
  options:
    model: $TEST_MODEL_NAME
`
	reader := strings.NewReader(yamlContent)
	cfg, err := LoadFromYAML(reader)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "openai", cfg.LLM.Provider)
	assert.Equal(t, "test-model-from-env", cfg.LLM.Options["model"])
}

func TestNormalize(t *testing.T) {
	cfg := &Config{}
	cfg.Normalize()
	assert.Equal(t, 10, cfg.TopK)

	cfg.TopK = 50
	cfg.Normalize()
	assert.Equal(t, 50, cfg.TopK)
}

func TestValidate(t *testing.T) {
	// Test missing LLM
	cfg := &Config{}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm configuration is required")

	// Test missing LLM provider
	cfg.LLM = &LLMConfig{}
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm.provider is required")

	// Test missing embedder
	cfg.LLM.Provider = "google"
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedder configuration is required")

	// Test missing embedder provider
	cfg.Embedder = &EmbedderConfig{}
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "embedder.provider is required")

	// Test missing retriever
	cfg.Embedder.Provider = "google"
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retrieve configuration is required")

	// Test missing retriever provider
	cfg.Retrieve = &RetrieveConfig{}
	err = cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retrieve.provider is required")

	// Test valid config
	cfg.Retrieve.Provider = "bm25"
	err = cfg.Validate()
	assert.NoError(t, err)
}
