package retrieve

import "github.com/duynguyendang/manglekit/core"

// InMemoryOptions provides typed configuration for the in-memory retriever.
// This retriever stores and searches documents directly in memory.
type InMemoryOptions struct {
	// Documents is a slice of core.Doc that will be loaded into the retriever's memory store.
	Documents []core.Doc
}

// BM25Options provides a type-safe way to configure the BM25 (keyword-based) retriever.
type BM25Options struct {
	// Path is the file path to a directory of documents that will be indexed by the retriever.
	Path string `yaml:"path" path:"resolve"`
	// TopK specifies the default number of documents to return if not specified in the request.
	TopK int `yaml:"topK"`
}

// DenseOptions provides a type-safe way to configure a dense (vector-based) retriever.
// Its primary dependencies, such as an Embedder and a VectorStore, are expected
// to be configured separately and injected by the builder.
type DenseOptions struct{}

// HybridOptions provides a type-safe way to configure the hybrid retriever, which
// combines results from multiple retrievers (typically a keyword and a dense retriever).
// The child retrievers are constructed by the builder and injected into these fields.
type HybridOptions struct {
	// BM25Retriever is the keyword-based retriever instance.
	BM25Retriever Retriever
	// DenseRetriever is the vector-based retriever instance.
	DenseRetriever Retriever
}