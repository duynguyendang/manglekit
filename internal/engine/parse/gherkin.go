package parse

import (
	"fmt"
	"regexp"
	"strings"
)

// Feature represents a Gherkin feature file.
type Feature struct {
	Name        string
	Description string
	Scenarios   []Scenario
}

// Scenario represents a single test scenario within a feature.
type Scenario struct {
	Name  string
	Steps []Step
}

// Step represents a single step (Given/When/Then) in a scenario.
type Step struct {
	Keyword string            // Given, When, Then, And, But
	Text    string            // The full step text
	Args    map[string]string // Extracted parameters (populated by step matcher)
}

// StepType represents the type of step.
type StepType string

const (
	StepTypeGiven StepType = "Given"
	StepTypeWhen  StepType = "When"
	StepTypeThen  StepType = "Then"
	StepTypeAnd   StepType = "And"
	StepTypeBut   StepType = "But"
)

// ParseFeature parses a Gherkin feature file content into a Feature struct.
func ParseFeature(content string) (*Feature, error) {
	lines := strings.Split(content, "\n")
	feature := &Feature{}

	var currentScenario *Scenario
	var lastStepType StepType

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse Feature
		if strings.HasPrefix(line, "Feature:") {
			feature.Name = strings.TrimSpace(strings.TrimPrefix(line, "Feature:"))
			// Collect description lines until we hit a Scenario
			i++
			for i < len(lines) {
				descLine := strings.TrimSpace(lines[i])
				if descLine == "" || strings.HasPrefix(descLine, "#") {
					i++
					continue
				}
				if strings.HasPrefix(descLine, "Scenario:") {
					i-- // Back up so we process this line next
					break
				}
				// Check if this looks like a step keyword - if so, it's an error
				if isStepLine(descLine) {
					return nil, fmt.Errorf("line %d: step found outside of scenario", i+1)
				}
				if feature.Description != "" {
					feature.Description += " "
				}
				feature.Description += descLine
				i++
			}
			continue
		}

		// Parse Scenario
		if strings.HasPrefix(line, "Scenario:") {
			if currentScenario != nil {
				feature.Scenarios = append(feature.Scenarios, *currentScenario)
			}
			currentScenario = &Scenario{
				Name:  strings.TrimSpace(strings.TrimPrefix(line, "Scenario:")),
				Steps: []Step{},
			}
			lastStepType = "" // Reset for new scenario
			continue
		}

		// Parse Steps
		step, stepType, err := parseStep(line, lastStepType)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		if step != nil {
			if currentScenario == nil {
				return nil, fmt.Errorf("line %d: step found outside of scenario", i+1)
			}
			currentScenario.Steps = append(currentScenario.Steps, *step)
			if stepType != StepTypeAnd && stepType != StepTypeBut {
				lastStepType = stepType
			}
		}
	}

	// Add the last scenario
	if currentScenario != nil {
		feature.Scenarios = append(feature.Scenarios, *currentScenario)
	}

	if feature.Name == "" {
		return nil, fmt.Errorf("no Feature declaration found")
	}

	return feature, nil
}

// parseStep parses a single step line.
// Returns the step, its type, and any error.
func parseStep(line string, lastStepType StepType) (*Step, StepType, error) {
	keywords := []string{"Given", "When", "Then", "And", "But"}

	for _, keyword := range keywords {
		if strings.HasPrefix(line, keyword+" ") || strings.HasPrefix(line, keyword+"\t") {
			text := strings.TrimSpace(strings.TrimPrefix(line, keyword))
			stepType := StepType(keyword)

			// Handle And/But - they inherit the type from the previous step
			actualType := stepType
			if stepType == StepTypeAnd || stepType == StepTypeBut {
				if lastStepType == "" {
					return nil, "", fmt.Errorf("'%s' step without a preceding Given/When/Then", keyword)
				}
				actualType = lastStepType
			}

			return &Step{
				Keyword: keyword,
				Text:    text,
				Args:    make(map[string]string),
			}, actualType, nil
		}
	}

	// Not a step line - might be a description or other content
	return nil, "", nil
}

// isStepLine checks if a line starts with a step keyword.
func isStepLine(line string) bool {
	keywords := []string{"Given", "When", "Then", "And", "But"}
	for _, keyword := range keywords {
		if strings.HasPrefix(line, keyword+" ") || strings.HasPrefix(line, keyword+"\t") {
			return true
		}
	}
	return false
}

// NormalizeStepType returns the normalized step type (Given/When/Then) for And/But steps.
func (s *Step) NormalizeStepType(lastType StepType) StepType {
	if s.Keyword == "And" || s.Keyword == "But" {
		return lastType
	}
	return StepType(s.Keyword)
}

// ExtractQuotedStrings extracts all quoted strings from the step text.
// This is a helper for step matchers to extract parameters.
func (s *Step) ExtractQuotedStrings() []string {
	re := regexp.MustCompile(`"([^"]*)"`)
	matches := re.FindAllStringSubmatch(s.Text, -1)

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, match[1])
		}
	}
	return result
}

// ExtractCurlyBraceParams extracts parameters in {param} format.
func (s *Step) ExtractCurlyBraceParams() []string {
	re := regexp.MustCompile(`\{([^}]+)\}`)
	matches := re.FindAllStringSubmatch(s.Text, -1)

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, match[1])
		}
	}
	return result
}
