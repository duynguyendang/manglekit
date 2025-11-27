package config

// Config is the top-level configuration structure for Manglekit.
// It defines all the settings needed to initialize a Manglekit Client,
// including policies, observability, and pre-defined actions.
// This struct replaces the legacy configuration from v1.
type Config struct {
	// Policy configuration
	Policy PolicyConfig `yaml:"policy" mapstructure:"policy"`

	// Observability settings
	Observability ObservabilityConfig `yaml:"observability" mapstructure:"observability"`

	// Pre-defined Actions (LLMs, Retrievers) that can be loaded by name.
	// These are optional and reserved for future use.
	Actions map[string]ActionConfig `yaml:"actions" mapstructure:"actions"`
}

// PolicyConfig defines settings for the Policy Engine.
type PolicyConfig struct {
	// Path to the Datalog policy file (.dl)
	Path string `yaml:"path" mapstructure:"path"`

	// Timeout for policy evaluation (in seconds)
	EvaluationTimeout int `yaml:"evaluation_timeout,omitempty" mapstructure:"evaluation_timeout"`
}

// ObservabilityConfig defines settings for logging and tracing.
type ObservabilityConfig struct {
	// Enabled indicates whether observability is active
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// ServiceName is the name of the service for observability reporting
	ServiceName string `yaml:"service_name,omitempty" mapstructure:"service_name"`

	// LogLevel sets the logging level (e.g., "debug", "info", "warn", "error")
	LogLevel string `yaml:"log_level,omitempty" mapstructure:"log_level"`

	// OTLPEndpoint is the OpenTelemetry Protocol endpoint for traces and metrics
	OTLPEndpoint string `yaml:"otlp_endpoint,omitempty" mapstructure:"otlp_endpoint"`
}

// ActionConfig represents a pre-defined action (LLM, Retriever, etc.)
type ActionConfig struct {
	// Type indicates the action type (e.g., "llm", "retriever")
	Type string `yaml:"type" mapstructure:"type"`

	// Provider specifies the provider (e.g., "google", "openai")
	Provider string `yaml:"provider" mapstructure:"provider"`

	// Options are provider-specific configuration options
	Options map[string]interface{} `yaml:"options" mapstructure:"options"`
}
