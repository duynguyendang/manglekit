package inmemory_vector

import (
	"github.com/duynguyendang/manglekit/core"
)

// InMemoryVectorOptions configures the in-memory vector retriever.
// Supports three loading modes: pre-embedded documents, markdown files, or dynamic embedding.
type InMemoryVectorOptions struct {
	// Documents is a slice of pre-embedded documents (optional).
	// If provided, Embedder is ignored and documents must have non-zero vectors.
	Documents []core.Doc `yaml:"documents,omitempty"`

	// MarkdownFiles is a list of file paths to markdown files for RAG (optional).
	// Files are automatically chunked and embedded using the specified embedder.
	// Paths can be absolute or relative to the working directory.
	MarkdownFiles []string `yaml:"markdown_files,omitempty"`

	// ChunkSize is the target size for markdown text chunks (in characters).
	// Default is 500 characters.
	ChunkSize int `yaml:"chunk_size,omitempty"`

	// ChunkOverlap is the overlap between consecutive chunks (in characters).
	// Default is 100 characters. Helps maintain context across chunk boundaries.
	ChunkOverlap int `yaml:"chunk_overlap,omitempty"`

	// Embedder is the name of the embedder to use for dynamic embedding (required if Documents is empty).
	Embedder string `yaml:"embedder"`

	// TopK is the default number of nearest neighbors to return.
	TopK int `yaml:"top_k,omitempty"`
}

func (o *InMemoryVectorOptions) ProviderName() string    { return "inmemory-vector" }
func (o *InMemoryVectorOptions) ProviderKind() core.Kind { return core.KindRetriever }
func (o *InMemoryVectorOptions) GetProviderOptions() any { return o }
func (o *InMemoryVectorOptions) GetEmbedderName() string { return o.Embedder }
