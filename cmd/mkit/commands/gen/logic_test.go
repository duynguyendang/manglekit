package gen

import (
	"context"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/sdk"
)

// MockGenerator captures the prompt and returns a dummy response
type MockGenerator struct {
	CapturedPrompt string
}

func (m *MockGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	m.CapturedPrompt = prompt
	// Return a valid JSON response with VALID Datalog.
	// We must declare 'input' or use a dummy fact if we use it,
	// OR just use standard library predicates which are pre-declared.
	// `deny` is declared in ValidatePolicySyntax.
	// Let's use a trivial valid rule: "deny(Source, Reason) :- json_bool(Source, "foo", "bar")."
	// (json_bool is in stdlib).
	return `{"datalog_content": "deny(S, \"test\") :- json_bool(S, \"foo\", \"bar\").", "explanation": "test"}`, nil
}

// Ensure MockGenerator satisfies sdk.TextGenerator
var _ sdk.TextGenerator = &MockGenerator{}

func TestGenerateWithFeedback_PromptConstruction(t *testing.T) {
	mockGen := &MockGenerator{}
	ctx := context.Background()
	userReq := "Deny users who are not VIPs"
	domainVocab := []string{"is_vip(User)", "is_admin(User)"}
	iclContent := "Decl is_foo(R)."

	_, err := GenerateWithFeedback(ctx, mockGen, userReq, domainVocab, nil, iclContent)
	if err != nil {
		t.Fatalf("GenerateWithFeedback failed: %v", err)
	}

	// Verify Combined Prompt Content
	// Since MockGenerator triggers the Fallback path in GenerateStruct,
	// it receives "SystemPrompt\n\nUser Request: UserReq".
	combinedPrompt := mockGen.CapturedPrompt

	// 1. Check for "### Domain Vocabulary"
	if !strings.Contains(combinedPrompt, "### Domain Vocabulary:") {
		t.Errorf("System prompt missing '### Domain Vocabulary:' section")
	}

	// 2. Check for injected vocab items
	if !strings.Contains(combinedPrompt, "- is_vip(User)") {
		t.Errorf("System prompt missing vocab item 'is_vip(User)'")
	}
	if !strings.Contains(combinedPrompt, "- is_admin(User)") {
		t.Errorf("System prompt missing vocab item 'is_admin(User)'")
	}

	// 2.5 Check for Telemetry
	if !strings.Contains(combinedPrompt, "### Telemetry & Compliance (MANDATORY):") {
		t.Errorf("System prompt missing '### Telemetry & Compliance (MANDATORY):' section")
	}

	// 3. Check for specific instructions
	expectedInstr1 := "Prioritize using the Domain Vocabulary over raw json_xxx predicates if available."
	if !strings.Contains(combinedPrompt, expectedInstr1) {
		t.Errorf("System prompt missing instruction: %s", expectedInstr1)
	}

	expectedInstr2 := "If mapping raw JSON, prefer creating a \"Helper Predicate\" first"
	if !strings.Contains(combinedPrompt, expectedInstr2) {
		t.Errorf("System prompt missing instruction: %s", expectedInstr2)
	}

	// Verify User Request is passed through
	if !strings.Contains(combinedPrompt, userReq) {
		t.Errorf("User prompt mismatch. Expected to contain %q, got %q", userReq, combinedPrompt)
	}

	// Verify ICL Content Injection
	if !strings.Contains(combinedPrompt, "### 4. Code Style Reference (Golden Rules)") {
		t.Errorf("System prompt missing '### 4. Code Style Reference' section")
	}
	if !strings.Contains(combinedPrompt, iclContent) {
		t.Errorf("System prompt missing injected ICL content")
	}
}
