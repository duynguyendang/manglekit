package config

// Config is the root of the Manglekit configuration. It defines the components
// that make up a pipeline and their settings.
type Config struct {
	LLM       *LLMConfig      `yaml:"llm,omitempty"`
	Embedder  *EmbedderConfig `yaml:"embedder,omitempty"`
	Retrieve  *RetrieveConfig `yaml:"retrieve,omitempty"`
	Rerank    *RerankConfig   `yaml:"rerank,omitempty"`
	Vector    *VectorConfig   `yaml:"vector,omitempty"`
	State     *StateConfig    `yaml:"state,omitempty"`
	Parsers   []ParserConfig  `yaml:"parsers,omitempty"`
	Rules     []RuleConfig    `yaml:"rules,omitempty"`
	Clients   map[string]any  `yaml:"clients,omitempty"` // For generic clients like http
	TopK      int             `yaml:"topK,omitempty"`
	MaxTokens int             `yaml:"maxTokens,omitempty"`
}

// LLMConfig defines the configuration for a Language Model provider.
type LLMConfig struct {
	Provider string `yaml:"provider"`
	// Options contains provider-specific settings (e.g., model name, API key).
	// This will be decoded into a provider-specific options struct.
	Options map[string]any `yaml:"options,omitempty"`
}

// EmbedderConfig defines the configuration for a text embedding provider.
type EmbedderConfig struct {
	Provider string         `yaml:"provider"`
	Options  map[string]any `yaml:"options,omitempty"`
}

// RetrieveConfig defines the configuration for a document retriever.
type RetrieveConfig struct {
	Provider string `yaml:"provider"`
	// For a 'hybrid' retriever, this defines the sub-retrievers.
	Retrievers []RetrieveConfig `yaml:"retrievers,omitempty"`
	Options    map[string]any   `yaml:"options,omitempty"`
}

// RerankConfig defines the configuration for a document reranker.
type RerankConfig struct {
	Provider string         `yaml:"provider"`
	Options  map[string]any `yaml:"options,omitempty"`
}

// VectorConfig defines the configuration for a vector store.
type VectorConfig struct {
	Provider string         `yaml:"provider"`
	Options  map[string]any `yaml:"options,omitempty"`
}

// StateConfig defines the configuration for a state provider.
type StateConfig struct {
	Provider string         `yaml:"provider"`
	Options  map[string]any `yaml:"options,omitempty"`
}

// ParserConfig defines a document parser.
type ParserConfig struct {
	//TODO: Define parser configuration
}

// RuleConfig defines a set of rules for the rules engine.
type RuleConfig struct {
	//TODO: Define rules configuration
}