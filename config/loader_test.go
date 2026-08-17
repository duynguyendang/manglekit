package config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad_WithValidYAML(t *testing.T) {
	policyFile, err := os.CreateTemp("", "policy-*.dl")
	if err != nil {
		t.Fatalf("Failed to create temp policy file: %v", err)
	}
	defer os.Remove(policyFile.Name())
	policyFile.Close()

	// Create a temporary YAML file
	content := fmt.Sprintf(`
policy:
  path: %s
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
`, policyFile.Name())

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
	if cfg.Policy.Path != policyFile.Name() {
		t.Errorf("Expected policy path %q, got %q", policyFile.Name(), cfg.Policy.Path)
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
	policyFile, err := os.CreateTemp("", "policy-*.dl")
	if err != nil {
		t.Fatalf("Failed to create temp policy file: %v", err)
	}
	defer os.Remove(policyFile.Name())
	policyFile.Close()

	content := `
policy:
  path: ${POLICY_PATH}

observability:
  service_name: ${SERVICE_NAME}
  otlp_endpoint: ${OTLP_ENDPOINT}
`

	// Set environment variables
	os.Setenv("POLICY_PATH", policyFile.Name())
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
	if cfg.Policy.Path != policyFile.Name() {
		t.Errorf("Expected policy path %q, got %q", policyFile.Name(), cfg.Policy.Path)
	}
	if cfg.Observability.ServiceName != "my-gov-service" {
		t.Errorf("Expected service name my-gov-service, got %q", cfg.Observability.ServiceName)
	}
	if cfg.Observability.OTLPEndpoint != "http://otel-collector:4317" {
		t.Errorf("Expected OTLP endpoint http://otel-collector:4317, got %q", cfg.Observability.OTLPEndpoint)
	}
}

func TestLoad_WithDefaults(t *testing.T) {
	policyFile, err := os.CreateTemp("", "policy-*.dl")
	if err != nil {
		t.Fatalf("Failed to create temp policy file: %v", err)
	}
	defer os.Remove(policyFile.Name())
	policyFile.Close()

	content := fmt.Sprintf(`
policy:
  path: %s
`, policyFile.Name())

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
	if cfg.Observability.ServiceName != DefaultServiceName {
		t.Errorf("Expected default service name %q, got %q", DefaultServiceName, cfg.Observability.ServiceName)
	}
	if cfg.Observability.LogLevel != DefaultLogLevel {
		t.Errorf("Expected default log level %q, got %q", DefaultLogLevel, cfg.Observability.LogLevel)
	}
	if cfg.Policy.MaxSteps != DefaultPolicyMaxSteps {
		t.Errorf("Expected default max_steps %d, got %d", DefaultPolicyMaxSteps, cfg.Policy.MaxSteps)
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
	policyFile, err := os.CreateTemp("", "policy-*.dl")
	if err != nil {
		t.Fatalf("Failed to create temp policy file: %v", err)
	}
	defer os.Remove(policyFile.Name())
	policyFile.Close()

	content := fmt.Sprintf(`
policy:
  path: %s

observability:
  enabled: true
  service_name: reader-test
`, policyFile.Name())

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

	if cfg.Policy.Path != policyFile.Name() {
		t.Errorf("Expected policy path %q, got %q", policyFile.Name(), cfg.Policy.Path)
	}
	if cfg.Observability.ServiceName != "reader-test" {
		t.Errorf("Expected service name reader-test, got %q", cfg.Observability.ServiceName)
	}
}

func TestExpandEnvWithDefaults_DefaultSyntax(t *testing.T) {
	os.Unsetenv("POLICY_PATH")
	result, warnings := expandEnvWithDefaults("path: ${POLICY_PATH:-./policies/main.dl}")
	expected := "path: ./policies/main.dl"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings when a default is provided, got %v", warnings)
	}

	os.Setenv("POLICY_PATH", "/custom/path.dl")
	defer os.Unsetenv("POLICY_PATH")
	result, warnings = expandEnvWithDefaults("path: ${POLICY_PATH:-./policies/main.dl}")
	expected = "path: /custom/path.dl"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings when the var is set, got %v", warnings)
	}
}

func TestExpandEnvWithDefaults_Allowlist(t *testing.T) {
	os.Setenv("SECRET_KEY", "super-secret")
	defer os.Unsetenv("SECRET_KEY")

	result, warnings := expandEnvWithDefaults("key: ${SECRET_KEY}")
	if result != "key: " {
		t.Errorf("Expected disallowed var to expand to empty, got %q", result)
	}
	// A non-allowlisted var without a default warns (naming the variable).
	if len(warnings) != 1 || !strings.Contains(warnings[0], "${SECRET_KEY}") {
		t.Errorf("Expected a warning naming ${SECRET_KEY}, got %v", warnings)
	}

	os.Setenv("API_KEY", "allowed-key")
	defer os.Unsetenv("API_KEY")
	result, warnings = expandEnvWithDefaults("key: ${API_KEY}")
	if result != "key: allowed-key" {
		t.Errorf("Expected allowed var to expand, got %q", result)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings for an allowlisted, set var, got %v", warnings)
	}
}

