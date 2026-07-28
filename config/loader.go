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
func ParseConfig(data []byte) (*Config, error) {
	expandedContent := []byte(expandEnvWithDefaults(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(expandedContent, &cfg); err != nil {
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

func expandEnvWithDefaults(s string) string {
	return os.Expand(s, func(key string) string {
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
			return ""
		}

		if v, ok := os.LookupEnv(varName); ok {
			return v
		}
		return defaultVal
	})
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
		cfg.Observability.ServiceName = "manglekit-app"
	}

	if cfg.Observability.LogLevel == "" {
		cfg.Observability.LogLevel = "info"
	}
}
