package rulegenerator

import (
	"context"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/internal/testproviders/mock"
)

func TestGenerateRuleWithCustomRuleHead(t *testing.T) {
	// Sample schema for the test.
	type SampleSchema struct {
		Region string `mangle:"region"`
		Amount int    `mangle:"amount"`
	}

	tests := []struct {
		name              string
		opts              GeneratorOptions
		policyText        string
		mockLLMResponse   string
		expectedPrompt    string
		expectedRule      string
		expectError       bool
		expectedErrorText string
	}{
		{
			name: "Default RuleHead",
			opts: GeneratorOptions{},
			policyText: "Block transactions from the 'FR' region.",
			mockLLMResponse: `deny(Req) :- region(Req, "FR").`,
			expectedPrompt:  "The target rule head MUST match the signature: 'deny(Req)'",
			expectedRule:    `deny(Req) :- region(Req, "FR").`,
		},
		{
			name: "Custom RuleHead with multiple arguments",
			opts: GeneratorOptions{
				RuleHead: "route(Req, Target)",
			},
			policyText: "Route to 'fraud-queue' if amount > 1000.",
			mockLLMResponse: `route(Req, "fraud-queue") :- amount(Req, X), X > 1000.`,
			expectedPrompt:  "The target rule head MUST match the signature: 'route(Req, Target)'",
			expectedRule:    `route(Req, "fraud-queue") :- amount(Req, X), X > 1000.`,
		},
		{
			name:           "Empty generated rule",
			opts:           GeneratorOptions{},
			policyText:     "Any policy",
			mockLLMResponse: ``,
			expectError:    true,
			expectedErrorText: "generated rule is empty",
		},
		{
			name:           "Malformed generated rule",
			opts:           GeneratorOptions{},
			policyText:     "Any policy",
			mockLLMResponse: `deny(Req) :- .`,
			expectError:    true,
			expectedErrorText: "parsing clause failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Use the mock LLM that returns a fixed response.
			mockLLM := mock.NewLLM("test-model", tc.mockLLMResponse)

			generator, err := New(mockLLM, tc.opts)
			if err != nil {
				t.Fatalf("Failed to create generator: %v", err)
			}

			// Capture the prompt by overriding the CompleteFunc
			var capturedPrompt string
			mockLLM.CompleteFunc = func(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
				capturedPrompt = req.Prompt
				return core.LLMResponse{Text: tc.mockLLMResponse}, nil
			}

			generatedRule, err := generator.GenerateRule(context.Background(), SampleSchema{}, tc.policyText)

			if tc.expectError {
				if err == nil {
					t.Fatalf("Expected an error, but got none")
				}
				if !strings.Contains(err.Error(), tc.expectedErrorText) {
					t.Errorf("Expected error to contain '%s', but got '%s'", tc.expectedErrorText, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			if !strings.Contains(capturedPrompt, tc.expectedPrompt) {
				t.Errorf("Prompt mismatch:\nExpected to contain: %s\nGot: %s", tc.expectedPrompt, capturedPrompt)
			}

			if generatedRule != tc.expectedRule {
				t.Errorf("Rule mismatch:\nExpected: %s\nGot: %s", tc.expectedRule, generatedRule)
			}
		})
	}
}
