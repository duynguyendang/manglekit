package config

import "github.com/duynguyendang/manglekit/core"

// Config is the root of the Manglekit configuration. It defines the components
// that make up a pipeline and their settings.
type Config struct {
	Components        []ComponentConfig `yaml:"components"`
	TopK              int               `yaml:"topK,omitempty"`
	MaxTokens         int               `yaml:"maxTokens,omitempty"`
	Orchestrator      string            `yaml:"orchestrator,omitempty"`
	Updatable         string            `yaml:"updatable,omitempty"`
	FallbackThreshold float64           `yaml:"fallbackThreshold,omitempty"`
}

// ComponentConfig represents a single component in the configuration.
type ComponentConfig struct {
	Name   string         `yaml:"name"`
	Type   string         `yaml:"type"`
	Kind   core.Kind      `yaml:"kind"`
	Params map[string]any `yaml:"params"`
}
