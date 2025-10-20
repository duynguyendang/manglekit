package declarative

// ToolStepConfig defines a single tool to be executed.
// The "Name" must match the name of a component defined elsewhere in the config.
type ToolStepConfig struct {
	Name string `yaml:"name"`
}

// Options defines the configuration for the declarative orchestrator.
type Options struct {
	Steps []ToolStepConfig `yaml:"steps"`
}
