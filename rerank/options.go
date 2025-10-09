package rerank

// CosineOptions provides typed configuration for a reranker that operates by
// calculating the cosine similarity between the query embedding and each of the
// document embeddings. This type of reranker requires an embedder to be
// configured separately in the builder to generate the necessary vectors.
type CosineOptions struct {
	// TopK specifies the default number of top-scoring documents to return if a
	// different limit is not specified in the reranking request itself.
	TopK int `json:"topK,omitempty"`
	// VectorDim is the expected dimensionality of the embedding vectors. While
	// not always used directly by the reranker logic, it can be useful for
	// validation or pre-allocation.
	VectorDim int `json:"vectorDim,omitempty"`
}

// ColbertOptions provides a placeholder for configuring a Colbert-style reranker,
// which uses a more advanced, token-level interaction model for scoring.
//
// NOTE: This is for a future implementation and is not currently used.
type ColbertOptions struct {
	// TopK specifies the number of documents to return after reranking.
	TopK int `json:"topK,omitempty"`
}
