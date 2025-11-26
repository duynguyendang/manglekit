package config

import (
	"os"
	"testing"
)

func TestLoad_WithValidYAML(t *testing.T) {
	// Create a temporary YAML file
	content := `
policy:
  path: /path/to/policy.dl
  evaluation_timeout: 30

observability:
  enabled: true
  service_name: test-service
  log_level: debug
  otlp_endpoint: http://localhost:4317

actions:
  llm_google:
    type: llm
    provider: google
    options:
      model: gemini-pro
  retriever_vector:
    type: retriever
    provider: vector
    options:
      top_k: 5
`

	tmpFile, err := os.CreateTemp("", "mangle-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Load the configuration
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify policy config
	if cfg.Policy.Path != "/path/to/policy.dl" {
		t.Errorf("Expected policy path /path/to/policy.dl, got %q", cfg.Policy.Path)
	}
	if cfg.Policy.EvaluationTimeout != 30 {
		t.Errorf("Expected evaluation timeout 30, got %d", cfg.Policy.EvaluationTimeout)
	}

	// Verify observability config
	if !cfg.Observability.Enabled {
		t.Error("Expected observability to be enabled")
	}
	if cfg.Observability.ServiceName != "test-service" {
		t.Errorf("Expected service name test-service, got %q", cfg.Observability.ServiceName)
	}
	if cfg.Observability.LogLevel != "debug" {
		t.Errorf("Expected log level debug, got %q", cfg.Observability.LogLevel)
	}
	if cfg.Observability.OTLPEndpoint != "http://localhost:4317" {
		t.Errorf("Expected OTLP endpoint http://localhost:4317, got %q", cfg.Observability.OTLPEndpoint)
	}

	// Verify actions
	if len(cfg.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(cfg.Actions))
	}

	if llm, ok := cfg.Actions["llm_google"]; ok {
		if llm.Type != "llm" {
			t.Errorf("Expected LLM type llm, got %q", llm.Type)
		}
		if llm.Provider != "google" {
			t.Errorf("Expected LLM provider google, got %q", llm.Provider)
		}
		if model, ok := llm.Options["model"]; ok {
			if model != "gemini-pro" {
				t.Errorf("Expected model gemini-pro, got %q", model)
			}
		} else {
			t.Error("Expected model option in LLM config")
		}
	} else {
		t.Error("Expected llm_google action in config")
	}
}

func TestLoad_WithEnvironmentVariables(t *testing.T) {
	content := `
policy:
  path: ${POLICY_PATH}

observability:
  service_name: ${SERVICE_NAME}
  otlp_endpoint: ${OTLP_ENDPOINT}
`

	// Set environment variables
	os.Setenv("POLICY_PATH", "/opt/policies/main.dl")
	os.Setenv("SERVICE_NAME", "my-gov-service")
	os.Setenv("OTLP_ENDPOINT", "http://otel-collector:4317")
	defer func() {
		os.Unsetenv("POLICY_PATH")
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("OTLP_ENDPOINT")
	}()

	tmpFile, err := os.CreateTemp("", "mangle-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment variable expansion
	if cfg.Policy.Path != "/opt/policies/main.dl" {
		t.Errorf("Expected policy path /opt/policies/main.dl, got %q", cfg.Policy.Path)
	}
	if cfg.Observability.ServiceName != "my-gov-service" {
		t.Errorf("Expected service name my-gov-service, got %q", cfg.Observability.ServiceName)
	}
	if cfg.Observability.OTLPEndpoint != "http://otel-collector:4317" {
		t.Errorf("Expected OTLP endpoint http://otel-collector:4317, got %q", cfg.Observability.OTLPEndpoint)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	content := `
policy:
  path: /path/to/policy.dl
`

	tmpFile, err := os.CreateTemp("", "mangle-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify defaults were applied
	if cfg.Observability.ServiceName != "manglekit-app" {
		t.Errorf("Expected default service name manglekit-app, got %q", cfg.Observability.ServiceName)
	}
	if cfg.Observability.LogLevel != "info" {
		t.Errorf("Expected default log level info, got %q", cfg.Observability.LogLevel)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/to/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestParseConfig_InvalidYAML(t *testing.T) {
	invalidYAML := []byte(`
invalid:
  - yaml
    structure:
  broken
`)

	_, err := ParseConfig(invalidYAML)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestLoadFromReader(t *testing.T) {
	content := `
policy:
  path: /path/to/policy.dl

observability:
  enabled: true
  service_name: reader-test
`

	tmpFile, err := os.CreateTemp("", "mangle-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek to beginning: %v", err)
	}

	cfg, err := LoadFromReader(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config from reader: %v", err)
	}
	tmpFile.Close()

	if cfg.Policy.Path != "/path/to/policy.dl" {
		t.Errorf("Expected policy path /path/to/policy.dl, got %q", cfg.Policy.Path)
	}
	if cfg.Observability.ServiceName != "reader-test" {
		t.Errorf("Expected service name reader-test, got %q", cfg.Observability.ServiceName)
	}
}
