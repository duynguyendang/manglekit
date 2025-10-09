package retrieve

import "github.com/duynguyendang/manglekit/core"

// InMemoryOptions provides typed configuration for the in-memory retriever.
// This retriever is useful for simple applications, testing, or scenarios where
// the document set is small and can be held entirely in memory.
type InMemoryOptions struct {
	// Documents is a slice of core.Doc that will be pre-loaded into the
	// retriever's memory store upon initialization.
	Documents []core.Doc
}

// BM25Options provides a type-safe way to configure the BM25 retriever, which
// performs keyword-based search.
type BM25Options struct {
	// Path is the file system path to a directory of documents that will be
	// indexed by the retriever for keyword search.
	Path string `yaml:"path" path:"resolve"`
	// TopK specifies the default number of documents to return if a different
	// limit is not specified in the retrieval request.
	TopK int `yaml:"topK"`
}

// DenseOptions provides a type-safe way to configure a dense (vector-based)
// retriever. This struct is often a marker, as its primary dependencies,
// such as an Embedder and a VectorStore, are expected to be configured
// separately and injected by the central builder.
type DenseOptions struct{}

// HybridOptions provides a type-safe way to configure the hybrid retriever.
// This retriever combines results from multiple underlying retrievers (typically
// a keyword-based one and a dense/vector-based one) to leverage the strengths
// of both methods. The child retrievers are constructed by the builder and
// injected into these fields.
type HybridOptions struct {
	// BM25Retriever is the keyword-based (sparse) retriever instance.
	BM25Retriever Retriever
	// DenseRetriever is the vector-based (dense) retriever instance.
	DenseRetriever Retriever
}
