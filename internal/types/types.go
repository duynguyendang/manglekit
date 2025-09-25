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
	NormalizedQuery string              `json:"normalized_query"`
	NormalizedTerms []string            `json:"normalized_terms,omitempty"`
	ExpansionTerms  []string            `json:"expansion_terms,omitempty"`
	Entities        map[string][]string `json:"entities,omitempty"`
	Filters         map[string]string   `json:"filters,omitempty"`
	Constraints     ConstraintSet       `json:"constraints"`
	Explanation     string              `json:"explanation,omitempty"`
}

// ConstraintSet captures the structured constraint objects emitted by Mangle.
type ConstraintSet struct {
	Terms      TermConstraints      `json:"terms"`
	Visibility string               `json:"visibility,omitempty"`
	Metadata   []MetadataConstraint `json:"metadata,omitempty"`
}

// TermConstraints groups the must and should term buckets.
type TermConstraints struct {
	Must   []string `json:"must,omitempty"`
	Should []string `json:"should,omitempty"`
}

// MetadataConstraint represents a structured filter to apply against chunk metadata.
type MetadataConstraint struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
	Source   string   `json:"source,omitempty"`
}

// Chunk represents a document chunk with metadata and scoring.
type Chunk struct {
	ID        string                 `json:"id"`
	DocID     string                 `json:"doc_id"`
	Title     string                 `json:"title,omitempty"`
	Text      string                 `json:"text"`
	Snippet   string                 `json:"snippet,omitempty"`
	Embedding []float64              `json:"embedding,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Score     float64                `json:"score"`
}

// Context represents user and system context for processing.
type Context struct {
	UserContext map[string]interface{} `json:"user_context,omitempty"`
	Constraints ConstraintSet          `json:"constraints"`
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

// Reranker defines the fine ranking stage that reorders hybrid results.
type Reranker interface {
	Rerank(ctx context.Context, query *ExpandedQuery, candidates []*Chunk) ([]*Chunk, []Explanation, error)
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
