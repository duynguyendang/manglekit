package config

import (
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "embedder1",
				Type:   "openai",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
			{
				Name:   "retriever1",
				Type:   "dense",
				Kind:   core.KindRetriever,
				Params: map[string]any{"embedder": "embedder1"},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected valid config to pass validation, got error: %v", err)
	}
}

func TestValidate_MissingComponentName(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "",
				Type:   "openai",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for missing component name")
	}
	if err.Error() != "component name is required" {
		t.Errorf("Expected 'component name is required' error, got: %v", err)
	}
}

func TestValidate_MissingComponentKind(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "embedder1",
				Type:   "openai",
				Kind:   "",
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for missing component kind")
	}
	if err.Error() != "component kind is required for component \"embedder1\"" {
		t.Errorf("Expected 'component kind is required' error, got: %v", err)
	}
}

func TestValidate_MissingComponentType(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "embedder1",
				Type:   "",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for missing component type")
	}
	if err.Error() != "component type is required for component \"embedder1\"" {
		t.Errorf("Expected 'component type is required' error, got: %v", err)
	}
}

func TestValidate_MissingComponentParams(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "embedder1",
				Type:   "openai",
				Kind:   core.KindEmbedder,
				Params: nil,
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for missing component params")
	}
	if err.Error() != "component params are required for component \"embedder1\"" {
		t.Errorf("Expected 'component params are required' error, got: %v", err)
	}
}

func TestValidate_EmptyComponentList(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for empty component list")
	}
	if err.Error() != "at least one component must be defined in the configuration" {
		t.Errorf("Expected 'at least one component must be defined' error, got: %v", err)
	}
}

func TestValidate_DuplicateComponentName(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "embedder1",
				Type:   "openai",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
			{
				Name:   "embedder1",
				Type:   "openai",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-large"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for duplicate component name")
	}
	if err.Error() != "duplicate component name \"embedder1\"" {
		t.Errorf("Expected 'duplicate component name' error, got: %v", err)
	}
}

func TestValidate_InvalidComponentReference(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "retriever1",
				Type:   "dense",
				Kind:   core.KindRetriever,
				Params: map[string]any{"embedder": "nonexistent-embedder"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for invalid component reference")
	}
	if err.Error() != "component \"retriever1\" references invalid component \"nonexistent-embedder\" in param \"embedder\"" {
		t.Errorf("Expected 'invalid component reference' error, got: %v", err)
	}
}

func TestValidate_ValidComponentReferences(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "embedder1",
				Type:   "openai-embedder",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
			{
				Name:   "retriever1",
				Type:   "bm25",
				Kind:   core.KindRetriever,
				Params: map[string]any{},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected valid component references to pass validation, got error: %v", err)
	}
}

func TestValidate_MultipleInvalidReferences(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name: "reranker1",
				Type: "cosine",
				Kind: core.KindReranker,
				Params: map[string]any{
					"embedder": "invalid-embedder",
				},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for invalid component references")
	}
	// Should catch the invalid reference
	if err.Error() != "component \"reranker1\" references invalid component \"invalid-embedder\" in param \"embedder\"" {
		t.Errorf("Expected 'invalid component reference' error, got: %v", err)
	}
}

func TestValidate_DirectCircularDependency(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "retriever1",
				Type:   "hybrid",
				Kind:   core.KindRetriever,
				Params: map[string]any{"retriever": "retriever1"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for direct circular dependency")
	}
	if err.Error() != "circular dependency detected involving component \"retriever1\"" {
		t.Errorf("Expected 'circular dependency detected' error, got: %v", err)
	}
}

func TestValidate_IndirectCircularDependency(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "retriever1",
				Type:   "hybrid",
				Kind:   core.KindRetriever,
				Params: map[string]any{"retriever": "retriever2"},
			},
			{
				Name:   "retriever2",
				Type:   "hybrid",
				Kind:   core.KindRetriever,
				Params: map[string]any{"retriever": "retriever1"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for indirect circular dependency")
	}
	if err.Error() != "circular dependency detected involving component \"retriever1\"" {
		t.Errorf("Expected 'circular dependency detected' error, got: %v", err)
	}
}

