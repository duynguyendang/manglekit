# Simple Manglekit Example

This example provides a basic, self-contained demonstration of how to use the Manglekit framework as a library. It shows how to programmatically configure and run a complete Retrieval-Augmented Generation (RAG) pipeline.

## What it Does

The application performs the following steps:
1.  **Initializes** the necessary components, including the Genkit framework, a logger, and a Google AI embedder.
2.  **Loads** a sample markdown document from the `data/` directory to serve as the knowledge base.
3.  **Constructs** a hybrid retriever, combining a keyword-based (BM25) and a semantic (dense) retriever.
4.  **Applies** declarative rules from `rules.dlog` using Mangle for both pre- and post-processing.
5.  **Executes** a hardcoded query: `"What is manglekit v1.0?"`.
6.  **Prints** the final, synthesized answer and the source citation to the console.

## File Structure

```
.
├── data/
│   └── knowledge.md  # Sample document for the knowledge base.
├── main.go           # The main application logic.
├── README.md         # This file.
└── rules.dlog        # Declarative Mangle rules for processing.
```

## How to Run

1.  **Set up your environment.** You must have a Google AI API key for the embedder and generator to work.

    ```sh
    export GOOGLE_API_KEY="YOUR_API_KEY"
    ```

2.  **Run the application.** From the root of the `manglekit` repository, execute the following command:

    ```sh
    go run ./examples/simple
    ```

You should see output indicating the query being run, followed by the generated answer and its citation.