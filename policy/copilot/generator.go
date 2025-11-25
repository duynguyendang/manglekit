package copilot

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/google/mangle/parse"
)

// Generator uses an LLM to translate natural language into Mangle rules.
type Generator struct {
	llm core.LLMClient
}

// New creates a new Generator.
func New(llm core.LLMClient) *Generator {
	return &Generator{llm: llm}
}

// GenerateRule takes a sample struct (to learn the schema) and a text policy.
// It returns the raw Datalog string.
// This function demonstrates "In-Context Learning" by dynamically teaching the LLM
// the data schema derived from the Go struct.
func (g *Generator) GenerateRule(ctx context.Context, schemaSample any, policyText string) (string, error) {
	// Step A: Schema Extraction (The Context)
	schemaContext, err := g.extractSchema(schemaSample)
	if err != nil {
		return "", fmt.Errorf("failed to extract schema: %w", err)
	}

	// Step B: Construct System Prompt
	prompt := g.constructPrompt(schemaContext, policyText)

	// Step C: LLM Execution
	req := core.LLMRequest{
		Prompt: prompt,
	}
	resp, err := g.llm.Complete(ctx, req)
	if err != nil {
		return "", fmt.Errorf("llm completion failed: %w", err)
	}
	generatedRule := strings.TrimSpace(resp.Text)

	// Step D: Syntax Verification (Sanity Check)
	if err := g.verifyRuleSyntax(generatedRule); err != nil {
		return "", fmt.Errorf("generated rule failed syntax check: %w. Rule: '%s'", err, generatedRule)
	}

	return generatedRule, nil
}

// extractSchema uses reflection to inspect the schemaSample and create a schema context string.
func (g *Generator) extractSchema(schemaSample any) (string, error) {
	val := reflect.ValueOf(schemaSample)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return "", fmt.Errorf("schemaSample must be a struct or a pointer to a struct")
	}
	typ := val.Type()
	var predicates []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("mangle")
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		var fieldType string
		switch field.Type.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			fieldType = "IntValue"
		case reflect.String:
			fieldType = "StringValue"
		case reflect.Bool:
			fieldType = "BoolValue" // Represented as string "true" or "false" in Mangle
		default:
			// Skip unsupported types
			continue
		}
		predicates = append(predicates, fmt.Sprintf("%s(EntityID, %s)", tag, fieldType))
	}

	return "Available Predicates: " + strings.Join(predicates, ", "), nil
}

// constructPrompt builds the full prompt for the LLM.
func (g *Generator) constructPrompt(schemaContext, policyText string) string {
	// Using a more structured prompt to guide the LLM better.
	return fmt.Sprintf(`System: You are an expert Mangle Datalog Compiler. Your task is to translate a natural language policy into a single, valid Mangle Datalog rule.

Constraints:
1.  You MUST use only the predicates defined in the "Schema Context".
2.  The target rule head MUST be 'deny(Req)'.
3.  The output MUST be ONLY the raw Datalog code. Do not include markdown, explanations, or any other text.
4.  Variables in the rule body must be bound to the 'Req' variable from the head.
5.  Use ':=' for assignment and comparison operators like '>', '<', '==' for checks.

Schema Context:
%s

Few-Shot Examples:

Example 1:
- User Policy: "Block if amount > 1000"
- Your Output:
deny(Req) :- amount(Req, X), X > 1000.

Example 2:
- User Policy: "Block transactions from the 'FR' region."
- Your Output:
deny(Req) :- region(Req, "FR").

Example 3:
- User Policy: "Deny if region is 'US' and the amount is less than 50"
- Your Output:
deny(Req) :- region(Req, "US"), amount(Req, Amount), Amount < 50.

Now, complete the following task.

User Policy: "%s"
Your Output:
`, schemaContext, policyText)
}

// verifyRuleSyntax checks if the generated Datalog is syntactically valid.
func (g *Generator) verifyRuleSyntax(rule string) error {
	if rule == "" {
		return fmt.Errorf("generated rule is empty")
	}
	// A temporary engine to parse and validate the rule.
	// We don't need to initialize it with any facts or decls for syntax check.
	_, err := parse.Clause(rule)
	if err != nil {
		return fmt.Errorf("parsing clause failed: %w", err)
	}
	// A simple check to ensure it's a rule and not just a fact.
	if !strings.Contains(rule, ":-") {
		return fmt.Errorf("generated code is not a rule (missing ':-')")
	}
	return nil
}
