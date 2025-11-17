# Example: 01-programmatic-setup

This example demonstrates how to build and run a Manglekit pipeline programmatically using the `sdk.NewBuilder()` API.

## Setup

1.  **Set your Google API Key:**
    Copy the `.env.example` file in this directory and replace `"your-api-key-here"` with your actual Google API key.

    ```bash
    cp .env.example .env
    # now edit .env and add your GOOGLE_API_KEY
    ```

2.  **Set up Chroma for semantic search:**
    Start a Chroma instance (requires Docker):

    ```bash
    docker run -d -p 8000:8000 chromadb/chroma
    ```

    Or install Chroma locally:

    ```bash
    pip install chromadb
    chroma run --host localhost --port 8000
    ```

    The example expects Chroma to be available at `http://localhost:8000`.

3.  **Run the example:**
    From the root of the repository, run the following command:

    ```bash
    go run ./examples/01-programmatic-setup
    ```

    The example will execute a query using hybrid search (combining BM25 and Chroma semantic search) and return an answer from the Google LLM.

## What it does

This example builds a "Sandwich" RAG pipeline with the following components:

*   **Orchestrator:** `sandwich` - Orchestrates the RAG pipeline with pre-retrieval and post-retrieval rule stages
*   **LLM:** `google` (Gemini 2.5 Flash) - Generates answers based on retrieved documents
*   **Embedder:** `google-embedder` - Provides text embeddings for semantic search
*   **Retriever:** `hybrid` (combining BM25 keyword search and Chroma semantic search) - Performs hybrid search using both lexical and semantic matching with Reciprocal Rank Fusion (RRF)
*   **RuleSet:** `mangle` (using rules from `examples/rules/acme-rules.dlog`) - Enforces policies and rules at pre and post stages
*   **StateProvider:** `inmemory` - Manages session state during pipeline execution

It then executes the query "What is MangleKit?" and prints the answer to the console along with citations.

**Note:** The semantic search component requires Chroma to be running. See the Setup section for instructions.

## How it works

### 1. Builder Creation
The example starts by creating a new programmatic builder via `sdk.NewBuilder()`. This automatically registers all standard providers (retrievers, LLMs, orchestrators, etc.) from the `providers/all` package.

### 2. Component Configuration
Each component is configured with its specific options struct:
- **GoogleEmbedderOptions** - Configures the Google embedding model (defaults to `text-embedding-004`)
- **GenkitRetrieverOptions** - Configures Chroma as the semantic search provider with endpoint configuration
- **BM25Options** - Specifies the path to documents to index for keyword-based search
- **HybridOptions** - Combines BM25 and Chroma retrievers using Reciprocal Rank Fusion (RRF) for optimal ranking
- **GoogleOptions** - Specifies the model (gemini-2.5-flash) and reads API key from environment
- **MangleOptions** - Specifies the path to Datalog rule files
- **InMemoryOptions** - Configures the in-memory state provider
- **SandwichOptions** - Wires together the orchestrator with named component references

### 3. Builder Chaining
Components are added to the builder using the fluent `WithOptions()` API, which returns the builder for method chaining:

```go
builder.
    WithOptions("google_embedder", googleEmbedderOpts).
    WithOptions("semantic_retriever", genkitRetrieverOpts).
    WithOptions("keyword_retriever", bm25SubRetrieverOpts).
    WithOptions("hybrid_retriever", hybridRetrieverOpts).
    WithOptions("google", googleOpts).
    WithOptions("mangle", mangleOpts).
    WithOptions("inmemory", stateOpts).
    WithOptions("sandwich", sandwichOpts)
```

The order of component registration ensures that sub-retrievers (keyword and semantic) are built before the hybrid retriever that combines them.

### 4. Orchestrator Construction
The `Build()` method constructs the orchestrator and all its dependencies in the correct order:

```go
orch, _, err := builder.Build(ctx, "sandwich", "")
```

The second return value is an `Updatable` component (for dynamic document updates), which is empty in this example.

### 5. Query Execution
The orchestrator executes the query through the full pipeline:

```go
answer, err := orch.Execute(ctx, "session-123", query)
```

This triggers:
1. **Pre-retrieval rules** - Query validation, normalization, expansion
2. **Hybrid document retrieval** - Combines results from:
   - **Keyword search (BM25)** - Lexical matching on indexed documents
   - **Semantic search (Chroma)** - Vector similarity using embeddings
   - **Reciprocal Rank Fusion** - Merges rankings from both methods for optimal relevance
3. **Post-retrieval rules** - Document filtering, redaction, entitlement checks
4. **LLM generation** - Answer synthesis based on retrieved documents