func TestExpandEnvWithDefaults_DisallowedWithDefault(t *testing.T) {
	os.Unsetenv("SECRET_TOKEN")
	result, warnings := expandEnvWithDefaults("token: ${SECRET_TOKEN:-fallback}")
	if result != "token: fallback" {
		t.Errorf("Expected disallowed var to use default, got %q", result)
	}
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings when a default is provided, got %v", warnings)
	}
}

func TestExpandEnvWithDefaults_UnsetAllowlistedVarWarns(t *testing.T) {
	os.Unsetenv("POLICY_PATH")
	os.Unsetenv("POLICY_PAT") // simulates a typo'd var
	defer os.Unsetenv("POLICY_PATH")

	result, warnings := expandEnvWithDefaults("path: ${POLICY_PAT}")
	if result != "path: " {
		t.Errorf("Expected unset var to expand to empty, got %q", result)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "${POLICY_PAT}") {
		t.Errorf("Expected a warning naming ${POLICY_PAT}, got %v", warnings)
	}

	// The same variable referenced twice warns only once.
	_, warnings = expandEnvWithDefaults("a: ${POLICY_PAT}\nb: ${POLICY_PAT}")
	if len(warnings) != 1 {
		t.Errorf("Expected deduplicated warnings, got %v", warnings)
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := &Config{Observability: ObservabilityConfig{LogLevel: "verbose"}}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid log_level, got nil")
	}
}

func TestValidate_NegativeTimeout(t *testing.T) {
	cfg := &Config{Policy: PolicyConfig{EvaluationTimeout: -1}}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for negative evaluation_timeout, got nil")
	}
}

func TestValidate_NegativeMaxSteps(t *testing.T) {
	cfg := &Config{Policy: PolicyConfig{MaxSteps: -5}}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for negative max_steps, got nil")
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Observability: ObservabilityConfig{LogLevel: "debug"},
		Policy:        PolicyConfig{EvaluationTimeout: 30, MaxSteps: 10},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Expected valid config to pass, got: %v", err)
	}
}

func TestValidate_EmptyOptionalFields(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Expected empty config to pass validation, got: %v", err)
	}
}

func TestValidate_MissingPolicyPath(t *testing.T) {
	cfg := &Config{Policy: PolicyConfig{Path: "/nonexistent/policy.dl"}}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy.path")
	assert.Contains(t, err.Error(), "/nonexistent/policy.dl")
}

func TestValidate_ExistingPolicyPath(t *testing.T) {
	policyFile, err := os.CreateTemp("", "policy-*.dl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(policyFile.Name())
	policyFile.Close()

	cfg := &Config{Policy: PolicyConfig{Path: policyFile.Name()}}
	assert.NoError(t, cfg.Validate())
}

func TestValidate_UnknownMemoryProvider(t *testing.T) {
	cfg := &Config{Memory: MemoryConfig{Provider: "pinecone"}}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "memory.provider")
	assert.Contains(t, err.Error(), "pinecone")
}

func TestValidate_KnownMemoryProvider(t *testing.T) {
	for _, p := range KnownMemoryProviders {
		cfg := &Config{Memory: MemoryConfig{Provider: p}}
		assert.NoError(t, cfg.Validate(), "provider %s should validate", p)
	}
}

func TestValidate_InvalidMCPTransport(t *testing.T) {
	cfg := &Config{MCP: []MCPServerConfig{{
		Name:      "bad-server",
		Transport: "websocket",
	}}}
	err := cfg.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "websocket")
	assert.Contains(t, err.Error(), "stdio")
}

func TestValidate_ValidMCPTransport(t *testing.T) {
	for _, tr := range KnownMCPTransports {
		cfg := &Config{MCP: []MCPServerConfig{{Name: "srv", Transport: tr}}}
		assert.NoError(t, cfg.Validate(), "transport %s should validate", tr)
	}
}

func TestParseConfig_WarnsOnUnsetEnvVar(t *testing.T) {
	os.Unsetenv("POLICY_PAT")
	content := []byte("observability:\n  service_name: ${POLICY_PAT}\n")
	cfg, err := ParseConfig(content)
	assert.NoError(t, err)
	// The unset var expands to empty, then the default applies.
	// (The warning itself is printed to stderr; the expansion behavior is
	// unchanged so this stays a warning, not an error.)
	assert.Equal(t, DefaultServiceName, cfg.Observability.ServiceName)
}
