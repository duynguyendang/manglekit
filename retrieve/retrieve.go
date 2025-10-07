package retrieve

import "github.com/duynguyendang/manglekit/core"

// Request encapsulates a query to a retriever.
type Request struct {
	// Query is the textual query to be used for retrieval.
	Query string
	// TopK is the maximum number of documents to return.
	TopK int
	// Meta is a map for arbitrary metadata to be used by the retriever,
	// such as filters or other contextual information.
	Meta map[string]any
}

// Result holds the documents returned by a retriever.
type Result struct {
	// Docs is a slice of the retrieved documents.
	Docs []core.Doc
	// Meta is a map for arbitrary metadata returned by the retriever,
	// such as scores or debugging information.
	Meta map[string]any
}

// Retriever is the interface for components that can fetch documents based on a query.
type Retriever interface {
	// Retrieve takes a Request and returns a Result containing relevant documents.
	//
	// req is the retrieval request.
	// It returns a Result containing the documents or an error if the retrieval fails.
	Retrieve(req Request) (Result, error)
}

// Updatable defines the interface for retrievers that support runtime
// modification of their document index.
type Updatable interface {
	// Retriever embeds the base Retriever interface.
	Retriever
	// Upsert adds new documents or updates existing ones in the index.
	// If a document with the same ID already exists, it is replaced.
	//
	// docs is a slice of documents to add or update.
	// It returns an error if the operation fails.
	Upsert(docs []core.Doc) error
	// Replace clears the entire existing index and replaces it with a new set of documents.
	//
	// docs is the new slice of documents to populate the index with.
	// It returns an error if the operation fails.
	Replace(docs []core.Doc) error
}