package rerank

import "github.com/duynguyendang/manglekit/core"

// Request encapsulates the input for a reranking operation. It contains the
// query and the initial set of documents to be re-ordered.
type Request struct {
	// Query is the original user query string, which is used to assess the
	// relevance of each document.
	Query string
	// Docs is the slice of documents retrieved from the initial retrieval stage
	// that need to be reranked.
	Docs []core.Doc
	// TopK is the desired number of documents to return after reranking. The
	// implementation should return at most this many of the top-scoring documents.
	TopK int
}

// ScoredDoc represents a document that has been assigned a new relevance score
// by a Reranker. This struct is used to hold the output of the reranking process.
type ScoredDoc struct {
	// Doc is the original document that was scored.
	Doc core.Doc
	// Score is the new relevance score assigned by the reranker. A higher score
	// is expected to indicate greater relevance to the query.
	Score float64
}

// Reranker defines the standard interface for components that re-score and
// re-order a list of documents based on their semantic relevance to a query.
// Rerankers are typically used after an initial, broader retrieval phase to
// refine the set of documents before they are passed to a language model.
type Reranker interface {
	// Rerank takes a Request, computes a new relevance score for each document
	// with respect to the query, and returns a new list of ScoredDoc, sorted
	// in descending order of relevance (highest score first).
	//
	// req is the reranking request containing the query and the initial list of documents.
	// It returns a sorted slice of ScoredDoc or an error if the reranking
	// operation fails.
	Rerank(req Request) ([]ScoredDoc, error)
}
