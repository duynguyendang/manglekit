package google

import (
	"testing"
)

func TestFactoryMissingAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	_, err := Factory(map[string]any{})
	if err == nil {
		t.Fatal("expected error when api_key is missing and env is unset")
	}
}

func TestFactoryUsesEnvAPIKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "env-key")
	opt, err := Factory(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error with env api key: %v", err)
	}
	if opt == nil {
		t.Fatal("expected a non-nil ClientOption")
	}
}

func TestFactoryExplicitAPIKey(t *testing.T) {
	opt, err := Factory(map[string]any{
		"api_key": "explicit-key",
		"model":   "gemini-2.0-flash",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt == nil {
		t.Fatal("expected a non-nil ClientOption")
	}
}

func TestFactoryDefaultModel(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "env-key")
	opt, err := Factory(map[string]any{"_action_name": "my_llm"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opt == nil {
		t.Fatal("expected a non-nil ClientOption")
	}
}
