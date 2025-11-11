// Package cosine provides an implementation of a reranker that scores documents
// based on their cosine similarity to the query.
package cosine

import (
	"context"
	"fmt"
	"sort"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/firebase/genkit/go/ai"
	"golang.org/x/sync/errgroup"
)

// CosineOptions provides typed configuration for a reranker that operates by
// calculating the cosine similarity between the query embedding and each of the
// document embeddings. This type of reranker requires an embedder to be
// configured separately in the builder to generate the necessary vectors.
type CosineOptions struct {
	// Embedder is the name of the embedder component to use.
	Embedder string `json:"embedder"`
	// TopK specifies the default number of top-scoring documents to return if a
	// different limit is not specified in the reranking request itself.
	TopK int `json:"topK,omitempty"`
	// VectorDim is the expected dimensionality of the embedding vectors. While
	// not always used directly by the reranker logic, it can be useful for
	// validation or pre-allocation.
	VectorDim int `json:"vectorDim,omitempty"`
}

func (o *CosineOptions) ProviderName() string { return "cosine" }
func (o *CosineOptions) ProviderKind() core.Kind   { return core.KindReranker }
func (o *CosineOptions) GetEmbedder() string    { return o.Embedder }

func Register(r *manglekit.Registry) {
	manglekit.Register(r, &CosineOptions{},
		func(ctx context.Context, deps diapi.RerankerDeps, cfg *CosineOptions) (core.Reranker, error) {
			if deps.Embedder == nil {
				return nil, fmt.Errorf("cosine reranker factory requires an 'embedder' dependency, but it was not provided")
			}
			return New(*cfg, deps)
		},
	)
}

// Reranker implements the `rerank.Reranker` interface. It re-scores documents
// by calculating the cosine similarity between a query's vector embedding and
// the vector embedding of each document. This is a common and effective method
// for refining search results based on semantic relevance.
type Reranker struct {
	embedder ai.Embedder
	topK     int
}

// New is the constructor for the cosine similarity reranker. It is registered
// with the MangleKit registry for the "cosine" provider name.
//
// opts provides configuration, primarily the default `TopK` value.
// embedder is the embedding model component used to generate vector embeddings
// for both the query and the documents. This dependency is injected by the builder.
// It returns a configured `core.Reranker` or an error if the embedder is missing.
func New(opts CosineOptions, deps diapi.RerankerDeps) (core.Reranker, error) {
	if deps.Embedder == nil {
		return nil, fmt.Errorf("cosine reranker requires an embedder")
	}
	return &Reranker{
		embedder: deps.Embedder,
		topK:     opts.TopK,
	}, nil
}

// Rerank re-scores a list of documents based on their cosine similarity to the
// query. The process involves embedding the query and all documents in parallel,
// calculating the similarity score for each document, and then returning a new
// list sorted by this score in descending order.
// This method satisfies the `rerank.Reranker` interface.
//
// ctx is the context for the API call.
// req contains the query and the initial list of documents to be reranked.
// It returns a sorted slice of `core.ScoredDoc`, trimmed to the configured
// `TopK` value, or an error if any of the embedding operations fail.
func (r *Reranker) Rerank(ctx context.Context, req core.RerankRequest) ([]core.ScoredDoc, error) {
	if len(req.Docs) == 0 {
		return nil, nil // Nothing to rerank.
	}

	// 1. Embed the query.
	embedReq := &ai.EmbedRequest{
		Input: []*ai.Document{ai.DocumentFromText(req.Query, nil)},
	}
	embedResp, err := r.embedder.Embed(ctx, embedReq)
	if err != nil {
		return nil, fmt.Errorf("cosine: failed to embed query: %w", err)
	}
	if len(embedResp.Embeddings) == 0 {
		return nil, fmt.Errorf("cosine: embedder returned no embeddings for query")
	}
	queryVector := embedResp.Embeddings[0].Embedding

	// 2. Embed all the documents in parallel.
	docVectors := make([][]float32, len(req.Docs))
	g, gCtx := errgroup.WithContext(ctx)
	for i, doc := range req.Docs {
		i, doc := i, doc // https://golang.org/doc/faq#closures_and_goroutines
		g.Go(func() error {
			docEmbedReq := &ai.EmbedRequest{
				Input: []*ai.Document{ai.DocumentFromText(doc.Text, nil)},
			}
			docEmbedResp, err := r.embedder.Embed(gCtx, docEmbedReq)
			if err != nil {
				return fmt.Errorf("failed to embed doc %s: %w", doc.ID, err)
			}
			if len(docEmbedResp.Embeddings) > 0 {
				docVectors[i] = docEmbedResp.Embeddings[0].Embedding
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("cosine: one or more document embedding goroutines failed: %w", err)
	}

	// 3. Calculate cosine similarity and create ScoredDoc objects.
	scoredDocs := make([]core.ScoredDoc, 0, len(req.Docs))
	for i, doc := range req.Docs {
		if docVectors[i] != nil {
			score := cosineSimilarity(queryVector, docVectors[i])
			scoredDocs = append(scoredDocs, core.ScoredDoc{
				Doc:   doc,
				Score: float64(score),
			})
		} else {
			// Handle cases where embedding might fail for a doc.
			scoredDocs = append(scoredDocs, core.ScoredDoc{
				Doc:   doc,
				Score: 0.0,
			})
		}
	}

	// 4. Sort documents by their new score in descending order. For documents
	// with the same score, a secondary sort on the document ID is used to
	// guarantee a stable, deterministic ordering.
	sort.SliceStable(scoredDocs, func(i, j int) bool {
		if scoredDocs[i].Score != scoredDocs[j].Score {
			return scoredDocs[i].Score > scoredDocs[j].Score
		}
		return scoredDocs[i].Doc.ID < scoredDocs[j].Doc.ID
	})

	// 5. Trim the results to the specified TopK.
	topK := r.topK
	if req.TopK > 0 { // Allow request-time override
		topK = req.TopK
	}
	if topK > 0 && len(scoredDocs) > topK {
		scoredDocs = scoredDocs[:topK]
	}

	return scoredDocs, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float32) float32 {
	var dotProduct float32
	var aMag, bMag float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		aMag += a[i] * a[i]
		bMag += b[i] * b[i]
	}
	if aMag == 0 || bMag == 0 {
		return 0
	}
	return dotProduct / (sqrt(aMag) * sqrt(bMag))
}

// A simple sqrt function for float32.
func sqrt(n float32) float32 {
	if n < 0 {
		return 0
	}
	var x float32 = n
	var y float32 = 1
	e := float32(0.000001)
	for x-y > e {
		x = (x + y) / 2
		y = n / x
	}
	return x
}
