# Policy Copilot Quick Start Guide

## Overview

The **Policy Copilot** (powered by Genkit LLMs) translates natural language policies into executable Datalog rules. It's built using Manglekit's own architecture - demonstrating "dogfooding" of the `core.Action` interface.

## Quick Start

### 1. Basic Usage (Programmatic)

```go
package main

import (
	"context"
	"fmt"
	
	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/policy/rulegenerator"
	"github.com/firebase/genkit/go/ai"
)

func main() {
	ctx := context.Background()
	
	// Define your data schema
	type Transaction struct {
		Region string `mangle:"region"`
		Amount int    `mangle:"amount"`
	}
	
	// Get a Genkit LLM (e.g., from Google AI)
	var model ai.Model  // Initialize with your Genkit model
	
	// Create a core.Action from the LLM
	llmAction := ai.NewGenkitAction("my-llm", model)
	
	// Create the generator
	generator, _ := rulegenerator.New(llmAction, rulegenerator.GeneratorOptions{
		RuleHead: "deny(Req)",
	})
	
	// Generate a rule
	policy := "Block transactions from UK if amount > 1000"
	rule, _ := generator.GenerateRule(ctx, Transaction{}, policy)
	
	fmt.Println(rule)
	// Output: deny(Req) :- region(Req, "UK"), amount(Req, Amount), Amount > 1000.
}
```

### 2. CLI Usage

```bash
# Generate a rule and save to file
mkit gen rule \
  --provider google \
  --model gemini-2.0-flash \
  --schema schema.json \
  --prompt "Block high-value UK transactions" \
  --rule-head "deny(Req)" \
  --out generated_rule.dl

# Print to stdout
mkit gen rule \
  --provider google \
  --model gemini-2.0-flash \
  --schema schema.json \
  --prompt "Allow if amount < 100"
```

### 3. With Guard (Automatic Policy Enforcement)

```go
// Wrap the action with a Guard for automatic policy checks and tracing
guardedAction := guard.Protect(llmAction)

generator, _ := rulegenerator.New(guardedAction, opts)
rule, _ := generator.GenerateRule(ctx, schema, policy)

// The Guard automatically:
// ✅ Enforces policies on the LLM invocation
// ✅ Records traces and metrics
// ✅ Handles timeouts and retries
// No additional code needed!
```

---

## API Reference

### Generator

```go
// Create a new generator
generator, err := rulegenerator.New(
    llmAction core.Action,
    opts GeneratorOptions,
) (*Generator, error)

// Generate a Datalog rule
rule, err := generator.GenerateRule(
    ctx context.Context,
    schemaSample any,        // Sample struct for schema extraction
    policyText string,       // Natural language policy
) (string, error)
```

### GeneratorOptions

```go
type GeneratorOptions struct {
    RuleHead string           // Target predicate (default: "deny(Req)")
    PromptTemplate string     // Custom prompt template
    Examples string           // Few-shot examples
}
```

### Core Pattern: Envelope-Based Communication

```go
// Input to the action
input := core.NewEnvelope(prompt)
input.SetMeta("schema_type", "Transaction")

// Execute via universal core.Action interface
output, _ := generator.llmAction.Execute(ctx, input)

// Extract result
rule, _ := output.Payload.(string)
```

---

## Schema Definition

Your schema struct defines what predicates the LLM can use:

```go
type Transaction struct {
    Region     string `mangle:"region"`       // Becomes: region(Req, StringValue)
    Amount     int    `mangle:"amount"`       // Becomes: amount(Req, IntValue)
    Category   string `mangle:"category"`     // Becomes: category(Req, StringValue)
    Active     bool   `mangle:"is_active"`    // Becomes: is_active(Req, BoolValue)
    SkipField  string `mangle:"-"`            // Skipped from schema
}
```

Generated predicates in schema context:
```
Available Predicates: 
  region(EntityID, StringValue), 
  amount(EntityID, IntValue),
  category(EntityID, StringValue),
  is_active(EntityID, BoolValue)
```

---

## Example Policies

### Policy 1: Block High-Value Transactions
```
Policy: "Block if amount > 1000"
Generated: deny(Req) :- amount(Req, X), X > 1000.
```

### Policy 2: Regional Restrictions
```
Policy: "Deny all transactions from the UK region"
Generated: deny(Req) :- region(Req, "UK").
```

### Policy 3: Complex Rules
```
Policy: "Block if amount > 500 and region is US or UK"
Generated: deny(Req) :- amount(Req, X), X > 500, region(Req, R), (R = "US" ; R = "UK").
```

### Policy 4: Routing (Custom Rule Head)
```
Options: RuleHead = "route(Req, Queue)"
Policy: "Route to fraud-team if amount > 5000"
Generated: route(Req, "fraud-team") :- amount(Req, X), X > 5000.
```

---

## Environment Setup

### Google AI (Gemini)

```bash
# Set your API key
export GOOGLE_GENAI_API_KEY=your_api_key_here

# Optional: Set model
export GENKIT_MODEL=gemini-2.0-flash
```

