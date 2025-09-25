// Package types defines the core data structures and interfaces for Manglekit.
// This package provides the foundational types used across the Mangle-Genkit integration.
package types

import (
	"context"
	"time"
)

// QueryInput represents the initial user query with context.
type QueryInput struct {
	Query       string                 `json:"query"`
	UserContext map[string]interface{} `json:"user_context,omitempty"`
	Intent      *IntentResult          `json:"intent,omitempty"`
}

// ExpandedQuery represents the processed query after Mangle-Pre processing.
type ExpandedQuery struct {
	NormalizedQuery string            `json:"normalized_query"`
	ExpansionTerms  []string          `json:"expansion_terms,omitempty"`
	Filters         map[string]string `json:"filters,omitempty"`
	Explanation     string            `json:"explanation,omitempty"`
}

// Chunk represents a document chunk with metadata and scoring.
type Chunk struct {
	ID        string                 `json:"id"`
	DocID     string                 `json:"doc_id"`
	Text      string                 `json:"text"`
	Snippet   string                 `json:"snippet,omitempty"`
	Embedding []float64              `json:"embedding,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Score     float64                `json:"score"`
}

// Context represents user and system context for processing.
type Context struct {
	UserContext map[string]interface{} `json:"user_context,omitempty"`
}

// Explanation represents why a document was processed in a certain way.
type Explanation struct {
	Type      string    `json:"type"`   // "validation", "filter", "redaction", etc.
	Rule      string    `json:"rule"`   // The rule that was applied
	Action    string    `json:"action"` // "retained", "discarded", "modified"
	Reason    string    `json:"reason,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Response represents the final response from the system.
type Response struct {
	Answer       string        `json:"answer"`
	Citations    []string      `json:"citations,omitempty"`
	Explanations []Explanation `json:"explanations,omitempty"`
	Metadata     interface{}   `json:"metadata,omitempty"`
}

// IntentResult captures the output of the Genkit intent and NER parser stage.
type IntentResult struct {
	Intent      string              `json:"intent"`
	Confidence  float64             `json:"confidence,omitempty"`
	Entities    map[string][]string `json:"entities,omitempty"`
	Explanation string              `json:"explanation,omitempty"`
}

// Processor interface defines the Mangle rule engine operations.
type Processor interface {
	// PreProcess normalizes, validates, and expands the query
	PreProcess(input *QueryInput) (*ExpandedQuery, error)

	// PostProcess validates, filters, and redacts chunks
	PostProcess(chunks []*Chunk, ctx *Context) ([]*Chunk, *[]Explanation)
}

// Retriever interface defines the hybrid search operations.
type Retriever interface {
	// Search performs hybrid search with metadata filtering
	Search(ctx context.Context, query *ExpandedQuery, filters map[string]string) ([]*Chunk, error)
}

// BM25Config holds configuration for the BM25 retriever.
type BM25Config struct {
	TopK int `yaml:"topK"`
}

// DenseConfig holds configuration for the dense retriever.
type DenseConfig struct {
	TopK int `yaml:"topK"`
}

// RerankConfig holds configuration for the reranker.
type RerankConfig struct {
	TopK int `yaml:"topK"`
}

// RetrievalConfig holds configuration for the retrieval components.
type RetrievalConfig struct {
	Path   string `yaml:"path"`
	Hybrid struct {
		BM25  BM25Config  `yaml:"bm25"`
		Dense DenseConfig `yaml:"dense"`
	} `yaml:"hybrid"`
}

// LLMConfig holds the configuration for the LLM Gateway.
type LLMConfig struct {
	Provider string `json:"provider"` // "openai" or "ollama"
	Model    string `json:"model"`
	APIKey   string `json:"apiKey,omitempty"`
}

// Gateway interface defines the LLM operations.
type Gateway interface {
	// Generate creates a response using the provided context
	Generate(ctx context.Context, prompt string, chunks []*Chunk) (*Response, error)
}

// Orchestrator interface defines the main workflow coordination.
type Orchestrator interface {
	// RunFlow executes the complete Sandwich pattern workflow
	RunFlow(ctx context.Context, input *QueryInput) (*Response, error)
}

// IntentParser defines the Genkit-based intent and entity extraction stage.
type IntentParser interface {
	// Parse analyses a user query and returns detected intent and entities.
	Parse(ctx context.Context, input *QueryInput) (*IntentResult, error)
}
