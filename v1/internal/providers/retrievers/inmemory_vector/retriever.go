package inmemory_vector

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/core/diapi"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/firebase/genkit/go/ai"
)

// InMemoryVectorRetriever performs semantic search using cosine similarity
// over in-memory document vectors. Supports both pre-embedded documents and
// dynamic embedding via embedder dependency.
type InMemoryVectorRetriever struct {
	mu       sync.RWMutex
	docs     map[string]core.Doc
	embedder ai.Embedder
	topK     int
	dim      int // Vector dimensionality
	log      core.Logger
}

// New creates a new InMemoryVectorRetriever.
// It expects InMemoryVectorDeps with both CoreDeps (including Builder) and optionally an Embedder.
func New(opts InMemoryVectorOptions, deps any) (core.Retriever, error) {
	retDeps, ok := deps.(diapi.RetrieverDeps)
	if !ok {
		return nil, fmt.Errorf("invalid deps type for inmemory_vector retriever: got %T", deps)
	}

	log := retDeps.Obs.Logger
	if log == nil {
		log = obslogger.NewStdLogger().With("component", "retriever", "provider", "inmemory-vector")
	}

	// Validate loading mode
	if err := opts.validateLoadingMode(); err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	topK := opts.TopK
	if topK == 0 {
		topK = 10
	}

	r := &InMemoryVectorRetriever{
		docs: make(map[string]core.Doc),
		topK: topK,
		log:  log,
	}

	// Set embedder if configured
	if opts.Embedder != "" {
		log.Debugf("inmemory_vector configured with embedder: %s", opts.Embedder)
		r.dim = 1536 // Default OpenAI embedding dimension
	}

	// Load pre-embedded documents (no embedding needed)
	if len(opts.Documents) > 0 {
		if err := r.loadPreEmbedded(opts.Documents); err != nil {
			return nil, fmt.Errorf("failed to load pre-embedded documents: %w", err)
		}
		r.log.Infof("loaded %d pre-embedded documents", len(opts.Documents))
	}

	// Load and embed markdown files
	if len(opts.MarkdownFiles) > 0 {
		// We need embedder for markdown - this will be handled by setting it via resolver
		// For now we skip markdown loading if no embedder is provided at factory level
		if opts.Embedder == "" {
			r.log.Warnf("markdown files specified but no embedder configured; skipping markdown loading")
		} else {
			r.log.Debugf("markdown loading requested but embedder resolution deferred to handler")
		}
	}

	return r, nil
}

// Retrieve performs vector similarity search using cosine similarity.
// If dynamic embedding is configured, embeds the query first.
func (r *InMemoryVectorRetriever) Retrieve(ctx context.Context, req core.RetrieveRequest) (core.RetrieveResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.docs) == 0 {
		return core.RetrieveResult{}, nil
	}

	// Embed query if we have a dynamic embedder
	queryVector := make([]float32, r.dim)
	if r.embedder != nil {
		embedReq := &ai.EmbedRequest{
			Input: []*ai.Document{ai.DocumentFromText(req.Query, nil)},
		}
		embedResp, err := r.embedder.Embed(ctx, embedReq)
		if err != nil {
			return core.RetrieveResult{}, fmt.Errorf("failed to embed query: %w", err)
		}
		if embedResp == nil || len(embedResp.Embeddings) == 0 {
			return core.RetrieveResult{}, fmt.Errorf("empty embedding returned")
		}
		if len(embedResp.Embeddings[0].Embedding) != r.dim {
			return core.RetrieveResult{}, fmt.Errorf("embedding dimension mismatch: expected %d, got %d", r.dim, len(embedResp.Embeddings[0].Embedding))
		}
		copy(queryVector, embedResp.Embeddings[0].Embedding)
	}

	// Compute cosine similarities and track top-K
	type scoredDoc struct {
		doc   core.Doc
		score float64
	}
	var candidates []scoredDoc

	for _, doc := range r.docs {
		if vectorI, ok := doc.Meta["vector"]; ok {
			vector, ok := vectorI.([]float32)
			if ok && len(vector) == r.dim {
				score := cosineSimilarity(queryVector, vector)
				candidates = append(candidates, scoredDoc{doc: doc, score: score})
			}
		}
	}

	// Sort by score (descending) and take top-K
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	k := req.TopK
	if k <= 0 {
		k = r.topK
	}

	var results []core.Doc
	for i := 0; i < k && i < len(candidates); i++ {
		doc := candidates[i].doc
		doc.Meta["score"] = candidates[i].score
		results = append(results, doc)
	}

	r.log.Debugf("vector retrieval complete", "query", req.Query, "hits", len(results), "total_docs", len(r.docs))

	return core.RetrieveResult{Docs: results}, nil
}

