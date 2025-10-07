package core

import (
	"context"
	"errors"
)

// Doc represents a single document chunk, which is the fundamental unit of
// information for retrieval and generation.
type Doc struct {
	// ID is the unique identifier for the document chunk.
	ID string
	// Source is an identifier for the origin of the document, e.g., a file path or URL.
	Source string
	// URI is a Uniform Resource Identifier to locate the document source.
	URI string
	// Text is the main content of the document chunk.
	Text string
	// Meta is a map of arbitrary metadata associated with the document.
	Meta map[string]any
}

// LocalvecOptions provides configuration for the local vector store.
// Deprecated: This has been moved to the localvec provider's package.
// It is kept here for backward compatibility but will be removed in a future version.
type LocalvecOptions struct {
	// Path to the directory containing markdown files to be indexed.
	Path string
}

// VectorStore is an interface for vector database operations, allowing for
// pluggable vector storage backends.
type VectorStore interface {
	// AddDocuments embeds and adds a slice of documents to the store.
	AddDocuments(ctx context.Context, docs []Doc) error
	// Search retrieves the most relevant documents for a given query vector.
	// It returns a slice of documents, ranked by similarity.
	Search(ctx context.Context, queryVector []float32, topK int, filter map[string]any) ([]Doc, error)
}

// Orchestrator is the central interface for the MangleKit SDK. It defines the
// main entry point for processing a query and returning an answer.
type Orchestrator interface {
	// Run executes the full Mangle-Retrieval-Rerank-LLM pipeline for a given query.
	//
	// ctx is the context for the operation.
	// q is the user's query.
	// It returns a final Answer or an error if the process fails.
	Run(ctx context.Context, q Query) (Answer, error)

	// Retriever returns the retriever instance configured for the orchestrator.
	// The return type is `any` to avoid a circular dependency with the `retrieve` package.
	// The caller is expected to perform a type assertion to `retrieve.Retriever`.
	Retriever() any
}

// Query represents a user's request to the orchestrator.
type Query struct {
	// Text is the natural language query string from the user.
	Text string `json:"text"`
	// Meta is a map for arbitrary user-supplied metadata that can be used by
	// rules or other pipeline components.
	Meta map[string]any `json:"meta,omitempty"`
}

// Answer represents the final result of a query processed by the orchestrator.
type Answer struct {
	// Text is the generated textual answer to the query.
	Text string `json:"text"`
	// Citations is a slice of references to the source documents used to
	// generate the answer.
	Citations []Citation `json:"citations,omitempty"`
	// Meta is a map containing operational metadata, such as timings,
	// token usage, confidence scores, or debugging information.
	Meta map[string]any `json:"meta,omitempty"`
}

// Citation is a reference to a source document that supports the Answer.
type Citation struct {
	// ID is the unique identifier of the cited document chunk.
	ID string `json:"id"`
	// Source is the identifier for the origin of the document (e.g., file path).
	Source string `json:"source"`
	// URI is a link to the original source document.
	URI string `json:"uri,omitempty"`
	// Snippet is the specific text excerpt from the document relevant to the answer.
	Snippet string `json:"snippet,omitempty"`
	// Score is the relevance or confidence score of the citation.
	Score float64 `json_:"score,omitempty"`
}

// Options configures the MangleKit SDK's orchestrator. The fields are left
// as `any` to be populated by the builder, avoiding circular dependencies.
type Options struct {
	// Retriever is the component responsible for fetching relevant documents.
	// Expected type: retrieve.Retriever
	Retriever any
	// Reranker is an optional component to re-score and re-order retrieved documents.
	// Expected type: rerank.Reranker
	Reranker any
	// LLM is the language model client used for generation.
	// Expected type: llm.Client
	LLM any
	// Rules is the set of Mangle rules to be applied during the pre- and post-processing stages.
	Rules RuleSet
	// TopK is the number of documents to retrieve initially.
	TopK int
	// MaxTokens is the maximum number of tokens for the LLM response.
	MaxTokens int
	// FallbackThreshold is a confidence score below which a fallback answer is used.
	FallbackThreshold float64
	// Obs provides hooks for observability (logging, tracing, metrics).
	Obs Observability
}

// Observability provides interfaces for integrating logging, tracing, and metrics
// into the MangleKit pipeline.
type Observability struct {
	// Logger is the structured logger instance.
	Logger Logger
	// Tracer is the distributed tracing instance.
	Tracer Tracer
	// Meter is the metrics recording instance.
	Meter Meter
}

// Logger defines a basic interface for structured logging.
type Logger interface {
	// Info logs an informational message with key-value pairs.
	Info(msg string, kv ...any)
	// Error logs an error message with key-value pairs.
	Error(msg string, kv ...any)
}

// Tracer defines a basic interface for creating spans for distributed tracing,
// compatible with systems like OpenTelemetry.
type Tracer interface {
	// StartSpan begins a new trace span and returns a function to end it.
	StartSpan(name string) (end func(attrs ...any))
}

// Meter defines a basic interface for recording metrics.
type Meter interface {
	// Record captures a metric with a given value and attributes.
	Record(metric string, value float64, attrs ...any)
}

var (
	// ErrInvalidOptions is returned when the SDK is initialized with missing
	// or invalid options, such as a nil Retriever or LLM.
	ErrInvalidOptions = errors.New("invalid_options")
	// ErrNoEvidence is returned when the retriever finds no documents or evidence
	// to answer the query after filtering.
	ErrNoEvidence = errors.New("insufficient_evidence")
	// ErrDenied is returned when a Mangle rule explicitly denies the request,
	// halting the pipeline.
	ErrDenied = errors.New("rules_denied")
)