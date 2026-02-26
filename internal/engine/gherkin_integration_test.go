package engine

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyEngine_LoadGherkinPolicy_Simple(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	gherkin := `Feature: PII Protection

  Scenario: Block PII to LLM
    Given the user has "pii" label
    When calling "llm_generate"
    Then halt with "PII leakage detected"
`

	err = engine.LoadGherkinPolicy(context.Background(), gherkin)
	require.NoError(t, err)

	// Test: Create an envelope with PII label
	input := core.NewEnvelope(map[string]string{"test": "data"})
	input.AddLabel("pii")

	// Call Assess with llm_generate action
	assessErr := engine.Assess(context.Background(), core.ActionMetadata{Name: "llm_generate"}, input)

	// Should be blocked
	require.Error(t, assessErr)
	assert.True(t, core.IsAlignmentError(assessErr))

	// Check the error message
	var alignErr *core.AlignmentError
	require.True(t, core.IsAlignmentError(assessErr))
	if core.IsAlignmentError(assessErr) {
		alignErr = assessErr.(*core.AlignmentError)
		assert.Equal(t, "PII leakage detected", alignErr.Message)
	}
}

func TestPolicyEngine_LoadGherkinPolicy_Allow(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	gherkin := `Feature: PII Protection

  Scenario: Block PII to LLM
    Given the user has "pii" label
    When calling "llm_generate"
    Then halt with "PII leakage detected"
`

	err = engine.LoadGherkinPolicy(context.Background(), gherkin)
	require.NoError(t, err)

	// Test: Create an envelope WITHOUT PII label
	input := core.NewEnvelope(map[string]string{"test": "data"})
	input.AddLabel("public")

	// Call Assess with llm_generate action
	assessErr := engine.Assess(context.Background(), core.ActionMetadata{Name: "llm_generate"}, input)

	// Should be allowed (no PII label)
	assert.NoError(t, assessErr)
}

func TestPolicyEngine_LoadGherkinPolicy_MultipleScenarios(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	gherkin := `Feature: Access Control

  Scenario: Admin can delete
    Given the user has "admin" label
    When calling "delete_user"
    Then route to "admin_service"

  Scenario: User cannot delete
    Given the user has "user" label
    When calling "delete_user"
    Then halt with "Insufficient permissions"
`

	err = engine.LoadGherkinPolicy(context.Background(), gherkin)
	require.NoError(t, err)

	// Test 1: Admin user
	adminInput := core.NewEnvelope(map[string]string{"user": "admin"})
	adminInput.AddLabel("admin")

	assessErr := engine.Assess(context.Background(), core.ActionMetadata{Name: "delete_user"}, adminInput)
	assert.NoError(t, assessErr) // Admin is routed, not halted

	// Test 2: Regular user
	userInput := core.NewEnvelope(map[string]string{"user": "regular"})
	userInput.AddLabel("user")

	assessErr = engine.Assess(context.Background(), core.ActionMetadata{Name: "delete_user"}, userInput)
	require.Error(t, assessErr)
	assert.True(t, core.IsAlignmentError(assessErr))

	var alignErr *core.AlignmentError
	if core.IsAlignmentError(assessErr) {
		alignErr = assessErr.(*core.AlignmentError)
		assert.Equal(t, "Insufficient permissions", alignErr.Message)
	}
}

func TestPolicyEngine_LoadGherkinPolicy_ComparisonWithDatalog(t *testing.T) {
	// Test that Gherkin and raw Datalog produce identical behavior

	// Engine 1: Load Gherkin
	engine1, err := New()
	require.NoError(t, err)

	gherkin := `Feature: Test Policy

  Scenario: Block sensitive action
    Given the user has "sensitive" label
    When calling "process_data"
    Then halt with "Sensitive data detected"
`

	err = engine1.LoadGherkinPolicy(context.Background(), gherkin)
	require.NoError(t, err)

	// Engine 2: Load equivalent Datalog
	engine2, err := New()
	require.NoError(t, err)

	datalog := `halt(Req, "Sensitive data detected") :-
    action_operation(Req, "process_data"),
    label("sensitive").`

	err = engine2.LoadPolicy(context.Background(), datalog)
	require.NoError(t, err)

	// Test both engines with same input
	input := core.NewEnvelope(map[string]string{"data": "test"})
	input.AddLabel("sensitive")

	err1 := engine1.Assess(context.Background(), core.ActionMetadata{Name: "process_data"}, input)
	err2 := engine2.Assess(context.Background(), core.ActionMetadata{Name: "process_data"}, input)

	// Both should produce the same result
	assert.Equal(t, core.IsAlignmentError(err1), core.IsAlignmentError(err2))

	if core.IsAlignmentError(err1) && core.IsAlignmentError(err2) {
		alignErr1 := err1.(*core.AlignmentError)
		alignErr2 := err2.(*core.AlignmentError)
		assert.Equal(t, alignErr1.Message, alignErr2.Message)
	}
}

func TestPolicyEngine_LoadGherkinPolicy_InvalidGherkin(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	gherkin := `This is not valid Gherkin`

	err = engine.LoadGherkinPolicy(context.Background(), gherkin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to compile Gherkin policy")
}

func TestPolicyEngine_LoadGherkinPolicy_Empty(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	err = engine.LoadGherkinPolicy(context.Background(), "")
	assert.NoError(t, err) // Empty content should not error
}

func TestPolicyEngine_LoadGherkinPolicy_MetadataCheck(t *testing.T) {
	engine, err := New()
	require.NoError(t, err)

	gherkin := `Feature: User Verification

  Scenario: Verify user ID
    Given the metadata "user_id" is "12345"
    When calling "sensitive_action"
    Then halt with "Invalid user"
`

	err = engine.LoadGherkinPolicy(context.Background(), gherkin)
	require.NoError(t, err)

	// Test with matching metadata
	input := core.NewEnvelope(map[string]string{"action": "test"})
	input.SetMeta("user_id", "12345")

	assessErr := engine.Assess(context.Background(), core.ActionMetadata{Name: "sensitive_action"}, input)
	require.Error(t, assessErr)
	assert.True(t, core.IsAlignmentError(assessErr))

	// Test with different metadata
	input2 := core.NewEnvelope(map[string]string{"action": "test"})
	input2.SetMeta("user_id", "99999")

	assessErr2 := engine.Assess(context.Background(), core.ActionMetadata{Name: "sensitive_action"}, input2)
	assert.NoError(t, assessErr2) // Different user_id, should not match
}
