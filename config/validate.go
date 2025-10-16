package config

import "fmt"

// Normalize sets default values for the configuration. This ensures that the
// builder works with a complete and predictable config object.
func (c *Config) Normalize() {
	// Example of setting a default.
	if c.TopK == 0 {
		c.TopK = 10 // Default to 10 results
	}
	// TODO: Add normalization logic for all components, filling in default
	// providers, models, etc. where they are not specified.
}

// Validate checks the configuration for semantic correctness. It ensures that
// all required fields are present and that the values are valid.
func (c *Config) Validate() error {
	if c.LLM == nil {
		return fmt.Errorf("llm configuration is required")
	}
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm.provider is required")
	}

	if c.Embedder == nil {
		return fmt.Errorf("embedder configuration is required")
	}
	if c.Embedder.Provider == "" {
		return fmt.Errorf("embedder.provider is required")
	}

	if c.Retrieve == nil {
		return fmt.Errorf("retrieve configuration is required")
	}
	if c.Retrieve.Provider == "" {
		return fmt.Errorf("retrieve.provider is required")
	}

	// TODO: Add comprehensive validation for all components, checking for
	// things like provider-specific required options.

	return nil
}
