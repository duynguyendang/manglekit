package core

import (
	"os"
	"strings"
	"testing"
)

func TestProviderDependencyValidation(t *testing.T) {
	registry := NewProviderDependencyRegistry()

	tests := []struct {
		name        string
		provider    string
		setEnvVar   string
		shouldPass  bool
		description string
	}{
		{
			name:        "Google provider with GOOGLE_API_KEY set",
			provider:    "google",
			setEnvVar:   "GOOGLE_API_KEY",
			shouldPass:  true,
			description: "Should pass when required env var is set",
		},
		{
			name:        "Google provider without GOOGLE_API_KEY",
			provider:    "google",
			setEnvVar:   "",
			shouldPass:  false,
			description: "Should fail when required env var is not set",
		},
		{
			name:        "BM25 retriever (no requirements)",
			provider:    "bm25",
			setEnvVar:   "",
			shouldPass:  true,
			description: "Should pass when provider has no requirements",
		},
		{
			name:        "OpenAI provider with OPENAI_API_KEY",
			provider:    "openai",
			setEnvVar:   "OPENAI_API_KEY",
			shouldPass:  true,
			description: "Should pass when OpenAI key is set",
		},
		{
			name:        "OpenAI provider without key",
			provider:    "openai",
			setEnvVar:   "",
			shouldPass:  false,
			description: "Should fail when OpenAI key is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment
			os.Unsetenv("GOOGLE_API_KEY")
			os.Unsetenv("OPENAI_API_KEY")

			// Set env var if specified
			if tt.setEnvVar != "" {
				os.Setenv(tt.setEnvVar, "test-key")
			}

			err := registry.ValidateProvider(tt.provider)

			if tt.shouldPass && err != nil {
				t.Errorf("Expected validation to pass, but got error: %v", err)
			}

			if !tt.shouldPass && err == nil {
				t.Errorf("Expected validation to fail, but it passed")
			}

			// Clean up
			os.Unsetenv("GOOGLE_API_KEY")
			os.Unsetenv("OPENAI_API_KEY")
		})
	}
}

func TestProviderDependencyErrorMessage(t *testing.T) {
	tests := []struct {
		name            string
		provider        string
		expectedMessage string
	}{
		{
			name:            "Google provider error",
			provider:        "google",
			expectedMessage: "GOOGLE_API_KEY",
		},
		{
			name:            "OpenAI provider error",
			provider:        "openai",
			expectedMessage: "OPENAI_API_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewProviderDependencyRegistry()

			// Ensure env var is not set
			if tt.provider == "google" {
				os.Unsetenv("GOOGLE_API_KEY")
			} else if tt.provider == "openai" {
				os.Unsetenv("OPENAI_API_KEY")
			}

			err := registry.ValidateProvider(tt.provider)

			if err == nil {
				t.Fatalf("Expected error but got nil")
			}

			errMsg := err.Error()
			if !strings.Contains(errMsg, tt.expectedMessage) {
				t.Errorf("Expected error message to contain '%s', got: %s", tt.expectedMessage, errMsg)
			}
		})
	}
}
