package rerank

import "github.com/duynguyendang/manglekit/core"

// Request encapsulates the input for a reranking operation.
type Request struct {
	// Query is the original user query.
	Query string
	// Docs is the slice of documents retrieved from the initial retrieval stage.
	Docs []core.Doc
	// TopK is the number of documents to return after reranking.
	TopK int
}

// ScoredDoc represents a document that has been assigned a relevance score by a Reranker.
type ScoredDoc struct {
	// Doc is the original document.
	Doc core.Doc
	// Score is the relevance score assigned by the reranker. A higher score
	// indicates greater relevance.
	Score float64
}

// Reranker defines the interface for components that re-score and re-order a
// list of documents based on their relevance to a query.
type Reranker interface {
	// Rerank takes a Request, scores the contained documents, and returns a
	// new list of ScoredDoc, sorted in descending order of relevance.
	//
	// req is the reranking request containing the query and documents.
	// It returns a sorted slice of ScoredDoc or an error if the operation fails.
	Rerank(req Request) ([]ScoredDoc, error)
}