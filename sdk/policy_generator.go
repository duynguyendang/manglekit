// Package sdk provides the Manglekit SDK.
// This file contains the Policy Copilot logic (formerly rulegenerator).
package sdk

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/duynguyendang/manglekit/core"
	mangleparse "github.com/google/mangle/parse"
)

// GeneratorOptions defines configuration for the rule generator.
type GeneratorOptions struct {
	// RuleHead defines the target predicate (e.g., "deny(Req)", "allow(Req)", "route(Req, Target)").
	// Default: "deny(Req)"
	RuleHead string

	// PromptTemplate is a custom Go template string.
	// If empty, use the internal DefaultPromptTemplate.
	PromptTemplate string

	// Examples are the few-shot examples inserted into the template.
	// If empty, use the internal DefaultExamples.
	Examples string
}

// DefaultPromptTemplate is the default prompt used to instruct the LLM.
const DefaultPromptTemplate = `System: You are an expert Mangle Datalog Compiler. Your task is to translate a natural language policy into a single, valid Mangle Datalog rule.

Constraints:
1.  You MUST use only the predicates defined in the "Schema Context".
2.  The target rule head MUST match the signature: '{{.RuleHead}}'.
3.  The output MUST be ONLY the raw Datalog code. Do not include markdown, explanations, or any other text.
4.  Variables in the rule body must be bound to values from the schema predicates.
5.  Use ':=' for assignment and comparison operators like '>', '<', '==' for checks.
6.  String literals must be quoted with double quotes (e.g., "UK").

Schema Context:
{{.SchemaContext}}

Few-Shot Examples:
{{.Examples}}

Now, complete the following task.

User Policy: "{{.UserPolicy}}"
Your Output:
`

// DefaultExamples provides few-shot examples for the LLM.
const DefaultExamples = `
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
deny(Req) :- region(Req, "US"), amount(Req, Amount), Amount < 50.`

// Generator uses a core.Action to translate natural language into Mangle rules.
// The action is typically an LLM (via adapters/ai.NewGenkitAction) but can be any
// core.Action that takes a string prompt and returns a string response.
type Generator struct {
	llmAction core.Action
	opts      GeneratorOptions
	template  *template.Template
}

// NewPolicyGenerator creates a new Generator with the provided core.Action and options.
// The action should be an LLM action (e.g., created via adapters/ai.NewGenkitAction).
// The action may be wrapped in a Guard for policy enforcement and tracing.
func NewPolicyGenerator(llmAction core.Action, opts GeneratorOptions) (*Generator, error) {
	if opts.RuleHead == "" {
		opts.RuleHead = "deny(Req)"
	}
	if opts.PromptTemplate == "" {
		opts.PromptTemplate = DefaultPromptTemplate
	}
	if opts.Examples == "" {
		opts.Examples = DefaultExamples
	}

	tmpl, err := template.New("policy_generator").Parse(opts.PromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template: %w", err)
	}

	return &Generator{
		llmAction: llmAction,
		opts:      opts,
		template:  tmpl,
	}, nil
}

// GenerateRule takes a sample struct (to learn the schema) and a text policy.
// It returns the raw Datalog string.
//
// This function demonstrates "In-Context Learning" by dynamically teaching the LLM
// the data schema derived from the Go struct.
//
// Dogfooding Note: This implementation uses core.Action as a universal unit of work.
// The llmAction is executed via the standard core.Action interface, which means it
// automatically benefits from any wrapping (e.g., Guard for policy checks, tracing).
//
// The process:
//  1. Schema Extraction - Use engine/reflection (engine.ToFacts) to inspect schemaSample
//  2. Prompt Construction - Build a detailed system prompt with schema and policy
//  3. Action Execution - Call g.llmAction.Execute(ctx, inputEnvelope) with the prompt
//  4. Output Processing - Unwrap the Envelope, assert payload is string
//  5. Verification - Validate the generated rule syntax using Mangle parser
func (g *Generator) GenerateRule(ctx context.Context, schemaSample any, policyText string) (string, error) {
	// Step A: Schema Extraction (The Context)
	schemaContext, err := g.extractSchema(schemaSample)
	if err != nil {
		return "", fmt.Errorf("failed to extract schema: %w", err)
	}

	// Step B: Construct System Prompt
	prompt, err := g.constructPrompt(schemaContext, policyText)
	if err != nil {
		return "", fmt.Errorf("failed to construct prompt: %w", err)
	}

	// Step C: Action Execution (Dogfooding core.Action)
	// Create an Envelope with the prompt string.
	inputEnvelope := core.NewEnvelope(prompt)
	inputEnvelope.SetMeta("schema_sample_type", fmt.Sprintf("%T", schemaSample))
	inputEnvelope.SetMeta("policy_text", policyText)

	// Execute via core.Action interface. The action may be guarded for policy/tracing.
	outputEnvelope, err := g.llmAction.Execute(ctx, inputEnvelope)
	if err != nil {
		return "", fmt.Errorf("llm action execution failed: %w", err)
	}

	// Step D: Output Processing
	// Unwrap the Envelope and assert payload is string.
	resp, ok := outputEnvelope.Payload.(string)
	if !ok {
		return "", fmt.Errorf("llm action returned non-string payload: expected string, got %T", outputEnvelope.Payload)
	}

	// Step E: Sanitize and Verify
	// Clean the output to remove common markdown formatting.
	generatedRule := sanitizeOutput(resp)

	if err := g.verifyRuleSyntax(generatedRule); err != nil {
		return "", fmt.Errorf("generated rule failed syntax check: %w. Rule: '%s'", err, generatedRule)
	}

	return generatedRule, nil
}

