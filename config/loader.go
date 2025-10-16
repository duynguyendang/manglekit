package config

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFromYAML reads a YAML configuration from the provided reader and returns a
// Config object. It also expands environment variables in the YAML content.
func LoadFromYAML(r io.Reader) (*Config, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}

	expandedContent := []byte(os.ExpandEnv(string(content)))

	var cfg Config
	if err := yaml.Unmarshal(expandedContent, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &cfg, nil
}

// LoadFromYAMLFile reads a YAML configuration file from the given path.
func LoadFromYAMLFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %q: %w", path, err)
	}
	defer f.Close()

	return LoadFromYAML(f)
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
