package gen

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/gen/inductor"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
	"github.com/duynguyendang/manglekit/internal/engine/resources"
)

// GeneratedPolicy represents the structured response from the LLM.
type GeneratedPolicy struct {
	// DatalogContent contains the raw .dl code.
	DatalogContent string `json:"datalog_content"`
	// Explanation provides a human-readable summary of the logic.
	Explanation string `json:"explanation"`
}

// ValidatePolicySyntax checks if the generated Datalog code is valid using the Mangle engine.
func ValidatePolicySyntax(datalog, schemaDeclarations string) error {
	// Initialize a runtime.
	eng := engine.NewMangleRuntime()

	// Prepend standard declarations to the validator so usages of std lib don't fail.
	// StdLib already includes deny declarations (both deny/1 and deny/2).
	stdDecls := resources.StdLib()
	fullProgram := stdDecls + "\n" + schemaDeclarations + "\n" + datalog

	// Attempt to parse and compile the policy
	return eng.LoadFromSource(fullProgram)
}

// GenerateWithFeedback orchestrates the Teacher-Student protocol.
func GenerateWithFeedback(ctx context.Context, gen core.TextGenerator, userReq string, domainVocab []string, schema *inductor.SchemaHint, iclContent string) (*GeneratedPolicy, error) {
	// 1. Construct System Prompt
	factsList := ""
	for _, f := range domainVocab {
		factsList += "- " + f + "\n"
	}

	autoVocab := ""
	if schema != nil {
		if schema.FileType == "graph" {
			// For Graph: "Decl is_vip(S, O)."
			// We format declarations as a list
			decls := strings.Join(schema.Declarations, "\n")
			autoVocab = fmt.Sprintf(`
### Auto-Detected Vocabulary (Graph):
The data uses these predicates. Use them directly:
%s
`, decls)
		} else if schema.FileType == "json" {
			// For JSON: "amount (number)", "desc (string)"
			keys := strings.Join(schema.JsonKeys, "\n")
			autoVocab = fmt.Sprintf(`
### Auto-Detected JSON Structure (Path -> Type):
%s
### Handling Nested & Array Paths:
1. **Dot Notation** ("deployment.replicas"): Use 'json_link' to traverse objects.
   json_link(Root, "deployment", DeployNode), json_num(DeployNode, "replicas", Val).

2. **Object Arrays** ("servers[].ip"): Use 'json_link' to list, then 'json_link' with '_' (wildcard) to iterate items.
   json_link(Root, "servers", List), json_link(List, _, Server), json_str(Server, "ip", IP).

3. **Primitive Arrays** ("env_vars (array of string)"): Use 'json_link' to list, then 'json_str' with '_' to iterate values.
   json_link(Root, "env_vars", List), json_str(List, _, "TargetValue").
`, keys)
		}
	}

	systemPrompt := fmt.Sprintf(`You are a Senior Knowledge Engineer specializing in Google Mangle Datalog.
Your task is to translate natural language requirements into strict, compilable Datalog rules.

### Standard Library (Always Available):
- json_num(Source, Key, Value)  // Int/Float fields
- json_str(Source, Key, Value)  // String fields
- json_bool(Source, Key, Value) // Boolean fields
- json_link(Parent, Key, Child) // Nested objects
- deny(Source, Reason)          // Main policy output (Do NOT redeclare this)

### Telemetry & Compliance (MANDATORY):
1. You MUST declare a violation rule predicate: Decl violation_rule(Entity, RuleID).
2. For EVERY 'deny' rule you create, you MUST create a corresponding 'violation_rule'.
   - The RuleID must be UPPERCASE_SNAKE_CASE (e.g., "COST_LIMIT_EXCEEDED").
   - It captures the same conditions as the deny rule.

Example:
   deny(Req, "Cost too high") :- exceeds_cost(Req).
   violation_rule(Req, "COST_LIMIT_CHECK_01") :- exceeds_cost(Req).

### Domain Vocabulary:
%s
%s
### 4. Code Style Reference (Golden Rules)
The following are verified Manglekit Datalog examples.
Pay close attention to how 'json_link' is used for nested objects and how predicates are declared.

--- BEGIN REFERENCE ---
%s
--- END REFERENCE ---

### Syntax Rules:
1. Use 'Decl name(Arg1, Arg2, ...).' to declare predicates. 
   - CRITICAL: Do NOT use '.Decl', '.decl', or 'decl'. MUST be 'Decl' (Case-sensitive, no dot).
   - Args MUST start with Uppercase (e.g., Decl amount(Source, Val).).
2. Do NOT redeclare 'deny'. It is already declared.
3. Prioritize using the Domain Vocabulary over raw json_xxx predicates if available.
4. If mapping raw JSON, prefer creating a "Helper Predicate" first (e.g., amount(R, V) :- json_num...) to keep the deny rule clean.
5. Strings must be double-quoted.
6. Variables start with uppercase (e.g., P, Amount).
7. Do NOT use aggregation (max, count) unless absolutely necessary.

Output JSON only: {"datalog_content": "...", "explanation": "..."}`, factsList, autoVocab, iclContent)

	currentReq := userReq
	var lastErr error

	// The Loop (Max 5 Retries)
	for i := 0; i < 5; i++ {
		// Step B (Generation)
		policy, err := ai.GenerateStruct[GeneratedPolicy](ctx, gen, systemPrompt, currentReq)
		if err != nil {
			// If generation itself fails (e.g. network), we probably shouldn't blindly retry unless it's transient.
			lastErr = err
			// Feedback for JSON error
			feedback := fmt.Sprintf("Generation failed: %v", err)
			currentReq = fmt.Sprintf("%s\n\n[SYSTEM CORRECTION]: Previous attempt invalid.\nError: %s\nFix syntax immediately.", userReq, feedback)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Step C (Validation)
		schemaDeclsVal := ""
		if schema != nil && len(schema.Declarations) > 0 {
			schemaDeclsVal = strings.Join(schema.Declarations, "\n")
		}
		if err := ValidatePolicySyntax(policy.DatalogContent, schemaDeclsVal); err != nil {
			lastErr = err
			// Step D (Decision) - Update feedback
			feedback := err.Error()
			fmt.Printf("DEBUG: Validation failed for Datalog:\n%s\nError: %s\n", policy.DatalogContent, feedback)
			// Step A (Prompting) - Update prompt for next iteration
			currentReq = fmt.Sprintf("%s\n\n[SYSTEM CORRECTION]: Previous attempt invalid.\nError: %s\nCheck your decl syntax. Do NOT use .Decl or .decl.", userReq, feedback)

			// Sleep briefly
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Success
		return &policy, nil
	}

	return nil, fmt.Errorf("failed after 5 retries. Last error: %w", lastErr)
}
