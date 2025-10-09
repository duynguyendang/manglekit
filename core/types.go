package core

import (
	"context"
	"errors"
)

// Doc represents a single document chunk, which is the fundamental unit of
// information for retrieval and generation. It contains the content as well as
// metadata about its origin.
type Doc struct {
	// ID is the unique identifier for the document chunk.
	ID string
	// Source is an identifier for the origin of the document, e.g., a file path or URL.
	Source string
	// URI is a Uniform Resource Identifier that provides a locator for the document source.
	URI string
	// Text is the main content of the document chunk.
	Text string
	// Meta is a map of arbitrary metadata associated with the document,
	// such as author, creation date, or other annotations.
	Meta map[string]any
}

// LocalvecOptions provides configuration for the local vector store.
//
// Deprecated: This has been moved to the localvec provider's package.
// It is kept here for backward compatibility but will be removed in a future version.
type LocalvecOptions struct {
	// Path is the file system path to the directory containing markdown files to be indexed.
	Path string `yaml:"path" path:"resolve"`
}

// VectorStore defines the standard interface for vector database operations,
// allowing for pluggable vector storage backends. Implementations of this
// interface handle the storage and retrieval of documents based on vector similarity.
type VectorStore interface {
	// AddDocuments embeds and adds a slice of documents to the vector store.
	// This method is responsible for processing the documents, generating vectors
	// if necessary, and indexing them for future searches.
	//
	// ctx is the context for the operation.
	// docs is a slice of documents to be added to the store.
	// It returns an error if the documents could not be added.
	AddDocuments(ctx context.Context, docs []Doc) error

	// Search retrieves the most relevant documents for a given query vector.
	//
	// ctx is the context for the operation.
	// queryVector is the vector representation of the search query.
	// topK specifies the maximum number of documents to return.
	// filter is an optional map of metadata to filter documents before searching.
	// It returns a slice of documents ranked by similarity and an error if the search fails.
	Search(ctx context.Context, queryVector []float32, topK int, filter map[string]any) ([]Doc, error)
}

// Orchestrator is the central interface for the MangleKit SDK. It defines the
// main entry point for processing a query and returning an answer. Implementations
// of this interface, like the "Sandwich" or "Declarative" orchestrators, manage
// the flow of data through the various components of the system.
type Orchestrator interface {
	// Run executes the full processing pipeline for a given query. This typically
	// involves pre-processing rules, document retrieval, reranking, LLM generation,
	// and post-processing rules.
	//
	// ctx is the context for the entire operation.
	// q is the user's query to be processed.
	// It returns a final Answer containing the generated text and citations,
	// or an error if any part of the process fails.
	Run(ctx context.Context, q Query) (Answer, error)

	// Retriever returns the retriever instance configured for the orchestrator.
	// This allows for runtime operations on the retriever, such as adding or updating
	// documents in an updatable retriever. The return type is `any` to avoid a
	// circular dependency with the `retrieve` package; the caller is expected to
	// perform a type assertion to `retrieve.Retriever`.
	Retriever() any
}

// Query represents a user's request to the orchestrator. It contains the query
// text and any associated metadata.
type Query struct {
	// Text is the natural language query string from the user.
	Text string `json:"text"`
	// Meta is a map for arbitrary user-supplied metadata. This can be used by
	// rules or other pipeline components to influence their behavior, for example,
	// by passing user identity or session information.
	Meta map[string]any `json:"meta,omitempty"`
}

// Answer represents the final result of a query processed by the orchestrator.
// It includes the generated text, supporting citations, and operational metadata.
type Answer struct {
	// Text is the generated textual answer to the query, synthesized by the LLM.
	Text string `json:"text"`
	// Citations is a slice of references to the source documents that were used
	// to generate the answer. This provides traceability and allows users to
	// verify the information.
	Citations []Citation `json:"citations,omitempty"`
	// Meta is a map containing operational metadata about the pipeline execution,
	// such as component timings, token usage, confidence scores, or debugging information.
	Meta map[string]any `json:"meta,omitempty"`
}

