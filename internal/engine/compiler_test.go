package engine

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGherkinCompiler_SimpleScenario(t *testing.T) {
	content := `Feature: PII Protection

  Scenario: Block PII to LLM
    Given the user has "pii" label
    When calling "llm_generate"
    Then halt with "PII leakage detected"
`

	compiler := NewGherkinCompiler()
	datalog, err := compiler.CompileFromString(content)
	require.NoError(t, err)

	// Check that the generated Datalog contains expected elements
	assert.Contains(t, datalog, "% Generated from Feature: PII Protection")
	assert.Contains(t, datalog, "% Scenario: Block PII to LLM")
	assert.Contains(t, datalog, `halt(Req, "PII leakage detected")`)
	assert.Contains(t, datalog, `action_operation(Req, "llm_generate")`)
	assert.Contains(t, datalog, `label("pii")`)

	// Check structure: head :- body.
	assert.Contains(t, datalog, ":-")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(datalog), "."))
}

func TestGherkinCompiler_MultipleScenarios(t *testing.T) {
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

	compiler := NewGherkinCompiler()
	datalog, err := compiler.CompileFromString(content)
	require.NoError(t, err)

	// Should have two scenarios
	assert.Contains(t, datalog, "% Scenario: Admin access")
	assert.Contains(t, datalog, "% Scenario: User denied")

	// Check both rules
	assert.Contains(t, datalog, `route(Req, "admin_service")`)
	assert.Contains(t, datalog, `halt(Req, "Insufficient permissions")`)
	assert.Contains(t, datalog, `label("admin")`)
	assert.Contains(t, datalog, `label("user")`)
}

func TestGherkinCompiler_WithAndSteps(t *testing.T) {
	content := `Feature: Complex Policy

  Scenario: Multiple conditions
    Given the user has "pii" label
    And the entity is labeled "sensitive"
    When calling "llm_generate"
    Then halt with "Access denied"
`

	compiler := NewGherkinCompiler()
	datalog, err := compiler.CompileFromString(content)
	require.NoError(t, err)

	// Both Given conditions should be in the body
	assert.Contains(t, datalog, `label("pii")`)
	assert.Contains(t, datalog, `label("sensitive")`)
	assert.Contains(t, datalog, `halt(Req, "Access denied")`)
}

func TestGherkinCompiler_OnlyThenStep(t *testing.T) {
	content := `Feature: Simple Fact

  Scenario: Always allow
    Then allow the request
`

	compiler := NewGherkinCompiler()
	datalog, err := compiler.CompileFromString(content)
	require.NoError(t, err)

	// Should be a fact (no :-)
	assert.Contains(t, datalog, `allow(Req).`)
	assert.NotContains(t, datalog, ":-")
}

func TestGherkinCompiler_NoThenStep(t *testing.T) {
	content := `Feature: Bad Feature

  Scenario: Missing Then
    Given the user has "admin" label
    When calling "test"
`

	compiler := NewGherkinCompiler()
	_, err := compiler.CompileFromString(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have at least one Then step")
}

func TestGherkinCompiler_EmptyScenario(t *testing.T) {
	content := `Feature: Bad Feature

  Scenario: No steps
`

	compiler := NewGherkinCompiler()
	_, err := compiler.CompileFromString(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scenario has no steps")
}

func TestGherkinCompiler_UnrecognizedStep(t *testing.T) {
	content := `Feature: Bad Feature

  Scenario: Unknown step
    Given something unrecognized
    Then halt with "error"
`

	compiler := NewGherkinCompiler()
	_, err := compiler.CompileFromString(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching step definition")
}

func TestGherkinCompiler_MetadataSteps(t *testing.T) {
	content := `Feature: Metadata Check

  Scenario: Check user ID
    Given the metadata "user_id" is "12345"
    When calling "sensitive_action"
    Then route to "verified_service"
`

	compiler := NewGherkinCompiler()
	datalog, err := compiler.CompileFromString(content)
	require.NoError(t, err)

	assert.Contains(t, datalog, `meta("user_id", "12345")`)
	assert.Contains(t, datalog, `action_operation(Req, "sensitive_action")`)
	assert.Contains(t, datalog, `route(Req, "verified_service")`)
}

func TestGherkinCompiler_RealWorldExample(t *testing.T) {
	content := `Feature: Data Governance
  Prevent sensitive data from being sent to external services

  Scenario: Block PII to external LLM
    Given the user has "pii" label
    When calling "llm_generate"
    Then halt with "PII leakage detected"

  Scenario: Allow public data to LLM
    Given the entity is labeled "public"
    When calling "llm_generate"
    Then route to "llm_service"
`

	compiler := NewGherkinCompiler()
	datalog, err := compiler.CompileFromString(content)
	require.NoError(t, err)

	// Verify structure
	assert.Contains(t, datalog, "% Generated from Feature: Data Governance")
	assert.Contains(t, datalog, "% Prevent sensitive data")

	// Verify both scenarios compiled
	lines := strings.Split(datalog, "\n")
	scenarioCount := 0
	for _, line := range lines {
		if strings.Contains(line, "% Scenario:") {
			scenarioCount++
		}
	}
	assert.Equal(t, 2, scenarioCount)

	// Verify the rules are valid Datalog
	assert.Contains(t, datalog, `halt(Req, "PII leakage detected") :-`)
	assert.Contains(t, datalog, `route(Req, "llm_service") :-`)
}

func TestGherkinCompiler_ParseError(t *testing.T) {
	content := `This is not valid Gherkin`

	compiler := NewGherkinCompiler()
	_, err := compiler.CompileFromString(content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse error")
}