// loadPreEmbedded loads documents with pre-computed vectors from Meta.vector field.
func (r *InMemoryVectorRetriever) loadPreEmbedded(docs []core.Doc) error {
	for _, doc := range docs {
		vectorI, ok := doc.Meta["vector"]
		if !ok {
			return fmt.Errorf("pre-embedded document %q missing vector", doc.ID)
		}
		vector, ok := vectorI.([]float32)
		if !ok || len(vector) == 0 {
			return fmt.Errorf("pre-embedded document %q has invalid vector", doc.ID)
		}
		if r.dim == 0 {
			r.dim = len(vector)
		} else if len(vector) != r.dim {
			return fmt.Errorf("vector dimension mismatch: expected %d, got %d", r.dim, len(vector))
		}
		r.docs[doc.ID] = doc
	}
	r.log.Infof("loaded %d pre-embedded documents, dim=%d", len(docs), r.dim)
	return nil
}

// loadAndEmbedMarkdownFiles loads markdown files, chunks them, embeds each chunk, and stores them.
func (r *InMemoryVectorRetriever) loadAndEmbedMarkdownFiles(opts InMemoryVectorOptions, embedder ai.Embedder) error {
	if embedder == nil {
		return fmt.Errorf("embedder is required for markdown file loading")
	}

	// Load markdown files
	loader := newMarkdownLoader(opts.ChunkSize, opts.ChunkOverlap)
	docs, err := loader.loadMarkdownFiles(opts.MarkdownFiles)
	if err != nil {
		return fmt.Errorf("failed to load markdown files: %w", err)
	}

	if len(docs) == 0 {
		return fmt.Errorf("no documents loaded from markdown files")
	}

	// Set embedding dimension from first response
	firstChunk := docs[0]
	embedReq := &ai.EmbedRequest{
		Input: []*ai.Document{ai.DocumentFromText(firstChunk.Text, nil)},
	}
	embedResp, err := embedder.Embed(context.Background(), embedReq)
	if err != nil {
		return fmt.Errorf("failed to embed first chunk: %w", err)
	}
	if embedResp == nil || len(embedResp.Embeddings) == 0 {
		return fmt.Errorf("embedder returned empty response")
	}

	r.dim = len(embedResp.Embeddings[0].Embedding)
	firstChunk.Meta["vector"] = embedResp.Embeddings[0].Embedding
	r.docs[firstChunk.ID] = firstChunk

	r.log.Infof("loaded and embedded markdown documents, dim=%d, first_doc_id=%s", r.dim, firstChunk.ID)

	// Embed remaining chunks (could be parallelized for efficiency)
	for i := 1; i < len(docs); i++ {
		doc := docs[i]
		embedReq := &ai.EmbedRequest{
			Input: []*ai.Document{ai.DocumentFromText(doc.Text, nil)},
		}
		embedResp, err := embedder.Embed(context.Background(), embedReq)
		if err != nil {
			r.log.Warnf("failed to embed chunk %s: %v", doc.ID, err)
			continue
		}
		if embedResp != nil && len(embedResp.Embeddings) > 0 {
			doc.Meta["vector"] = embedResp.Embeddings[0].Embedding
			r.docs[doc.ID] = doc
		}
	}

	r.log.Infof("completed embedding %d markdown chunks", len(r.docs))
	return nil
}

// cosineSimilarity computes cosine similarity between two float32 vectors.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	dot := float64(0)
	normA := float64(0)
	normB := float64(0)
	for i := 0; i < len(a); i++ {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