**Future Enhancement:** When Genkit retriever providers become available, the hybrid retriever will combine results from both keyword search (BM25) and semantic search (Genkit-retriever) using Reciprocal Rank Fusion (RRF) for optimal relevance.

### 6. Resource Cleanup
The orchestrator's resources are properly cleaned up using a deferred close:

```go
defer func() {
    if err := orch.Close(ctx); err != nil {
        log.Printf("Warning: Error closing orchestrator: %v", err)
    }
}()
```

## Key Differences from YAML Configuration

This programmatic approach differs from the declarative YAML approach in that:

- **Type-safe:** Component options are strongly typed Go structs
- **Fluent API:** Method chaining for readable configuration
- **Runtime flexibility:** Components can be configured based on runtime conditions
- **Direct control:** Full control over component instantiation and wiring

## Environment Variables

The example uses the following environment variables:

- `GOOGLE_API_KEY` - Your Google API key for authentication with Gemini models

Set these in your `.env` file:
```
GOOGLE_API_KEY=your-actual-api-key-here
```

## Troubleshooting

### "Failed to create builder"
Ensure the genkit library is properly initialized. Check that your Go environment is set up correctly.

### "Failed to build orchestrator"
This typically means a required component is missing or misconfigured. Check:
- All component names in options match the names passed to `WithOptions()`
- All required dependencies are configured (e.g., StateProvider for Sandwich)
- The orchestrator name passed to `Build()` matches a configured orchestrator

### "insufficient_evidence" error
This error occurs when the retriever cannot find any documents matching the query. This can happen for several reasons:

1. **Insufficient documents in corpus:** The BM25 retriever requires at least 2 documents in the corpus to calculate proper Inverse Document Frequency (IDF) scores. With only 1 document, all IDF values become 0, resulting in zero scores for all queries.
   - Ensure you have at least 2 documents in the indexed directory
   - The example includes `doc1.md` and `doc2.md` in `examples/01-programmatic-setup/docs/`

2. **Query doesn't match documents:** The BM25 retriever uses keyword matching. Ensure your query contains words that appear in the indexed documents.
   - Current test documents contain: "MangleKit", "Go", "framework", "Retrieval-Augmented-Generation", "RAG"
   - Try queries like: "What is MangleKit?", "Go framework", "RAG"

3. **Documents not indexed:** Verify that documents exist in the configured directory (default: `examples/01-programmatic-setup/docs/`) and are in `.md` format.

4. **Rules denying the request:** Check the Mangle rules in `examples/rules/acme-rules.dlog`. The current rules deny queries containing "secret".

### "Authentication failed"
Ensure your `GOOGLE_API_KEY` environment variable is set correctly and has the necessary permissions to access Gemini models.

### "Failed to execute query"
This could indicate:
- Missing or invalid Google API key
- Missing rule files or documents
- Invalid query structure
- Network connectivity issues with Google API

### "chroma retriever support not yet implemented" or connection errors
If you see an error about Chroma not being implemented, this means the Genkit Chroma plugin is not yet available in your version of the Go Genkit SDK.

**Troubleshooting:**
1. **Chroma not running:** Ensure Chroma is running at `http://localhost:8000`:
   ```bash
   docker ps | grep chromadb
   ```
2. **Connection refused:** Check that the Chroma service is accessible:
   ```bash
   curl http://localhost:8000
   ```
3. **Genkit plugin not available:** If you see "chroma retriever support not yet implemented", the Chroma plugin may not be available in your Genkit version. This example requires Genkit Chroma support to be available in the Go SDK.
4. **Fallback to BM25 only:** If you don't have Chroma set up, you can temporarily comment out the semantic_retriever configuration and use only BM25 keyword search by changing `hybridRetrieverOpts.Retrievers` to `[]string{"keyword_retriever"}`.

## Example Output

When successful, the example produces output like:

```
Executing query: What is MangleKit?

Answer: MangleKit is a Go framework for building Retrieval-Augmented Generation (RAG) systems...

Citations:
  - doc1.md (Source: testdata/acme-corp/doc1.md)
```

## Next Steps

- Explore the `examples/rules/acme-rules.dlog` file to understand the rule syntax
- Check `testdata/acme-corp/` to see the documents being indexed
- Try modifying the query or rules to see how the pipeline responds
- Look at the `config.yaml` example for the declarative YAML approach
- Experiment with different Google models (e.g., `gemini-1.5-pro`, `gemini-2.0-flash`)
- Add more documents to `testdata/acme-corp/` to expand the knowledge base
