// File: pipeline/sandwich/options.go
package sandwich

import "github.com/duynguyendang/manglekit/core"

// Options defines the YAML configuration for the Sandwich orchestrator.
type Options struct {
	// Named dependencies
	LLM           string `yaml:"llm"`
	Retriever     string `yaml:"retriever"`
	Reranker      string `yaml:"reranker,omitempty"`
	StateProvider string `yaml:"state_provider,omitempty"`
	RuleSet       string `yaml:"ruleSet,omitempty"`

	// Migrated global settings
	TopK              int     `yaml:"top_k,omitempty"`
	MaxTokens         int     `yaml:"max_tokens,omitempty"`
	FallbackThreshold float64 `yaml:"fallback_threshold,omitempty"`
}

// ProviderName returns the name of the provider.
func (o *Options) ProviderName() string {
	return "sandwich"
}

// ProviderKind returns the kind of the provider.
func (o *Options) ProviderKind() core.Kind {
	return core.KindOrchestrator
}