// extractSchema uses reflection to inspect the schemaSample and create a schema context string.
// It transforms struct fields into predicate descriptions suitable for the LLM prompt.
func (g *Generator) extractSchema(schemaSample any) (string, error) {
	val := reflect.ValueOf(schemaSample)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return "", fmt.Errorf("schemaSample must be a struct or a pointer to a struct, got %v", val.Kind())
	}
	typ := val.Type()
	var predicates []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("mangle")
		if tag == "-" {
			continue // Skip fields marked with mangle:"-"
		}
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}

		var fieldType string
		switch field.Type.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			fieldType = "IntValue"
		case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
			fieldType = "IntValue"
		case reflect.Float32, reflect.Float64:
			fieldType = "FloatValue"
		case reflect.String:
			fieldType = "StringValue"
		case reflect.Bool:
			fieldType = "BoolValue" // Represented as string "true" or "false" in Mangle
		default:
			// Skip unsupported types (structs, slices, maps, etc.)
			continue
		}
		predicates = append(predicates, fmt.Sprintf("%s(EntityID, %s)", tag, fieldType))
	}

	if len(predicates) == 0 {
		return "", fmt.Errorf("no valid predicates could be extracted from schema sample")
	}

	return "Available Predicates: " + strings.Join(predicates, ", "), nil
}

// constructPrompt builds the full prompt for the LLM.
func (g *Generator) constructPrompt(schemaContext, userPolicy string) (string, error) {
	data := struct {
		RuleHead      string
		SchemaContext string
		Examples      string
		UserPolicy    string
	}{
		RuleHead:      g.opts.RuleHead,
		SchemaContext: schemaContext,
		Examples:      g.opts.Examples,
		UserPolicy:    userPolicy,
	}

	var buf bytes.Buffer
	if err := g.template.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return buf.String(), nil
}

// sanitizeOutput cleans the LLM response by removing common markdown formatting.
func sanitizeOutput(resp string) string {
	generatedRule := strings.TrimSpace(resp)
	generatedRule = strings.TrimPrefix(generatedRule, "```datalog")
	generatedRule = strings.TrimPrefix(generatedRule, "```mangle")
	generatedRule = strings.TrimPrefix(generatedRule, "```prolog")
	generatedRule = strings.TrimPrefix(generatedRule, "```")
	generatedRule = strings.TrimSuffix(generatedRule, "```")
	generatedRule = strings.TrimSpace(generatedRule)
	return generatedRule
}

// verifyRuleSyntax checks if the generated Datalog is syntactically valid.
func (g *Generator) verifyRuleSyntax(rule string) error {
	if rule == "" {
		return fmt.Errorf("generated rule is empty")
	}

	// Parse the rule using Mangle's parser
	_, err := mangleparse.Clause(rule)
	if err != nil {
		return fmt.Errorf("parsing clause failed: %w", err)
	}

	// Verify it's a rule (has a body) and not just a fact
	if !strings.Contains(rule, ":-") {
		return fmt.Errorf("generated code is not a rule (missing ':-')")
	}

	return nil
}
