package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Load reads a YAML configuration file from the given path and returns a Config object.
// It also expands environment variables in the YAML content.
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
// It also expands environment variables in the YAML content before unmarshaling.
func ParseConfig(data []byte) (*Config, error) {
	// Expand environment variables in the YAML content
	expandedContent := []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(expandedContent, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
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
