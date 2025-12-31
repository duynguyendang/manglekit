package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFeature_Simple(t *testing.T) {
	content := `Feature: PII Protection
  Prevent PII data leakage

  Scenario: Block PII to LLM
    Given the user has "pii" label
    When calling "llm_generate"
    Then halt with "PII leakage detected"
`

	feature, err := ParseFeature(content)
	require.NoError(t, err)
	assert.Equal(t, "PII Protection", feature.Name)
	assert.Contains(t, feature.Description, "Prevent PII data leakage")
	assert.Len(t, feature.Scenarios, 1)

	scenario := feature.Scenarios[0]
	assert.Equal(t, "Block PII to LLM", scenario.Name)
	assert.Len(t, scenario.Steps, 3)

	// Check steps
	assert.Equal(t, "Given", scenario.Steps[0].Keyword)
	assert.Equal(t, `the user has "pii" label`, scenario.Steps[0].Text)

	assert.Equal(t, "When", scenario.Steps[1].Keyword)
	assert.Equal(t, `calling "llm_generate"`, scenario.Steps[1].Text)

	assert.Equal(t, "Then", scenario.Steps[2].Keyword)
	assert.Equal(t, `halt with "PII leakage detected"`, scenario.Steps[2].Text)
}

func TestParseFeature_MultipleScenarios(t *testing.T) {
	content := `Feature: Access Control

  Scenario: Admin access
    Given the user has "admin" label
    When calling "delete_user"
    Then route to "admin_service"

  Scenario: User denied
    Given the user has "user" label
    When calling "delete_user"
    Then halt with "Insufficient permissions"
`

	feature, err := ParseFeature(content)
	require.NoError(t, err)
	assert.Equal(t, "Access Control", feature.Name)
	assert.Len(t, feature.Scenarios, 2)

	// First scenario
	assert.Equal(t, "Admin access", feature.Scenarios[0].Name)
	assert.Len(t, feature.Scenarios[0].Steps, 3)

	// Second scenario
	assert.Equal(t, "User denied", feature.Scenarios[1].Name)
	assert.Len(t, feature.Scenarios[1].Steps, 3)
}

func TestParseFeature_WithAndSteps(t *testing.T) {
	content := `Feature: Complex Policy

  Scenario: Multiple conditions
    Given the user has "pii" label
    And the entity is labeled "sensitive"
    When calling "llm_generate"
    Then halt with "Access denied"
`

	feature, err := ParseFeature(content)
	require.NoError(t, err)
	assert.Len(t, feature.Scenarios, 1)

	scenario := feature.Scenarios[0]
	assert.Len(t, scenario.Steps, 4)

	assert.Equal(t, "Given", scenario.Steps[0].Keyword)
	assert.Equal(t, "And", scenario.Steps[1].Keyword)
	assert.Equal(t, "When", scenario.Steps[2].Keyword)
	assert.Equal(t, "Then", scenario.Steps[3].Keyword)
}

func TestParseFeature_WithComments(t *testing.T) {
	content := `# This is a comment
Feature: Test Feature
  # Another comment
  
  Scenario: Test Scenario
    # Step comment
    Given the user has "admin" label
    When calling "test_action"
    Then route to "service"
`

	feature, err := ParseFeature(content)
	require.NoError(t, err)
	assert.Equal(t, "Test Feature", feature.Name)
	assert.Len(t, feature.Scenarios, 1)
	assert.Len(t, feature.Scenarios[0].Steps, 3)
}

func TestParseFeature_NoFeature(t *testing.T) {
	content := `Scenario: Orphan Scenario
    Given something
`

	_, err := ParseFeature(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Feature declaration found")
}

func TestParseFeature_StepOutsideScenario(t *testing.T) {
	content := `Feature: Bad Feature

  Given this is outside a scenario
  When something happens
`

	_, err := ParseFeature(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "step found outside of scenario")
}

func TestParseFeature_AndWithoutPrecedingStep(t *testing.T) {
	content := `Feature: Bad Feature

  Scenario: Bad Scenario
    And this has no preceding step
`

	_, err := ParseFeature(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "'And' step without a preceding")
}

func TestStep_ExtractQuotedStrings(t *testing.T) {
	step := &Step{
		Text: `the user has "pii" label and "admin" role`,
	}

	quoted := step.ExtractQuotedStrings()
	assert.Equal(t, []string{"pii", "admin"}, quoted)
}

func TestStep_ExtractQuotedStrings_Empty(t *testing.T) {
	step := &Step{
		Text: `the user has no quotes`,
	}

	quoted := step.ExtractQuotedStrings()
	assert.Empty(t, quoted)
}

func TestStep_ExtractCurlyBraceParams(t *testing.T) {
	step := &Step{
		Text: `the {entity} has {attribute} value`,
	}

	params := step.ExtractCurlyBraceParams()
	assert.Equal(t, []string{"entity", "attribute"}, params)
}

func TestParseFeature_EmptyFile(t *testing.T) {
	content := ``

	_, err := ParseFeature(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Feature declaration found")
}

func TestParseFeature_OnlyComments(t *testing.T) {
	content := `# Just comments
# Nothing else
`

	_, err := ParseFeature(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no Feature declaration found")
}
