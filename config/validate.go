package config

import "fmt"

// Normalize sets default values for the configuration. This ensures that the
// builder works with a complete and predictable config object.
func (c *Config) Normalize() {
	// Example of setting a default.
	if c.TopK == 0 {
		c.TopK = 10 // Default to 10 results
	}
}

// Validate checks the configuration for semantic correctness. It ensures that
// all required fields are present and that the values are valid.
func (c *Config) Validate() error {
	if len(c.Components) == 0 {
		return fmt.Errorf("at least one component must be defined in the configuration")
	}

	for _, comp := range c.Components {
		if comp.Name == "" {
			return fmt.Errorf("component name is required")
		}
		if comp.Kind == "" {
			return fmt.Errorf("component kind is required for component %q", comp.Name)
		}
		if comp.Params == nil {
			return fmt.Errorf("component params are required for component %q", comp.Name)
		}
	}

	return nil
}
