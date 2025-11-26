package config

import (
	"fmt"
	"regexp"
)

// Normalize sets default values for the configuration. This ensures that the
// builder works with a complete and predictable config object.
func (c *Config) Normalize() {
	// Example of setting a default.
	if c.TopK == 0 {
		c.TopK = 10 // Default to 10 results
	}
}

// Validate checks the configuration for semantic correctness. It ensures that
// all required fields are present and that the values are valid. It also validates
// that all component references are valid and detects circular dependencies.
func (c *Config) Validate() error {
	if len(c.Components) == 0 {
		return fmt.Errorf("at least one component must be defined in the configuration")
	}

	// Validate individual components
	componentNames := make(map[string]bool)
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
		if comp.Type == "" {
			return fmt.Errorf("component type is required for component %q", comp.Name)
		}

		// Check for duplicate component names
		if componentNames[comp.Name] {
			return fmt.Errorf("duplicate component name %q", comp.Name)
		}
		componentNames[comp.Name] = true
	}

	// Validate component references
	if err := c.validateComponentReferences(componentNames); err != nil {
		return err
	}

	// Detect circular dependencies
	if err := c.detectCircularDependencies(componentNames); err != nil {
		return err
	}

	return nil
}

// validateComponentReferences checks that all references in component params
// point to valid components that exist in the configuration.
func (c *Config) validateComponentReferences(componentNames map[string]bool) error {
	for _, comp := range c.Components {
		if comp.Params == nil {
			continue
		}

		// Check for references to components in params
		for key, value := range comp.Params {
			if strVal, ok := value.(string); ok {
				// Check if this looks like a component reference (common parameter names)
				if isComponentReferenceKey(key) {
					if strVal != "" && !componentNames[strVal] {
						return fmt.Errorf("component %q references invalid component %q in param %q", comp.Name, strVal, key)
					}
				}
			}
		}
	}

	return nil
}

// isComponentReferenceKey returns true if the key name suggests it's a component reference
func isComponentReferenceKey(key string) bool {
	referencePatterns := []string{
		"retriever", "reranker", "llm", "embedder",
		"vectorstore", "vector_store", "state_provider",
		"state", "rules", "rule_set", "orchestrator",
		"provider", "schema_parser", "tool", "reasoner", "planner",
	}

	for _, pattern := range referencePatterns {
		if match, _ := regexp.MatchString(pattern, key); match {
			return true
		}
	}

	return false
}

// detectCircularDependencies checks for circular dependencies in component params.
// A circular dependency occurs when component A depends on component B, and B depends on A.
func (c *Config) detectCircularDependencies(componentNames map[string]bool) error {
	// Build dependency map
	dependencyMap := make(map[string][]string)

	for _, comp := range c.Components {
		deps := extractComponentDependencies(comp)
		dependencyMap[comp.Name] = deps
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for componentName := range componentNames {
		if !visited[componentName] {
			if hasCycle(componentName, visited, recStack, dependencyMap) {
				return fmt.Errorf("circular dependency detected involving component %q", componentName)
			}
		}
	}

	return nil
}

// extractComponentDependencies extracts all component names referenced in a component's params
func extractComponentDependencies(comp ComponentConfig) []string {
	var deps []string

	if comp.Params == nil {
		return deps
	}

	for key, value := range comp.Params {
		if strVal, ok := value.(string); ok {
			if isComponentReferenceKey(key) && strVal != "" {
				deps = append(deps, strVal)
			}
		}
	}

	return deps
}

// hasCycle performs depth-first search to detect cycles in the dependency graph
func hasCycle(node string, visited, recStack map[string]bool, graph map[string][]string) bool {
	visited[node] = true
	recStack[node] = true

	// Visit all adjacent nodes
	for _, neighbor := range graph[node] {
		if !visited[neighbor] {
			if hasCycle(neighbor, visited, recStack, graph) {
				return true
			}
		} else if recStack[neighbor] {
			// Back edge found, cycle exists
			return true
		}
	}

	recStack[node] = false
	return false
}
