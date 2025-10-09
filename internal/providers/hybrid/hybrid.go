// Package hybrid provides an implementation of a hybrid retriever that combines
// results from sparse (keyword) and dense (vector) search to improve relevance.
package hybrid

import (
	"fmt"
	"sort"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/retrieve"
	"golang.org/x/sync/errgroup"
)

func init() {
	// Register the type-safe constructor with the MangleKit framework.
	manglekit.RegisterRetriever("hybrid", New)
}

// Retriever implements the `retrieve.Retriever` interface by combining results
// from multiple underlying retrievers, typically a sparse (BM25) and a dense
// (vector-based) retriever. This approach leverages the strengths of both
// keyword matching and semantic understanding.
type Retriever struct {
	bm25  retrieve.Retriever
	dense retrieve.Retriever
}

// New is the constructor for the hybrid retriever. It is registered with the
// MangleKit registry for the "hybrid" provider name. The builder is responsible
// for constructing the child retrievers and injecting them via the options struct.
//
// opts contains the pre-built child retrievers. A `BM25Retriever` is required,
// while the `DenseRetriever` is optional. If the dense retriever is nil, the
// hybrid retriever will fall back to using only the BM25 retriever.
// It returns an initialized `retrieve.Retriever` or an error if the BM25
// retriever dependency is missing.
func New(opts retrieve.HybridOptions) (retrieve.Retriever, error) {
	if opts.BM25Retriever == nil {
		return nil, fmt.Errorf("hybrid: BM25Retriever is required")
	}
	// The dense retriever is optional.
	return &Retriever{
		bm25:  opts.BM25Retriever,
		dense: opts.DenseRetriever,
	}, nil
}

// Retrieve concurrently executes searches on its underlying sparse (BM25) and dense
// retrievers. It then merges the two result sets using a Reciprocal Rank Fusion
// (RRF) algorithm. RRF creates a new score for each unique document based on its
// rank in each result set, providing a robust, combined ranking that does not
// depend on the absolute scores of the underlying systems.
// This method satisfies the `retrieve.Retriever` interface.
//
// If the dense retriever is not configured, this method will fall back to
// returning only the results from the BM25 retriever.
//
// req contains the query string and `TopK` value, which are passed to the
// underlying retrievers and used to trim the final fused result set.
// It returns a single, fused, and re-ranked `retrieve.Result` or an error if
// either of the underlying retrieval operations fail.
func (h *Retriever) Retrieve(req retrieve.Request) (retrieve.Result, error) {
	if h.dense == nil {
		// If dense retriever is not configured, just use BM25.
		return h.bm25.Retrieve(req)
	}

	var g errgroup.Group
	var bm25Result, denseResult retrieve.Result

	g.Go(func() error {
		var err error
		bm25Result, err = h.bm25.Retrieve(req)
		return err
	})

	g.Go(func() error {
		var err error
		denseResult, err = h.dense.Retrieve(req)
		return err
	})

	if err := g.Wait(); err != nil {
		return retrieve.Result{}, err
	}

	// --- Reciprocal Rank Fusion (RRF) Logic ---
	scores := make(map[string]float64)
	const k = 60.0 // RRF constant

	// Process BM25 results.
	for rank, doc := range bm25Result.Docs {
		scores[doc.ID] += 1.0 / (k + float64(rank))
	}

	// Process Dense results.
	for rank, doc := range denseResult.Docs {
		scores[doc.ID] += 1.0 / (k + float64(rank))
	}

	// Combine all unique documents.
	allDocsMap := make(map[string]core.Doc)
	for _, doc := range bm25Result.Docs {
		allDocsMap[doc.ID] = doc
	}
	for _, doc := range denseResult.Docs {
		allDocsMap[doc.ID] = doc
	}

	// Create the final list and sort it by the new RRF score.
	var finalDocs []core.Doc
	for id := range scores {
		finalDocs = append(finalDocs, allDocsMap[id])
	}

	sort.Slice(finalDocs, func(i, j int) bool {
		// Sort in descending order of score.
		return scores[finalDocs[i].ID] > scores[finalDocs[j].ID]
	})

	// Ensure the number of returned documents does not exceed TopK.
	if len(finalDocs) > req.TopK {
		finalDocs = finalDocs[:req.TopK]
	}

	return retrieve.Result{Docs: finalDocs}, nil
}
