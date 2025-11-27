package declarative

import "github.com/duynguyendang/manglekit/v1/core"

// ToolStepConfig defines a single tool to be executed.
// The "Name" must match the name of a component defined elsewhere in the config.
type ToolStepConfig struct {
	Name   string         `yaml:"name"`
	Params map[string]any `yaml:"params,omitempty"`
}

// Options defines the configuration for the declarative orchestrator.
type Options struct {
	Steps         []ToolStepConfig `yaml:"steps"`
	StateProvider string           `yaml:"state_provider,omitempty"`
}

// ProviderName returns the name of the provider.
func (o *Options) ProviderName() string { return "declarative" }

// ProviderKind returns the kind of the provider.
func (o *Options) ProviderKind() core.Kind { return core.KindOrchestrator }
