# Example: Sandwich Orchestrator

This example demonstrates how to configure and run the `Sandwich` orchestrator using the `sdk.Load` function.

It uses mock components for the LLM, retriever, and state provider, making it a self-contained example that does not require any external API keys or dependencies.

## How to Run

To run this example, execute the `main.go` program from the **root of the project directory**, passing a query as a command-line argument.

```bash
go run ./examples/02-sandwich "what is the meaning of life?"
```

## Expected Output

You should see the mock response defined in `config.yaml` printed to the console:

```
Response from mock_llm: This is a mock response from the Sandwich orchestrator.
```
