# Example: 01-programmatic-setup

This example demonstrates how to build and run a Manglekit pipeline programmatically using the `sdk.NewBuilder()` API.

## Setup

1.  **Set your OpenAI API Key:**
    Copy the `.env` file in this directory and replace `"your-api-key-here"` with your actual OpenAI API key.

    ```bash
    cp .env.example .env
    # now edit .env
    ```

2.  **Run the example:**
    From the root of the repository, run the following command:

    ```bash
    go run ./examples/01-programmatic-setup
    ```

## What it does

This example builds a "Sandwich" RAG pipeline with the following components:

*   **Orchestrator:** `sandwich`
*   **LLM:** `openai` (GPT-3.5 Turbo)
*   **Retriever:** `bm25` (indexing documents from `testdata/acme-corp`)
*   **RuleSet:** `mangle` (using rules from `examples/rules/acme-rules.dlog`)

It then executes the query "What is MangleKit?" and prints the answer to the console.
