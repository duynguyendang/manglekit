// Package hybrid provides an implementation of a hybrid retriever that combines
// results from sparse (keyword) and dense (vector) search to improve relevance.
package hybrid

import (
	"context"
	"fmt"
	"sort"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"golang.org/x/sync/errgroup"
)

// HybridOptions provides a type-safe way to configure the hybrid retriever.
// This retriever combines results from multiple underlying retrievers (typically
// a keyword-based one and a dense/vector-based one) to leverage the strengths
// of both methods. The child retrievers are constructed by the builder and
// injected into these fields.
type HybridOptions struct {
	// Retrievers is a list of names of the sub-retrievers to be used.
	// These retrievers will be resolved by the builder and injected.
	Retrievers []string `yaml:"retrievers"`
	// RRF_K is the constant used in the Reciprocal Rank Fusion (RRF) algorithm.
	// A value of 0.0 indicates that the default should be used.
	RRF_K float64 `yaml:"rrf_k,omitempty"`
}

func (o HybridOptions) ProviderName() string    { return "hybrid" }
func (o HybridOptions) ProviderKind() core.Kind      { return core.KindRetriever }
func (o HybridOptions) GetSubRetrievers() []string { return o.Retrievers }

func Register(r *manglekit.Registry) {
	manglekit.Register(r, HybridOptions{},
		func(ctx context.Context, deps diapi.RetrieverDeps, cfg HybridOptions) (core.Retriever, error) {
			rrf_k := cfg.RRF_K
			if rrf_k == 0.0 {
				rrf_k = 60.0 // Default value
			}
			return New(deps.SubRetrievers, rrf_k)
		},
	)
}

// Retriever implements the `retrieve.Retriever` interface by combining results
// from multiple underlying retrievers, typically a sparse (BM25) and a dense
// (vector-based) retriever. This approach leverages the strengths of both
// keyword matching and semantic understanding.
type Retriever struct {
	retrievers []core.Retriever
	rrf_k      float64
}

// New is the constructor for the hybrid retriever. It is registered with the
// MangleKit registry for the "hybrid" provider name. The builder is responsible
// for constructing the child retrievers and injecting them via the options struct.
//
// opts contains the pre-built child retrievers. A `BM25Retriever` is required,
// while the `DenseRetriever` is optional. If the dense retriever is nil, the
// hybrid retriever will fall back to using only the BM25 retriever.
// It returns an initialized `core.Retriever` or an error if the BM25
// retriever dependency is missing.
func New(retrievers []core.Retriever, rrf_k float64) (core.Retriever, error) {
	if len(retrievers) == 0 {
		return nil, fmt.Errorf("hybrid: at least one retriever is required")
	}
	return &Retriever{
		retrievers: retrievers,
		rrf_k:      rrf_k,
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
// ctx is the context for the API call.
// req contains the query string and `TopK` value, which are passed to the
// underlying retrievers and used to trim the final fused result set.
// It returns a single, fused, and re-ranked `core.RetrieveResult` or an error if
// either of the underlying retrieval operations fail.
func (h *Retriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	if len(h.retrievers) == 1 {
		return h.retrievers[0].Retrieve(ctx, req)
	}

	g, gCtx := errgroup.WithContext(ctx)
	results := make(chan core.RetrieveResult, len(h.retrievers))

	for _, r := range h.retrievers {
		retriever := r // Capture loop variable
		g.Go(func() error {
			res, err := retriever.Retrieve(gCtx, req)
			if err != nil {
				return err
			}
			results <- res
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return core.RetrieveResult{}, err
	}
	close(results)

	// --- Reciprocal Rank Fusion (RRF) Logic ---
	scores := make(map[string]float64)
	allDocsMap := make(map[string]core.Doc)
	k := h.rrf_k

	for result := range results {
		for rank, doc := range result.Docs {
			scores[doc.ID] += 1.0 / (k + float64(rank))
			if _, exists := allDocsMap[doc.ID]; !exists {
				allDocsMap[doc.ID] = doc
			}
		}
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

	return core.RetrieveResult{Docs: finalDocs}, nil
}
