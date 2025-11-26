package rulegenerator

import (
	"context"
	"strings"
	"testing"
)

// mockLLM implements ai.TextGenerator for testing
type mockLLM struct {
	response       string
	capturedPrompt string
}

func (m *mockLLM) Complete(ctx context.Context, prompt string) (string, error) {
	m.capturedPrompt = prompt
	return m.response, nil
}

func TestGenerateRule(t *testing.T) {
	// Sample schema for testing
	type Transaction struct {
		Amount int    `mangle:"amount"`
		Region string `mangle:"region"`
	}

	tests := []struct {
		name              string
		opts              GeneratorOptions
		policyText        string
		mockResponse      string
		expectedInPrompt  string
		expectedRule      string
		expectError       bool
		expectedErrorText string
	}{
		{
			name:             "Default deny rule head",
			opts:             GeneratorOptions{},
			policyText:       "Block transactions from the 'FR' region.",
			mockResponse:     `deny(Req) :- region(Req, "FR").`,
			expectedInPrompt: "deny(Req)",
			expectedRule:     `deny(Req) :- region(Req, "FR").`,
		},
		{
			name: "Custom allow rule head",
			opts: GeneratorOptions{
				RuleHead: "allow(Req)",
			},
			policyText:       "Allow if amount < 100",
			mockResponse:     `allow(Req) :- amount(Req, X), X < 100.`,
			expectedInPrompt: "allow(Req)",
			expectedRule:     `allow(Req) :- amount(Req, X), X < 100.`,
		},
		{
			name: "Custom routing rule head with multiple args",
			opts: GeneratorOptions{
				RuleHead: "route(Req, Target)",
			},
			policyText:       "Route to 'fraud-queue' if amount > 1000.",
			mockResponse:     `route(Req, "fraud-queue") :- amount(Req, X), X > 1000.`,
			expectedInPrompt: "route(Req, Target)",
			expectedRule:     `route(Req, "fraud-queue") :- amount(Req, X), X > 1000.`,
		},
		{
			name:              "Empty generated rule",
			opts:              GeneratorOptions{},
			policyText:        "Any policy",
			mockResponse:      ``,
			expectError:       true,
			expectedErrorText: "generated rule is empty",
		},
		{
			name:              "Malformed generated rule - empty body",
			opts:              GeneratorOptions{},
			policyText:        "Any policy",
			mockResponse:      `deny(Req) :- .`,
			expectError:       true,
			expectedErrorText: "parsing clause failed",
		},
		{
			name:              "Fact instead of rule",
			opts:              GeneratorOptions{},
			policyText:        "Any policy",
			mockResponse:      `deny("something").`,
			expectError:       true,
			expectedErrorText: "not a rule",
		},
		{
			name:         "Markdown code block cleanup",
			opts:         GeneratorOptions{},
			policyText:   "Block high amounts",
			mockResponse: "```datalog\ndeny(Req) :- amount(Req, X), X > 500.\n```",
			expectedRule: `deny(Req) :- amount(Req, X), X > 500.`,
		},
		// Additional syntax error test cases
		{
			name:              "Missing period at end",
			opts:              GeneratorOptions{},
			policyText:        "Block if amount > 100",
			mockResponse:      `deny(Req) :- amount(Req, X), X > 100`,
			expectError:       true,
			expectedErrorText: "parsing clause failed",
		},
		{
			name:              "Unclosed parenthesis",
			opts:              GeneratorOptions{},
			policyText:        "Block if amount > 100",
			mockResponse:      `deny(Req :- amount(Req, X), X > 100.`,
			expectError:       true,
			expectedErrorText: "parsing clause failed",
		},
		{
			name:              "Invalid operator",
			opts:              GeneratorOptions{},
			policyText:        "Block if amount > 100",
			mockResponse:      `deny(Req) :- amount(Req, X), X >> 100.`,
			expectError:       true,
			expectedErrorText: "parsing clause failed",
		},
		{
			name:         "Unquoted string literal",
			opts:         GeneratorOptions{},
			policyText:   "Block UK transactions",
			mockResponse: `deny(Req) :- region(Req, UK).`,
			expectError:  false, // UK is treated as a variable, not an error
			expectedRule: `deny(Req) :- region(Req, UK).`,
		},
		{
			name:              "Missing comma between predicates",
			opts:              GeneratorOptions{},
			policyText:        "Complex rule",
			mockResponse:      `deny(Req) :- amount(Req, X) region(Req, "UK").`,
			expectError:       true,
			expectedErrorText: "parsing clause failed",
		},
		{
			name:              "Double colon instead of rule arrow",
			opts:              GeneratorOptions{},
			policyText:        "Block high amounts",
			mockResponse:      `deny(Req) :: amount(Req, X), X > 100.`,
			expectError:       true,
			expectedErrorText: "parsing clause failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			llm := &mockLLM{response: tc.mockResponse}

			generator, err := New(llm, tc.opts)
			if err != nil {
				t.Fatalf("Failed to create generator: %v", err)
			}

			generatedRule, err := generator.GenerateRule(context.Background(), Transaction{}, tc.policyText)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected an error, but got none. Generated rule: %s", generatedRule)
				}
				if !strings.Contains(err.Error(), tc.expectedErrorText) {
					t.Errorf("Expected error to contain '%s', but got '%s'", tc.expectedErrorText, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			if tc.expectedInPrompt != "" && !strings.Contains(llm.capturedPrompt, tc.expectedInPrompt) {
				t.Errorf("Prompt should contain '%s', got: %s", tc.expectedInPrompt, llm.capturedPrompt)
			}

			if generatedRule != tc.expectedRule {
				t.Errorf("Rule mismatch:\nExpected: %s\nGot: %s", tc.expectedRule, generatedRule)
			}
		})
	}
}

func TestExtractSchema(t *testing.T) {
	type FullSchema struct {
		Amount  int     `mangle:"amount"`
		Region  string  `mangle:"region"`
		Active  bool    `mangle:"active"`
		Rate    float64 `mangle:"rate"`
		Ignored string  `mangle:"-"`
		private int     // unexported, should be skipped
	}

	llm := &mockLLM{response: `deny(Req) :- amount(Req, X), X > 0.`}
	gen, err := New(llm, GeneratorOptions{})
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	schema, err := gen.extractSchema(FullSchema{})
	if err != nil {
		t.Fatalf("extractSchema failed: %v", err)
	}

	// Check that expected predicates are present
	expectedPredicates := []string{
		"amount(EntityID, IntValue)",
		"region(EntityID, StringValue)",
		"active(EntityID, BoolValue)",
		"rate(EntityID, FloatValue)",
	}

	for _, pred := range expectedPredicates {
		if !strings.Contains(schema, pred) {
			t.Errorf("Schema should contain '%s', got: %s", pred, schema)
		}
	}

	// Check that ignored field is not present
	if strings.Contains(schema, "Ignored") || strings.Contains(schema, "ignored") {
		t.Errorf("Schema should not contain 'Ignored' field, got: %s", schema)
	}
}

func TestExtractSchema_InvalidInput(t *testing.T) {
	llm := &mockLLM{response: ""}
	gen, err := New(llm, GeneratorOptions{})
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Test with non-struct
	_, err = gen.extractSchema("not a struct")
	if err == nil {
		t.Error("Expected error for non-struct input")
	}

	// Test with struct that has no valid fields
	type EmptySchema struct {
		private int // unexported
	}
	_, err = gen.extractSchema(EmptySchema{})
	if err == nil {
		t.Error("Expected error for struct with no valid predicates")
	}
}
