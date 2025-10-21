# Manglekit Examples

This directory contains a set of examples demonstrating the various features and use cases of the Manglekit framework. Each example is self-contained and can be run independently.

## Examples

| # | Example | Description |
|---|---|---|
| 1 | [01-basic-rag](./01-basic-rag) | Demonstrates the simplest way to get started with a basic Retrieval-Augmented Generation (RAG) pipeline using a local YAML configuration. |
| 2 | [02-logic-layer-mode](./02-logic-layer-mode) | Shows how to use the logic layer mode to provide additional context to the RAG pipeline in the form of Datalog facts, allowing for more dynamic and context-aware retrieval. |
| 3 | [03-custom-prompt](./03-custom-prompt) | Illustrates how to use a custom prompt template to tailor the generation process to specific needs. |
| 4 | [04-declarative-flow](./04-declarative-flow) | Demonstrates the declarative flow, where the entire RAG pipeline is defined and controlled by Mangle rules, enabling complex logic and conditional execution. |
| 5 | [05-chat-with-data](./05-chat-with-data) | Provides an example of building a chat application that can interact with and answer questions about your data. |
| 6 | [06-schema-validation](./06-schema-validation) | Shows how to use Mangle's powerful rule engine for input and output schema validation, ensuring data integrity and security. |
| 7 | [07-rdf-knowledge-base](./07-rdf-knowledge-base) | Demonstrates how to integrate an RDF knowledge base into the RAG pipeline, allowing for structured and semantic data retrieval. |
| 8 | [08-symbolic-rag](./08-symbolic-rag) | Illustrates a symbolic RAG pipeline, where the retrieval process is guided by symbolic reasoning and knowledge representation. |
| 9 | [09-genkit-tool](./09-genkit-tool) | Shows how to integrate Manglekit with Genkit, using it as a tool within a larger Genkit-based application. |