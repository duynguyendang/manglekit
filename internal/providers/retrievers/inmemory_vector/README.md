# InMemoryVectorRetriever with Markdown RAG Support

## Overview

The `InMemoryVectorRetriever` now provides comprehensive support for **Retrieval-Augmented Generation (RAG) with local markdown files**. This enables you to build RAG systems that automatically load, chunk, and embed markdown documentation for semantic search.

## Features

- **Markdown File Loading**: Automatically load and parse markdown files from local file system
- **Intelligent Chunking**: Smart text chunking that respects semantic boundaries with configurable chunk size and overlap
- **Flexible Embedding**: Support for dynamic embedding via any Genkit-compatible embedder (OpenAI, Google, Cohere, etc.)
- **Pre-embedded Documents**: Load documents with pre-computed vectors for offline scenarios
- **Hybrid Mode**: Combine pre-embedded documents with markdown files in a single retriever
- **Thread-safe Retrieval**: Concurrent query handling with RWMutex protection
- **Production-ready**: Full validation, error handling, and comprehensive logging

## Configuration

### Option 1: Markdown Files with Dynamic Embedding

Load markdown files and embed them on startup:

```yaml
retrievers:
  markdown_retriever:
    provider: inmemory-vector
    embedder: openai                    # Required for markdown loading
    markdown_files:
      - ./docs/guide.md
      - ./docs/api.md
      - ./docs/examples.md
    chunk_size: 500                     # Characters per chunk
    chunk_overlap: 100                  # Overlap for context
    top_k: 5                            # Default results per query
```

### Option 2: Pre-embedded Documents

Use documents with pre-computed vectors:

```yaml
retrievers:
  preembedded_retriever:
    provider: inmemory-vector
    top_k: 5
    # Documents loaded programmatically with vector field in Meta
```

### Option 3: Embedder-only Configuration

Configure embedder for query-time embedding:

```yaml
retrievers:
  query_embedding_retriever:
    provider: inmemory-vector
    embedder: openai
    top_k: 5
```

## Configuration Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `markdown_files` | `[]string` | No | - | List of markdown file paths (absolute or relative) |
| `embedder` | `string` | See notes | - | Name of embedder provider (required for markdown loading) |
| `chunk_size` | `int` | No | 500 | Target size for text chunks in characters |
| `chunk_overlap` | `int` | No | 100 | Overlap between consecutive chunks in characters |
| `top_k` | `int` | No | 10 | Default number of nearest neighbors to return |
| `documents` | `[]core.Doc` | No | - | Pre-embedded documents (programmatic only) |

**Notes:**
- Either `documents`, `markdown_files`, or `embedder` must be specified
- `embedder` is required when `markdown_files` are specified
- All file paths are resolved to absolute paths; relative paths are relative to current working directory

## Usage Examples

### Go Programmatic API

```go
import (
    "context"
    "github.com/duynguyendang/manglekit"
    "github.com/duynguyendang/manglekit/internal/providers/retrievers/inmemory_vector"
    _ "github.com/duynguyendang/manglekit/providers/all"
)

// Load markdown files with embedding
opts := inmemory_vector.InMemoryVectorOptions{
    MarkdownFiles: []string{"./docs/guide.md", "./docs/api.md"},
    Embedder:      "openai",
    ChunkSize:     500,
    ChunkOverlap:  100,
    TopK:          5,
}

// Build via YAML configuration
b, err := sdk.FromConfig(ctx, configYAML)
if err != nil {
    log.Fatal(err)
}

// Use the retriever
query := core.RetrieveRequest{
    Query: "How do I get started with the API?",
    TopK:  3,
}

result, err := retriever.Retrieve(ctx, query)
if err != nil {
    log.Fatal(err)
}

for _, doc := range result.Docs {
    fmt.Printf("Document: %s (Score: %.3f)\n", doc.ID, doc.Meta["score"])
    fmt.Printf("Content: %s\n\n", doc.Text)
}
```

### YAML Configuration

See `testdata/inmemory_vector_markdown_example.yaml` for complete examples.

## How It Works

### 1. Markdown File Loading

- Reads markdown files from specified paths
- Validates file existence and readability
- Preserves file metadata (path, URI)

### 2. Smart Chunking

The chunking algorithm:
- Splits text by newlines
- Respects configurable chunk size
- Maintains overlap between chunks for context preservation
- Cleans up whitespace and handles UTF-8 encoding

Example chunking with size=200, overlap=50:
```
Input: "# Section 1\nThis is content...\n\n# Section 2\nMore content..."
↓
Chunks:
  1. "# Section 1\nThis is content that fills up about 200 characters..."
  2. "...characters from end of chunk 1 (50 char overlap)\n\n# Section 2\nMore content..."
  3. "...content from chunk 2 (50 char overlap)..."
```

### 3. Embedding and Storage

- Each chunk is sent to the embedder
- Returned embeddings (vectors) are stored in `document.Meta["vector"]`
- Documents are indexed by unique ID combining filename and chunk index
- Dimension is detected from first embedding response

### 4. Semantic Search at Query Time

- Query is embedded using the same embedder
- Cosine similarity is computed against all document vectors
- Top-K documents are returned sorted by similarity score

## Chunking Strategy

### When to Adjust Chunk Size

