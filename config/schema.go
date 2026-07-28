package config

import "fmt"

// Config is the root configuration structure for Manglekit.
// It maps to the YAML configuration file and defines all settings for the system.
type Config struct {
	// Policy configuration for the Datalog engine.
	Policy PolicyConfig `yaml:"policy" mapstructure:"policy"`

	// FailureMode determines how the system behaves when the policy engine or guard fails.
	// - "closed" (Default): Blocks the action (returns error).
	// - "open": Allows the action to proceed (logs warning).
	FailureMode string `yaml:"failure_mode" mapstructure:"failure_mode"`

	// Observability configuration (Logging and Tracing).
	Observability ObservabilityConfig `yaml:"observability" mapstructure:"observability"`

	// Actions defines pre-configured actions that can be referenced by name.
	// This maps action names to their configuration.
	Actions map[string]ActionConfig `yaml:"actions" mapstructure:"actions"`

	// MCP defines a list of Model Context Protocol servers to connect to.
	MCP []MCPServerConfig `yaml:"mcp" mapstructure:"mcp"`

	// Knowledge configuration for static RDF facts.
	Knowledge KnowledgeConfig `yaml:"knowledge" mapstructure:"knowledge"`

	// Memory configuration for Semantic Memory (RAG).
	Memory MemoryConfig `yaml:"memory" mapstructure:"memory"`
}

const (
	// FailureModeClosed ensures the system blocks on governance errors.
	FailureModeClosed = "closed"
	// FailureModeOpen allows the system to proceed (fail-open) on governance errors.
	FailureModeOpen = "open"
)

// KnowledgeConfig settings for loading static knowledge bases.
type KnowledgeConfig struct {
	// Path to the RDF Turtle (.ttl) file containing static facts.
	Path string `yaml:"path" mapstructure:"path"`
}

// MemoryConfig settings for Semantic Memory provider.
type MemoryConfig struct {
	// Provider specifies the memory backend (e.g., "inmem", "qdrant").
	Provider string `yaml:"provider" mapstructure:"provider"`

	// Path is a file path or connection string for the provider.
	Path string `yaml:"path" mapstructure:"path"`

	// Options contains arbitrary provider-specific settings.
	Options map[string]interface{} `yaml:"options" mapstructure:"options"`
}

// PolicyConfig settings for the Datalog Policy Engine.
type PolicyConfig struct {
	// Path to the Datalog policy source file (.dl or .dlog) or directory.
	Path string `yaml:"path" mapstructure:"path"`

	// EvaluationTimeout is the max duration (in seconds) for rule evaluation.
	EvaluationTimeout int `yaml:"evaluation_timeout,omitempty" mapstructure:"evaluation_timeout"`

	// MaxSteps limits the total number of loop iterations in the Semantic State Machine.
	// Zero means use the SDK default (10). YAML key: max_steps.
	MaxSteps int `yaml:"max_steps,omitempty" mapstructure:"max_steps"`

	// SteeringEnabled controls whether EAST (Entropic Activation Steering) prompt
	// injection is active. When false, prompts pass through unmodified.
	// Default: false (disabled until validated). YAML key: steering_enabled.
	SteeringEnabled bool `yaml:"steering_enabled,omitempty" mapstructure:"steering_enabled"`

	// ParadoxThreshold is the EAST magnitude above which cognitive paradox injection
	// is triggered. Only effective when SteeringEnabled is true.
	// Default: 0.8. YAML key: paradox_threshold.
	ParadoxThreshold float64 `yaml:"paradox_threshold,omitempty" mapstructure:"paradox_threshold"`
}

// ObservabilityConfig settings for telemetry.
type ObservabilityConfig struct {
	// Enabled toggles all observability features.
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// ServiceName is the application name used in traces and logs.
	ServiceName string `yaml:"service_name,omitempty" mapstructure:"service_name"`

	// LogLevel sets the minimum log severity ("debug", "info", "warn", "error").
	LogLevel string `yaml:"log_level,omitempty" mapstructure:"log_level"`

	// OTLPEndpoint is the URL of the OpenTelemetry collector (gRPC/HTTP).
	OTLPEndpoint string `yaml:"otlp_endpoint,omitempty" mapstructure:"otlp_endpoint"`
}

// ActionConfig defines a static action configuration.
type ActionConfig struct {
	// Type identifies the kind of action (e.g., "llm", "retriever").
	Type string `yaml:"type" mapstructure:"type"`

	// Provider specifies the implementation provider (e.g., "google", "openai").
	Provider string `yaml:"provider" mapstructure:"provider"`

	// FailOnStartup determines if the application should crash if this action fails to load.
	FailOnStartup bool `yaml:"fail_on_startup" mapstructure:"fail_on_startup"`

	// Options contains arbitrary provider-specific settings.
	Options map[string]interface{} `yaml:"options" mapstructure:"options"`
}

// Validate checks the configuration for invalid values.
func (c *Config) Validate() error {
	switch c.FailureMode {
	case "", FailureModeClosed, FailureModeOpen:
	default:
		return fmt.Errorf("failure_mode must be %q or %q, got %q", FailureModeClosed, FailureModeOpen, c.FailureMode)
	}

	switch c.Observability.LogLevel {
	case "", "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log_level must be one of debug, info, warn, error; got %q", c.Observability.LogLevel)
	}

	if c.Policy.EvaluationTimeout < 0 {
		return fmt.Errorf("policy.evaluation_timeout must be non-negative, got %d", c.Policy.EvaluationTimeout)
	}

	if c.Policy.MaxSteps < 0 {
		return fmt.Errorf("policy.max_steps must be non-negative, got %d", c.Policy.MaxSteps)
	}

	return nil
}

// MCPServerConfig defines how to connect to an MCP server.
type MCPServerConfig struct {
	// Name is a unique identifier for this MCP server connection.
	Name string `yaml:"name" mapstructure:"name"`
	// Transport specifies the connection method: "stdio" or "sse".
	Transport string `yaml:"transport" mapstructure:"transport"`
	// Command is the executable command (for stdio) or URL (for sse).
	Command string `yaml:"command" mapstructure:"command"`
	// Args are command-line arguments (for stdio).
	Args []string `yaml:"args" mapstructure:"args"`
	// Env specifies environment variables for the process (for stdio).
	Env []string `yaml:"env" mapstructure:"env"`
	// FailOnStartup determines if the application should crash if this server fails to connect.
	FailOnStartup bool `yaml:"fail_on_startup" mapstructure:"fail_on_startup"`
	// Tools lists expected tool names for resilience.
	// If the server fails to connect, these tools will be registered as "Unhealthy"
	// so the agent knows they exist but are unavailable.
	Tools []string `yaml:"tools" mapstructure:"tools"`
}
