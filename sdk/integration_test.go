package sdk

import (
	"context"
	"os"
	"testing"
)

func TestNewClientFromConfig(t *testing.T) {
	// Create a temporary config file that does not require loading a policy
	// (policy.path is empty, so no policy file loading is attempted)
	configContent := `
observability:
  enabled: true
  service_name: test-service
  log_level: debug
`

	configFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write to config file: %v", err)
	}
	configFile.Close()

	// Test NewClientFromConfig
	ctx := context.Background()
	client, err := NewClientFromConfig(ctx, configFile.Name())
	if err != nil {
		t.Fatalf("Failed to create client from config: %v", err)
	}

	// Verify client was initialized
	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.engine == nil {
		t.Error("Expected non-nil engine")
	}

	if client.logger == nil {
		t.Error("Expected non-nil logger")
	}

	if client.tracer == nil {
		t.Error("Expected non-nil tracer")
	}
}

func TestNewClientFromConfig_WithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("SERVICE_NAME", "env-test-service")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("LOG_LEVEL")
	}()

	// Create a temporary config file with environment variable references
	configContent := `
observability:
  enabled: true
  service_name: ${SERVICE_NAME}
  log_level: ${LOG_LEVEL}
`

	configFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write to config file: %v", err)
	}
	configFile.Close()

	// Test NewClientFromConfig with environment variable expansion
	ctx := context.Background()
	client, err := NewClientFromConfig(ctx, configFile.Name())
	if err != nil {
		t.Fatalf("Failed to create client from config: %v", err)
	}

	// Verify client was initialized
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
}

func TestNewClientFromConfig_FileNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := NewClientFromConfig(ctx, "/nonexistent/path/to/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file, got nil")
	}
}

func TestNewClientFromConfig_InvalidPolicyPath(t *testing.T) {
	// Create a temporary config file with invalid policy path
	configContent := `
policy:
  path: /nonexistent/path/to/policy.dlog

observability:
  enabled: true
  service_name: test-service
`

	configFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write to config file: %v", err)
	}
	configFile.Close()

	// Test NewClientFromConfig with invalid policy path
	ctx := context.Background()
	_, err = NewClientFromConfig(ctx, configFile.Name())
	if err == nil {
		t.Error("Expected error for invalid policy path, got nil")
	}
}
