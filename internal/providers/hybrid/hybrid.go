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
	// BM25Retriever is the keyword-based (sparse) retriever instance.
	BM25Retriever core.Retriever
	// DenseRetriever is the vector-based (dense) retriever instance.
	DenseRetriever core.Retriever
}

func (o HybridOptions) ProviderName() string { return "hybrid" }
func (o HybridOptions) ProviderKind() core.Kind   { return core.KindRetriever }

func Register(r *manglekit.Registry) {
	manglekit.Register(r, HybridOptions{},
		func(ctx context.Context, deps diapi.RetrieverDeps, cfg HybridOptions) (core.Retriever, error) {
			if deps.BuildSubRetriever == nil {
				return nil, fmt.Errorf("hybrid retriever factory requires the BuildSubRetriever capability, but it was not provided")
			}

			// Build sub-components using the provided capability function.
			bm25Retriever, err := deps.BuildSubRetriever(ctx, "bm25", nil)
			if err != nil {
				return nil, fmt.Errorf("failed to build bm25 component for hybrid retriever: %w", err)
			}
			denseRetriever, err := deps.BuildSubRetriever(ctx, "dense", nil)
			if err != nil {
				return nil, fmt.Errorf("failed to build dense component for hybrid retriever: %w", err)
			}
			cfg.BM25Retriever = bm25Retriever
			cfg.DenseRetriever = denseRetriever
			return New(cfg)
		},
	)
}

// Retriever implements the `retrieve.Retriever` interface by combining results
// from multiple underlying retrievers, typically a sparse (BM25) and a dense
// (vector-based) retriever. This approach leverages the strengths of both
// keyword matching and semantic understanding.
type Retriever struct {
	bm25  core.Retriever
	dense core.Retriever
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
func New(opts HybridOptions) (core.Retriever, error) {
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
// ctx is the context for the API call.
// req contains the query string and `TopK` value, which are passed to the
// underlying retrievers and used to trim the final fused result set.
// It returns a single, fused, and re-ranked `core.RetrieveResult` or an error if
// either of the underlying retrieval operations fail.
func (h *Retriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	if h.dense == nil {
		// If dense retriever is not configured, just use BM25.
		return h.bm25.Retrieve(ctx, req)
	}

	g, gCtx := errgroup.WithContext(ctx)
	var bm25Result, denseResult core.RetrieveResult

	g.Go(func() error {
		var err error
		bm25Result, err = h.bm25.Retrieve(gCtx, req)
		return err
	})

	g.Go(func() error {
		var err error
		denseResult, err = h.dense.Retrieve(gCtx, req)
		return err
	})

	if err := g.Wait(); err != nil {
		return core.RetrieveResult{}, err
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

	return core.RetrieveResult{Docs: finalDocs}, nil
}
