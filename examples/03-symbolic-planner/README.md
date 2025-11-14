# Symbolic Planner Example

This example demonstrates how to use the symbolic planner in Manglekit to generate multi-step execution plans.

## Overview

The symbolic planner is a component that uses a `core.Reasoner` to generate structured plans based on logical reasoning. It converts queries into input facts, executes the reasoner, and parses the output into a sequence of steps.

## Running the Example

```bash
cd examples/03-symbolic-planner
go run main.go
```

## Expected Output

```
Generating plan for query: What are the latest advances in machine learning?

Generated plan with 2 steps:

Step 1:
  Tool:   retriever
  Reason: Search for relevant documents about the query topic
  Params: {
    "query": "machine learning",
    "topK": 5
  }

Step 2:
  Tool:   llm
  Reason: Generate a comprehensive summary from retrieved documents
  Params: {
    "maxTokens": 500,
    "prompt": "Summarize the following documents"
  }

✓ Plan generated successfully!

In a real application, this plan would be executed by the orchestrator,
which would call each tool in sequence with the specified parameters.
```

## How It Works

1. **Create a Reasoner**: The example uses a mock reasoner that returns a predefined plan structure. In a real application, you would use the Mangle Datalog reasoner or another symbolic reasoning engine.

2. **Configure the Planner**: Create a `symbolic.Options` struct specifying which reasoner to use.

3. **Generate a Plan**: Call `planner.Plan(ctx, query)` to generate a structured plan.

4. **Execute the Plan**: In a real application, an orchestrator would execute each step in sequence.

## Using with YAML Configuration

In production, you would configure the planner via YAML:

```yaml
components:
  - name: "mangle-reasoner"
    kind: "reasoner"
    type: "mangle"
    params:
      rules: |
        # Datalog rules for plan generation
        plan_tool(0, "retriever") :- query_intent("search").
        plan_tool(1, "llm") :- plan_tool(0, "retriever").
      
  - name: "my-planner"
    kind: "planner"
    type: "symbolic"
    params:
      reasoner: "mangle-reasoner"
```

## Reasoner Output Format

The reasoner should return facts in the following format:

- `plan_tool_N`: Tool name for step N
- `plan_params_N`: JSON string or map of parameters for step N
- `plan_reason_N`: Natural language explanation for step N

The planner automatically sorts steps by order (N) and constructs the final plan.

## Key Concepts

- **Symbolic Planning**: Uses logical reasoning to generate plans
- **Reasoner Integration**: Leverages `core.Reasoner` for plan generation
- **Structured Output**: Returns `core.Plan` with sorted `core.Step` sequences
- **Tool Abstraction**: Plans reference tools by name, allowing flexible execution

## Next Steps

- Explore the Mangle Datalog reasoner for more sophisticated planning
- Integrate the planner with orchestrators for automatic plan execution
- Write custom reasoning rules for domain-specific planning