// Citation is a reference to a source document that supports the Answer. It helps
// ground the generated response in verifiable evidence.
type Citation struct {
	// ID is the unique identifier of the cited document chunk.
	ID string `json:"id"`
	// Source is the identifier for the origin of the document (e.g., file path).
	Source string `json:"source"`
	// URI is a link to the original source document, allowing for easy access.
	URI string `json:"uri,omitempty"`
	// Snippet is the specific text excerpt from the document that is most
	// relevant to the answer.
	Snippet string `json:"snippet,omitempty"`
	// Score is the relevance or confidence score of the citation, often produced
	// by a retriever or reranker.
	Score float64 `json:"score,omitempty"`
}

// Options configures the MangleKit SDK's orchestrator. It serves as a container
// for all the components and settings that define a pipeline. The fields are
// left as `any` to be populated by the builder, which avoids circular dependencies
// between packages.
type Options struct {
	// Retriever is the component responsible for fetching relevant documents.
	// The builder ensures this is of type `retrieve.Retriever`.
	Retriever any
	// Reranker is an optional component to re-score and re-order retrieved documents
	// for better relevance before passing them to the LLM.
	// The builder ensures this is of type `rerank.Reranker`.
	Reranker any
	// LLM is the language model client used for generating the final answer.
	// The builder ensures this is of type `llm.Client`.
	LLM any
	// Rules is the engine responsible for evaluating Mangle Datalog rules at
	// different stages of the pipeline.
	Rules RuleSet
	// TopK is the number of documents to retrieve from the retriever.
	TopK int
	// MaxTokens is the maximum number of tokens to generate in the LLM response.
	MaxTokens int
	// FallbackThreshold is a confidence score (often from the reranker) below which
	// the pipeline may exit early and return a fallback answer.
	FallbackThreshold float64
	// Obs provides hooks for observability, including logging, tracing, and metrics.
	Obs Observability
}

// Observability provides a set of interfaces for integrating logging, tracing,
// and metrics into the MangleKit pipeline. This allows for detailed monitoring
// and debugging of the system's behavior.
type Observability struct {
	// Logger is the structured logger instance used for recording events.
	Logger Logger
	// Tracer is the distributed tracing instance for creating and managing spans.
	Tracer Tracer
	// Meter is the metrics recording instance for capturing performance data.
	Meter Meter
}

// Logger defines a basic interface for structured logging, allowing for the
// integration of various logging libraries like slog, zap, or logrus.
type Logger interface {
	// Info logs an informational message with a series of key-value pairs
	// providing context.
	Info(msg string, kv ...any)
	// Error logs an error message with a series of key-value pairs.
	Error(msg string, kv ...any)
}

// Tracer defines a basic interface for creating spans for distributed tracing.
// It is designed to be compatible with systems like OpenTelemetry.
type Tracer interface {

	// StartSpan begins a new trace span with the given name.
	// It returns a function that, when called, ends the span. The end function
	// can optionally accept attributes to be added to the span upon completion.
	StartSpan(name string) (end func(attrs ...any))
}

// Meter defines a basic interface for recording metrics, suitable for integration
// with monitoring systems like Prometheus or OpenTelemetry Metrics.
type Meter interface {
	// Record captures a measurement for a named metric.
	//
	// metric is the name of the metric (e.g., "manglekit.retrieve_ms").
	// value is the numerical value to record.
	// attrs is a sequence of key-value pairs for labeling the metric.
	Record(metric string, value float64, attrs ...any)
}

var (
	// ErrInvalidOptions is returned when the SDK is initialized with missing
	// or invalid options, such as a nil Retriever or LLM, or when component
	// types are mismatched.
	ErrInvalidOptions = errors.New("invalid_options")
	// ErrNoEvidence is returned when the retriever finds no documents or evidence
	// to answer the query, either because none exist or because all candidates
	// were filtered out by rules or other mechanisms.
	ErrNoEvidence = errors.New("insufficient_evidence")
	// ErrDenied is returned when a Mangle rule evaluation explicitly denies the
	// request, halting the pipeline as a matter of policy.
	ErrDenied = errors.New("rules_denied")
)
