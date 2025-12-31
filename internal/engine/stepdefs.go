package engine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/duynguyendang/manglekit/internal/engine/parse"
)

// StepDefinition defines a mapping from a Gherkin step pattern to a Datalog template.
type StepDefinition struct {
	Pattern  *regexp.Regexp
	Template string // Datalog template with {placeholder} syntax
	StepType parse.StepType
}

// StepRegistry holds all registered step definitions.
type StepRegistry struct {
	definitions []StepDefinition
}

// NewStepRegistry creates a new registry with standard step definitions.
func NewStepRegistry() *StepRegistry {
	registry := &StepRegistry{
		definitions: []StepDefinition{},
	}

	// Register standard step patterns
	registry.registerStandardSteps()

	return registry
}

// registerStandardSteps registers the built-in step patterns.
func (r *StepRegistry) registerStandardSteps() {
	// Given steps - Contextual preconditions
	r.Register(
		`^the user has "([^"]+)" label$`,
		`label("{label}")`,
		parse.StepTypeGiven,
	)

	r.Register(
		`^the entity is labeled "([^"]+)"$`,
		`label("{label}")`,
		parse.StepTypeGiven,
	)

	r.Register(
		`^the input contains "([^"]+)"$`,
		`meta("input_text", Text), fn:contains(Text, "{value}")`,
		parse.StepTypeGiven,
	)

	r.Register(
		`^the metadata "([^"]+)" is "([^"]+)"$`,
		`meta("{key}", "{value}")`,
		parse.StepTypeGiven,
	)

	r.Register(
		`^the metadata "([^"]+)" equals "([^"]+)"$`,
		`meta("{key}", "{value}")`,
		parse.StepTypeGiven,
	)

	// When steps - Action triggers
	r.Register(
		`^calling "([^"]+)"$`,
		`action_operation(Req, "{action}")`,
		parse.StepTypeWhen,
	)

	r.Register(
		`^calling the action "([^"]+)"$`,
		`action_operation(Req, "{action}")`,
		parse.StepTypeWhen,
	)

	r.Register(
		`^the action is "([^"]+)"$`,
		`action_operation(Req, "{action}")`,
		parse.StepTypeWhen,
	)

	// Then steps - Decision outcomes
	r.Register(
		`^halt with "([^"]+)"$`,
		`halt(Req, "{reason}")`,
		parse.StepTypeThen,
	)

	r.Register(
		`^retry with "([^"]+)"$`,
		`retry(Req, "{feedback}")`,
		parse.StepTypeThen,
	)

	r.Register(
		`^route to "([^"]+)"$`,
		`route(Req, "{target}")`,
		parse.StepTypeThen,
	)

	r.Register(
		`^allow the request$`,
		`allow(Req)`,
		parse.StepTypeThen,
	)
}

// Register adds a new step definition to the registry.
// Pattern should be a regex with capturing groups.
// Template should use {name} placeholders that will be replaced with captured values.
func (r *StepRegistry) Register(pattern, template string, stepType parse.StepType) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}

	r.definitions = append(r.definitions, StepDefinition{
		Pattern:  re,
		Template: template,
		StepType: stepType,
	})

	return nil
}

// Match attempts to match a step against registered definitions.
// Returns the matched definition and extracted parameters.
func (r *StepRegistry) Match(step parse.Step, expectedType parse.StepType) (*StepDefinition, map[string]string, error) {
	for _, def := range r.definitions {
		// Only match steps of the expected type
		if def.StepType != expectedType {
			continue
		}

		matches := def.Pattern.FindStringSubmatch(step.Text)
		if matches == nil {
			continue
		}

		// Extract parameters
		params := make(map[string]string)

		// Get parameter names from template
		paramNames := extractTemplateParams(def.Template)

		// Map captured groups to parameter names
		// matches[0] is the full match, matches[1:] are the captured groups
		for i, name := range paramNames {
			if i+1 < len(matches) {
				params[name] = matches[i+1]
			}
		}

		return &def, params, nil
	}

	return nil, nil, fmt.Errorf("no matching step definition for %q (type: %s)", step.Text, expectedType)
}

// extractTemplateParams extracts parameter names from a template string.
// E.g., "label(\"{label}\")" -> ["label"]
func extractTemplateParams(template string) []string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}

// Substitute replaces placeholders in a template with actual values.
func (r *StepRegistry) Substitute(template string, params map[string]string) string {
	result := template
	for key, value := range params {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// GenerateDatalog generates a Datalog fragment for a step.
func (r *StepRegistry) GenerateDatalog(step parse.Step, expectedType parse.StepType) (string, error) {
	def, params, err := r.Match(step, expectedType)
	if err != nil {
		return "", err
	}

	return r.Substitute(def.Template, params), nil
}
