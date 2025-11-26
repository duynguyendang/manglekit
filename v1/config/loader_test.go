package config

import (
	"os"
	"testing"
)

func TestParseConfigWithEnvVar(t *testing.T) {
	// Set an environment variable for the test.
	testKey := "TEST_API_KEY"
	testValue := "test-api-key-value"
	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Create a YAML config with the environment variable placeholder.
	yamlConfig := `
components:
  - name: "test-component"
    type: "test-type"
    params:
      apiKey: "${TEST_API_KEY}"
`

	// Parse the config.
	cfg, err := ParseConfig([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify that the environment variable was expanded.
	if len(cfg.Components) != 1 {
		t.Fatalf("Expected 1 component, got %d", len(cfg.Components))
	}

	component := cfg.Components[0]
	apiKey, ok := component.Params["apiKey"].(string)
	if !ok {
		t.Fatalf("apiKey is not a string")
	}

	if apiKey != testValue {
		t.Errorf("Expected apiKey to be '%s', but got '%s'", testValue, apiKey)
	}
}