- **Smaller chunks (200-300)**: Better for granular queries, more documents
- **Medium chunks (400-600)**: Balanced approach, recommended default
- **Larger chunks (800+)**: Better context preservation, fewer documents

### When to Adjust Chunk Overlap

- **No overlap (0)**: Fast, no context sharing between chunks
- **Small overlap (50-100)**: Recommended, preserves context
- **Large overlap (200+)**: Maximum context, slower processing

## Error Handling

The retriever validates:

1. **File existence**: Checks if markdown files exist before loading
2. **UTF-8 encoding**: Converts invalid UTF-8 to valid sequences
3. **Vector dimensions**: Ensures all vectors match first embedding dimension
4. **Empty content**: Filters out empty chunks after chunking
5. **Loading mode**: Ensures at least one valid loading source is specified

Common errors and solutions:

| Error | Cause | Solution |
|-------|-------|----------|
| "markdown file not found: ..." | File path doesn't exist | Check file path, use absolute paths |
| "Embedder is required when MarkdownFiles are provided" | Missing embedder config | Add `embedder:` field to options |
| "either Documents or MarkdownFiles... must be provided" | No loading source specified | Configure at least one loading mode |
| "embedding dimension mismatch" | Embedder returned inconsistent dimensions | Ensure consistent embedder model |

## Performance Characteristics

### Initialization

- **Time**: O(n*m) where n = number of files, m = average file size
- **For 10 files, ~100KB each**: ~1-5 seconds (depending on embedder latency)
- **Memory**: ~file_size * (1 + vector_dimension * 4 bytes)

### Query Time

- **Time**: O(d) where d = number of documents (chunks)
- **Typical**: <100ms for 1000 chunks with local vectors
- **Bottleneck**: Embedder API latency if querying before local caching

### Space Complexity

- **Documents**: Stored in memory map
- **Vectors**: float32 arrays (4 bytes per component)
- **For 1000 chunks with 1536-dim OpenAI embeddings**: ~6MB for vectors

## Testing

Comprehensive test suite included:

```bash
# Run all tests
go test ./internal/providers/retrievers/inmemory_vector/... -v

# Run specific tests
go test -run TestMarkdownLoader_LoadMarkdownFiles -v
go test -run TestCosineSimilarity -v
go test -run TestOptionsValidateLoadingMode -v
```

Tests cover:
- ✅ Markdown file loading and validation
- ✅ Intelligent chunking with overlap
- ✅ Option validation and loading modes
- ✅ Cosine similarity computation
- ✅ Document sanitization
- ✅ File not found error handling
- ✅ Vector dimension validation

## Limitations and Future Work

### Current Limitations

1. **Synchronous embedding**: Markdown loading blocks until all chunks are embedded
2. **Memory-only storage**: No persistence to disk (use case: in-memory RAG for development)
3. **No async markdown loading**: Must load all files at startup
4. **Single embedder**: Cannot use different embedders for different documents

### Future Enhancements

- [ ] Async markdown loading with progress tracking
- [ ] Persistent vector storage (SQLite, PostgreSQL)
- [ ] Batched embedding for faster startup
- [ ] Support for multiple embedders
- [ ] Incremental markdown file updates
- [ ] Automatic markdown reloading on file changes
- [ ] Vector caching layer

## Integration with Sandwich Pattern

Example full pipeline:

```yaml
embedders:
  openai:
    provider: openai
    model: text-embedding-3-small

llms:
  gpt4:
    provider: openai
    model: gpt-4

retrievers:
  markdown:
    provider: inmemory-vector
    embedder: openai
    markdown_files: [./docs/guide.md]

orchestrator:
  provider: sandwich
  retriever: markdown
  llm: gpt4
  top_k: 3
```

This creates:
1. **Rules→Retrieval**: Pre-retrieval rule evaluation (if configured)
2. **Retrieval**: Fetch top-3 documents from markdown files
3. **Reranking**: Optional reranking of results (if configured)
4. **Generation**: Feed context to GPT-4 for answer generation
5. **Rules**: Post-generation rule evaluation (if configured)

## Debugging

Enable debug logging:

```go
// Logs will show:
// - File loading progress
// - Chunk creation details
// - Embedding requests/responses
// - Vector similarity scores
// - Query processing steps

log.Logger.Debugf("loading markdown files...")
```

## FAQ

**Q: Can I update markdown files after initialization?**  
A: No, currently all markdown files are loaded at startup. Reloader patterns coming in future releases.

**Q: How large can my markdown files be?**  
A: Limited by available memory. For 1GB of markdown chunked into 500-char chunks ≈ 2M documents, ~12GB RAM needed for OpenAI embeddings.

**Q: Can I use this without embeddings?**  
A: Yes, use pre-embedded documents mode or embedder-only with documents loaded programmatically.

**Q: What embedders are supported?**  
A: Any Genkit-compatible embedder: OpenAI, Google (Vertex/MakerSuite), Cohere, Anthropic, Ollama, etc.

**Q: Is this suitable for production?**  
A: Yes, with caveats: Use in production for in-memory RAG scenarios. For large-scale, consider persistent vector stores (Chroma, Weaviate, etc.).

## Contributing

See `AGENTS.md` for guidelines on:
- Adding new features
- Writing tests
- Documentation standards
- Code review expectations
