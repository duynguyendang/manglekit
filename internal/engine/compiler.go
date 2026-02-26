package engine

import (
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit-wip/internal/engine/parse"
)

// GherkinCompiler transforms Gherkin feature files into Datalog rules.
type GherkinCompiler struct {
	registry *StepRegistry
}

// NewGherkinCompiler creates a new compiler with the standard step registry.
func NewGherkinCompiler() *GherkinCompiler {
	return &GherkinCompiler{
		registry: NewStepRegistry(),
	}
}

// NewGherkinCompilerWithRegistry creates a compiler with a custom registry.
func NewGherkinCompilerWithRegistry(registry *StepRegistry) *GherkinCompiler {
	return &GherkinCompiler{
		registry: registry,
	}
}

// Compile transforms a parsed Feature into Datalog rules.
// Returns the complete Datalog program as a string.
func (c *GherkinCompiler) Compile(feature *parse.Feature) (string, error) {
	var builder strings.Builder

	// Add header comment
	builder.WriteString(fmt.Sprintf("%% Generated from Feature: %s\n", feature.Name))
	if feature.Description != "" {
		builder.WriteString(fmt.Sprintf("%% %s\n", feature.Description))
	}
	builder.WriteString("\n")

	// Compile each scenario
	for i, scenario := range feature.Scenarios {
		datalog, err := c.compileScenario(scenario)
		if err != nil {
			return "", fmt.Errorf("scenario %q: %w", scenario.Name, err)
		}

		builder.WriteString(fmt.Sprintf("%% Scenario: %s\n", scenario.Name))
		builder.WriteString(datalog)
		builder.WriteString("\n")

		// Add blank line between scenarios (except after the last one)
		if i < len(feature.Scenarios)-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}

// compileScenario transforms a single scenario into a Datalog clause.
// The transformation follows the pattern: head :- body.
// - Given/When steps form the body (conditions)
// - Then steps form the head (outcome)
func (c *GherkinCompiler) compileScenario(scenario parse.Scenario) (string, error) {
	if len(scenario.Steps) == 0 {
		return "", fmt.Errorf("scenario has no steps")
	}

	var givenSteps []parse.Step
	var whenSteps []parse.Step
	var thenSteps []parse.Step

	// Classify steps by type, handling And/But
	lastType := parse.StepType("")
	for _, step := range scenario.Steps {
		stepType := step.NormalizeStepType(lastType)

		switch stepType {
		case parse.StepTypeGiven:
			givenSteps = append(givenSteps, step)
			lastType = parse.StepTypeGiven
		case parse.StepTypeWhen:
			whenSteps = append(whenSteps, step)
			lastType = parse.StepTypeWhen
		case parse.StepTypeThen:
			thenSteps = append(thenSteps, step)
			lastType = parse.StepTypeThen
		}
	}

	// Must have at least one Then step (the head)
	if len(thenSteps) == 0 {
		return "", fmt.Errorf("scenario must have at least one Then step")
	}

	// Generate Datalog fragments
	var bodyFragments []string

	// Add When conditions first (action triggers)
	for _, step := range whenSteps {
		fragment, err := c.registry.GenerateDatalog(step, parse.StepTypeWhen)
		if err != nil {
			return "", fmt.Errorf("when step %q: %w", step.Text, err)
		}
		bodyFragments = append(bodyFragments, fragment)
	}

	// Add Given conditions (preconditions)
	for _, step := range givenSteps {
		fragment, err := c.registry.GenerateDatalog(step, parse.StepTypeGiven)
		if err != nil {
			return "", fmt.Errorf("given step %q: %w", step.Text, err)
		}
		bodyFragments = append(bodyFragments, fragment)
	}

	// Generate Then outcomes (heads)
	var rules []string
	for _, step := range thenSteps {
		head, err := c.registry.GenerateDatalog(step, parse.StepTypeThen)
		if err != nil {
			return "", fmt.Errorf("then step %q: %w", step.Text, err)
		}

		// Build the complete rule
		if len(bodyFragments) > 0 {
			// Rule with conditions: head :- body.
			body := strings.Join(bodyFragments, ",\n    ")
			rule := fmt.Sprintf("%s :-\n    %s.", head, body)
			rules = append(rules, rule)
		} else {
			// Fact (no conditions): head.
			rule := fmt.Sprintf("%s.", head)
			rules = append(rules, rule)
		}
	}

	return strings.Join(rules, "\n\n"), nil
}

// CompileFromString parses and compiles a Gherkin feature string.
func (c *GherkinCompiler) CompileFromString(content string) (string, error) {
	feature, err := parse.ParseFeature(content)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	return c.Compile(feature)
}