func TestValidate_LongerCircularDependency(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "retriever1",
				Type:   "hybrid",
				Kind:   core.KindRetriever,
				Params: map[string]any{"retriever": "retriever2"},
			},
			{
				Name:   "retriever2",
				Type:   "hybrid",
				Kind:   core.KindRetriever,
				Params: map[string]any{"retriever": "retriever3"},
			},
			{
				Name:   "retriever3",
				Type:   "hybrid",
				Kind:   core.KindRetriever,
				Params: map[string]any{"retriever": "retriever1"},
			},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Expected validation to fail for longer circular dependency")
	}
	// The error message varies depending on which component is detected last
	if !strings.Contains(err.Error(), "circular dependency detected") {
		t.Errorf("Expected 'circular dependency detected' error, got: %v", err)
	}
}

func TestValidate_NoDependencyNoCircularDependency(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "retriever1",
				Type:   "dense",
				Kind:   core.KindRetriever,
				Params: map[string]any{"embedder": "embedder1"},
			},
			{
				Name:   "retriever2",
				Type:   "dense",
				Kind:   core.KindRetriever,
				Params: map[string]any{"embedder": "embedder2"},
			},
			{
				Name:   "embedder1",
				Type:   "openai",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
			{
				Name:   "embedder2",
				Type:   "openai",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-large"},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected valid config with no circular dependencies to pass validation, got error: %v", err)
	}
}

func TestValidate_EmptyStringReference(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "retriever1",
				Type:   "dense",
				Kind:   core.KindRetriever,
				Params: map[string]any{"embedder": ""},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected config with empty string reference to pass validation, got error: %v", err)
	}
}

func TestValidate_NonStringParamValue(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name: "retriever1",
				Type: "dense",
				Kind: core.KindRetriever,
				Params: map[string]any{
					"embedder":  "embedder1",
					"topK":      10,
					"threshold": 0.5,
				},
			},
			{
				Name:   "embedder1",
				Type:   "openai",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected config with non-string param values to pass validation, got error: %v", err)
	}
}

func TestValidate_ComplexValidConfig(t *testing.T) {
	cfg := &Config{
		Components: []ComponentConfig{
			{
				Name:   "embedder1",
				Type:   "openai-embedder",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
			{
				Name:   "embedder2",
				Type:   "openai-embedder",
				Kind:   core.KindEmbedder,
				Params: map[string]any{"model": "text-embedding-3-small"},
			},
			{
				Name:   "retriever1",
				Type:   "bm25",
				Kind:   core.KindRetriever,
				Params: map[string]any{},
			},
			{
				Name: "retriever2",
				Type: "hybrid",
				Kind: core.KindRetriever,
				Params: map[string]any{
					"sub_retrievers": []string{"retriever1"},
				},
			},
			{
				Name:   "reranker1",
				Type:   "cosine",
				Kind:   core.KindReranker,
				Params: map[string]any{"topK": 5},
			},
			{
				Name:   "llm1",
				Type:   "openai",
				Kind:   core.KindLLM,
				Params: map[string]any{"model": "gpt-4o-mini"},
			},
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("Expected complex valid config to pass validation, got error: %v", err)
	}
}

func TestIsComponentReferenceKey(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"retriever", true},
		{"reranker", true},
		{"llm", true},
		{"embedder", true},
		{"state_provider", true},
		{"rule_set", true},
		{"orchestrator", true},
		{"provider", true},
		{"schema_parser", true},
		{"my_retriever", true},
		{"the_reranker_name", true},
		{"topK", false},
		{"model", false},
		{"threshold", false},
		{"custom_param", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isComponentReferenceKey(tt.key)
			if result != tt.expected {
				t.Errorf("isComponentReferenceKey(%q) = %v, expected %v", tt.key, result, tt.expected)
			}
		})
	}
}
