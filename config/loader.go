package config

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads a YAML configuration file from the given path and returns a Config object.
// It also expands environment variables in the YAML content.
// This function ports the legacy loading logic to the new architecture.
//
// Environment variable expansion supports the standard ${VAR_NAME} syntax.
// Example: ${API_KEY} will be replaced with the value of the API_KEY environment variable.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}
	return ParseConfig(data)
}

// ParseConfig unmarshals a byte slice into a Config object.
// It expands environment variables referenced with ${VAR_NAME} or ${VAR_NAME:-default}
// syntax before unmarshaling. Only explicitly referenced variables are expanded.
// Warnings about ${VAR} references that expanded to an empty string (unset or
// not allowlisted) are printed to stderr.
func ParseConfig(data []byte) (*Config, error) {
	expandedContent, warnings := expandEnvWithDefaults(string(data))
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "config: warning: %s\n", w)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(expandedContent), &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	applyDefaults(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

var allowedEnvPrefixes = []string{
	"MANGLEKIT_",
	"POLICY_",
	"SERVICE_",
	"LOG_",
	"OTLP_",
	"API_",
	"GOOGLE_",
	"OPENAI_",
	"MCP_",
}

func isAllowedEnvVar(name string) bool {
	for _, prefix := range allowedEnvPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// expandEnvWithDefaults expands ${VAR} / ${VAR:-default} references and
// returns the expanded string plus warnings for references that silently
// expanded to an empty string (variable unset or not allowlisted and no
// default provided).
func expandEnvWithDefaults(s string) (string, []string) {
	var warnings []string
	warned := make(map[string]bool)
	warn := func(varName string) {
		if warned[varName] {
			return
		}
		warned[varName] = true
		warnings = append(warnings, fmt.Sprintf(
			"${%s} expanded to an empty string (variable is unset or not allowlisted); "+
				"if this is unintentional, fix the variable name or provide a default with ${%s:-value}",
			varName, varName))
	}

	out := os.Expand(s, func(key string) string {
		varName := key
		defaultVal := ""
		hasDefault := false

		if idx := strings.Index(key, ":-"); idx >= 0 {
			varName = key[:idx]
			defaultVal = key[idx+2:]
			hasDefault = true
		}

		if !isAllowedEnvVar(varName) {
			if hasDefault {
				return defaultVal
			}
			warn(varName)
			return ""
		}

		if v, ok := os.LookupEnv(varName); ok {
			return v
		}
		if !hasDefault {
			warn(varName)
		}
		return defaultVal
	})
	return out, warnings
}

// LoadFromReader reads a YAML configuration from the provided reader and returns a Config object.
// It also expands environment variables in the YAML content.
func LoadFromReader(r io.Reader) (*Config, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}
	return ParseConfig(content)
}

// applyDefaults applies sensible defaults to the configuration if not already set.
func applyDefaults(cfg *Config) {
	if cfg.Observability.ServiceName == "" {
		cfg.Observability.ServiceName = DefaultServiceName
	}

	if cfg.Observability.LogLevel == "" {
		cfg.Observability.LogLevel = DefaultLogLevel
	}

	// MaxSteps: zero means "use the SDK default", which is
	// sdk.DefaultMaxSteps. Materialize it here so the config carries the
	// effective value in one place.
	if cfg.Policy.MaxSteps == 0 {
		cfg.Policy.MaxSteps = DefaultPolicyMaxSteps
	}

	// EvaluationTimeout: zero means "no explicit timeout" (engine default),
	// which is the documented default.
	if cfg.Policy.EvaluationTimeout == 0 {
		cfg.Policy.EvaluationTimeout = DefaultPolicyEvaluationTimeout
	}
}