### Alternative Providers

Future support for:
- OpenAI (ChatGPT, GPT-4)
- Anthropic (Claude)
- Local LLMs (Ollama, LM Studio)

---

## Error Handling

```go
rule, err := generator.GenerateRule(ctx, schema, policy)

if err != nil {
    switch {
    case strings.Contains(err.Error(), "parsing clause failed"):
        fmt.Println("Generated Datalog is syntactically invalid")
    case strings.Contains(err.Error(), "llm action execution failed"):
        fmt.Println("LLM invocation failed")
    case strings.Contains(err.Error(), "not a rule"):
        fmt.Println("Generated a fact, not a rule")
    }
}
```

### Common Issues

| Error | Cause | Solution |
|-------|-------|----------|
| "model not found" | Genkit model not initialized | Ensure model is registered |
| "parsing clause failed" | Invalid Datalog syntax | Review LLM output or provide examples |
| "context deadline exceeded" | LLM timeout | Increase timeout or use smaller model |
| "empty payload" | LLM returned empty response | Check prompt or model availability |

---

## Testing

### Unit Test Example

```go
func TestMyPolicy(t *testing.T) {
    // Create mock action
    mockAction := &mockAction{
        response: `deny(Req) :- amount(Req, X), X > 100.`,
    }
    
    generator, _ := rulegenerator.New(mockAction, opts)
    rule, _ := generator.GenerateRule(ctx, MySchema{}, "Block > 100")
    
    if !strings.Contains(rule, "amount") {
        t.Fatal("Generated rule doesn't use amount predicate")
    }
}
```

---

## Advanced: Custom Actions

You can use ANY `core.Action` as the LLM:

```go
// Custom action with caching
type CachingAction struct {
    underlying core.Action
    cache      map[string]string
}

func (c *CachingAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
    key := fmt.Sprintf("%v", input.Payload)
    
    if cached, ok := c.cache[key]; ok {
        return core.NewEnvelope(cached), nil
    }
    
    output, err := c.underlying.Execute(ctx, input)
    if err == nil {
        c.cache[key] = output.Payload.(string)
    }
    return output, err
}

// Use it:
cachedAction := &CachingAction{
    underlying: aiAction,
    cache:      make(map[string]string),
}
generator, _ := rulegenerator.New(cachedAction, opts)
```

---

## Best Practices

1. **Use Guards in Production**
   ```go
   llmAction := ai.NewGenkitAction("llm", model)
   guardedAction := guard.Protect(llmAction)  // ← Add this
   generator, _ := rulegenerator.New(guardedAction, opts)
   ```

2. **Provide Schema Samples**
   - Accurate schema = better rules
   - Use real data structures as samples
   - Tag fields with `mangle` names

3. **Clear Policy Descriptions**
   - Be specific: "Block if amount > 1000" not "Block bad"
   - Use actual field names: "region is UK" not "country is UK"
   - Combine with examples in GeneratorOptions

4. **Validate Generated Rules**
   ```go
   rule, _ := generator.GenerateRule(ctx, schema, policy)
   
   // Generator validates syntax automatically
   // You can add business logic validation:
   if strings.Count(rule, "region") > 3 {
       fmt.Println("Warning: Rule has many region checks")
   }
   ```

5. **Use Appropriate Rule Heads**
   ```go
   // For access control
   opts.RuleHead = "deny(Req)"
   
   // For routing
   opts.RuleHead = "route(Req, Queue)"
   
   // For scoring
   opts.RuleHead = "score(Req, Score)"
   ```

---

## Troubleshooting

### LLM is returning facts instead of rules

**Problem:**
```
Generated: deny(Req).
Error: "generated code is not a rule (missing ':-')"
```

**Solution:** Update policy description:
```go
// Instead of:
policy := "Deny everything"

// Use:
policy := "Block if amount > 0"  // Forces a rule with body
```

### Schema predicates not being used

**Problem:** LLM ignores some schema fields

**Solution:** Provide examples:
```go
opts.Examples = `
Example:
- User Policy: "Block high-value UK transactions"
- Schema: region(Req, String), amount(Req, Int)
- Your Output:
deny(Req) :- region(Req, "UK"), amount(Req, Amount), Amount > 1000.
`
```

### Timeouts with large models

**Problem:** Genkit times out on large models

**Solution:**
```go
// Set longer context deadline
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

---

## Resources

- **Documentation:** See `docs/CONTEXT.md` for architecture
- **Examples:** Check `policy/rulegenerator/example_test.go`
- **Genkit Docs:** https://github.com/firebase/genkit
- **Datalog Syntax:** https://github.com/google/mangle

---

## Next Steps

1. ✅ Set up your Genkit model
2. ✅ Define your schema struct
3. ✅ Create an `adapters/ai.NewGenkitAction()`
4. ✅ Initialize the Generator
5. ✅ Start generating rules!

---

**Happy Policy Coding! 🚀**
