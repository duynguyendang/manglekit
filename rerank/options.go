package rerank

// CosineOptions provides typed configuration for a cosine similarity-based reranker.
// This reranker requires an embedder to be configured separately to generate
// vectors for the query and documents.
type CosineOptions struct {
	// TopK specifies the default number of documents to return if not specified
	// in the reranking request.
	TopK int
	// VectorDim is the expected dimensionality of the embedding vectors.
	VectorDim int
}

// ColbertOptions provides a placeholder for configuring a Colbert-style reranker.
// NOTE: This is for a future implementation and is not currently used.
type ColbertOptions struct {
	// TopK specifies the number of documents to return after reranking.
	TopK int
}