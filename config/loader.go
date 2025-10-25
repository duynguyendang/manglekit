package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// ParseConfig unmarshals a byte slice into a Config object.
// It also expands environment variables in the YAML content.
func ParseConfig(data []byte) (*Config, error) {
	expandedContent := []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(expandedContent, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &cfg, nil
}

// LoadConfig reads a YAML configuration file from the given path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}
	return ParseConfig(data)
}

// LoadFromYAML reads a YAML configuration from the provided reader and returns a
// Config object. It also expands environment variables in the YAML content.
func LoadFromYAML(r io.Reader) (*Config, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}
	return ParseConfig(content)
}

// LoadFromEnv loads configuration from environment variables.
// This is a placeholder for now and will be implemented based on the final
// environment variable strategy.
func LoadFromEnv() (*Config, error) {
	// TODO: Implement environment variable loading logic.
	// This will involve reading MANGLEKIT_* variables and mapping them
	// to the Config struct, likely using a library or reflection.
	return &Config{}, nil
}
