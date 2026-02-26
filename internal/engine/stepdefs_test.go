package engine

import (
	"testing"

	"github.com/duynguyendang/manglekit-wip/internal/engine/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepRegistry_MatchGivenStep(t *testing.T) {
	registry := NewStepRegistry()

	step := parse.Step{
		Keyword: "Given",
		Text:    `the user has "pii" label`,
	}

	def, params, err := registry.Match(step, parse.StepTypeGiven)
	require.NoError(t, err)
	assert.NotNil(t, def)
	assert.Equal(t, `label("{label}")`, def.Template)
	assert.Equal(t, "pii", params["label"])
}

func TestStepRegistry_MatchWhenStep(t *testing.T) {
	registry := NewStepRegistry()

	step := parse.Step{
		Keyword: "When",
		Text:    `calling "llm_generate"`,
	}

	def, params, err := registry.Match(step, parse.StepTypeWhen)
	require.NoError(t, err)
	assert.NotNil(t, def)
	assert.Equal(t, `action_operation(Req, "{action}")`, def.Template)
	assert.Equal(t, "llm_generate", params["action"])
}

func TestStepRegistry_MatchThenStep(t *testing.T) {
	registry := NewStepRegistry()

	step := parse.Step{
		Keyword: "Then",
		Text:    `halt with "PII leakage detected"`,
	}

	def, params, err := registry.Match(step, parse.StepTypeThen)
	require.NoError(t, err)
	assert.NotNil(t, def)
	assert.Equal(t, `halt(Req, "{reason}")`, def.Template)
	assert.Equal(t, "PII leakage detected", params["reason"])
}

func TestStepRegistry_MatchMetadataStep(t *testing.T) {
	registry := NewStepRegistry()

	step := parse.Step{
		Keyword: "Given",
		Text:    `the metadata "user_id" is "12345"`,
	}

	def, params, err := registry.Match(step, parse.StepTypeGiven)
	require.NoError(t, err)
	assert.NotNil(t, def)
	assert.Equal(t, "user_id", params["key"])
	assert.Equal(t, "12345", params["value"])
}

func TestStepRegistry_NoMatch(t *testing.T) {
	registry := NewStepRegistry()

	step := parse.Step{
		Keyword: "Given",
		Text:    `something completely unrecognized`,
	}

	_, _, err := registry.Match(step, parse.StepTypeGiven)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching step definition")
}

func TestStepRegistry_WrongType(t *testing.T) {
	registry := NewStepRegistry()

	// This is a When step pattern, but we're looking for Given
	step := parse.Step{
		Keyword: "When",
		Text:    `calling "test_action"`,
	}

	_, _, err := registry.Match(step, parse.StepTypeGiven)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching step definition")
}

func TestStepRegistry_Substitute(t *testing.T) {
	registry := NewStepRegistry()

	template := `halt(Req, "{reason}")`
	params := map[string]string{
		"reason": "Access denied",
	}

	result := registry.Substitute(template, params)
	assert.Equal(t, `halt(Req, "Access denied")`, result)
}

func TestStepRegistry_SubstituteMultiple(t *testing.T) {
	registry := NewStepRegistry()

	template := `meta("{key}", "{value}")`
	params := map[string]string{
		"key":   "user_id",
		"value": "12345",
	}

	result := registry.Substitute(template, params)
	assert.Equal(t, `meta("user_id", "12345")`, result)
}

func TestStepRegistry_GenerateDatalog(t *testing.T) {
	registry := NewStepRegistry()

	step := parse.Step{
		Keyword: "Given",
		Text:    `the user has "admin" label`,
	}

	datalog, err := registry.GenerateDatalog(step, parse.StepTypeGiven)
	require.NoError(t, err)
	assert.Equal(t, `label("admin")`, datalog)
}

func TestStepRegistry_GenerateDatalogWhen(t *testing.T) {
	registry := NewStepRegistry()

	step := parse.Step{
		Keyword: "When",
		Text:    `calling the action "delete_user"`,
	}

	datalog, err := registry.GenerateDatalog(step, parse.StepTypeWhen)
	require.NoError(t, err)
	assert.Equal(t, `action_operation(Req, "delete_user")`, datalog)
}

func TestStepRegistry_GenerateDatalogThen(t *testing.T) {
	registry := NewStepRegistry()

	step := parse.Step{
		Keyword: "Then",
		Text:    `route to "admin_service"`,
	}

	datalog, err := registry.GenerateDatalog(step, parse.StepTypeThen)
	require.NoError(t, err)
	assert.Equal(t, `route(Req, "admin_service")`, datalog)
}

func TestStepRegistry_CustomRegistration(t *testing.T) {
	registry := NewStepRegistry()

	// Register a custom step
	err := registry.Register(
		`^the system is in "([^"]+)" mode$`,
		`system_mode("{mode}")`,
		parse.StepTypeGiven,
	)
	require.NoError(t, err)

	step := parse.Step{
		Keyword: "Given",
		Text:    `the system is in "maintenance" mode`,
	}

	datalog, err := registry.GenerateDatalog(step, parse.StepTypeGiven)
	require.NoError(t, err)
	assert.Equal(t, `system_mode("maintenance")`, datalog)
}

func TestStepRegistry_InvalidPattern(t *testing.T) {
	registry := NewStepRegistry()

	// Invalid regex pattern
	err := registry.Register(
		`[invalid(regex`,
		`template`,
		parse.StepTypeGiven,
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid pattern")
}

func TestExtractTemplateParams(t *testing.T) {
	tests := []struct {
		name     string
		template string
		expected []string
	}{
		{
			name:     "single param",
			template: `label("{label}")`,
			expected: []string{"label"},
		},
		{
			name:     "multiple params",
			template: `meta("{key}", "{value}")`,
			expected: []string{"key", "value"},
		},
		{
			name:     "no params",
			template: `allow(Req)`,
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTemplateParams(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}
