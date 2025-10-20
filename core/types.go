package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/firebase/genkit/go/ai"
)

// Message represents a single message in a conversation from a specific role.
type Message struct {
	Role    string `json:"role"` // "user" or "model"
	Content string `json:"content"`
}

// ConversationHistory stores the list of messages for a session.
type ConversationHistory struct {
	Messages []Message `json:"messages"`
}

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

func (o LocalvecOptions) ProviderName() string { return "localvec" }
func (o LocalvecOptions) ProviderKind() Kind   { return KindVectorStore }

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
	Search(ctx context.Context, queryText string, queryVector []float32, topK int, filter map[string]any) ([]Doc, error)
}

// Orchestrator is the central behavioral interface for the MangleKit SDK. It is
// a pure executor, responsible for running a configured pipeline but not for
// exposing the components within it. Typed components should be returned by the
// builder at construction time.
type Orchestrator interface {
	// Execute runs the full processing pipeline for a given query.
	//
	// ctx is the context for the entire operation.
	// sessionID is the unique identifier for the session.
	// q is the user's query to be processed.
	// It returns a final Answer containing the generated text and citations,
	// or an error if any part of the process fails.
	Execute(ctx context.Context, sessionID string, q Query) (Answer, error)

	// Close releases any resources (such as API clients) associated with the
	// orchestrator's components. It should be invoked when the orchestrator is
	// no longer needed. Implementations should be idempotent.
	Close(ctx context.Context) error
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

// Resolved is the final, strongly-typed container of all built components and
// configuration settings. It is passed to the orchestrator factory, ensuring
// that orchestrators receive their dependencies in a type-safe manner, free
// from `any` types and runtime assertions.
type Resolved struct {
	Retrievers     map[string]Retriever
	VectorStores   map[string]VectorStore
	Rerankers      map[string]Reranker
	Rules          map[string]RuleSet
	LLMs           map[string]LLMClient
	Embedders      map[string]ai.Embedder
	StateProviders map[string]StateProvider

	Obs               Observability
	TopK              int
	MaxTokens         int
	FallbackThreshold float64
	Closers           []ResourceCloser
}

// GetToolByName finds a component by its registered name and returns it wrapped
// in a `core.Tool` adapter. This allows the declarative orchestrator to look up
// its steps in a generic, type-safe way.
func (r *Resolved) GetToolByName(name string) (Tool, error) {
	if t, ok := r.Retrievers[name]; ok {
		return &RetrieverTool{R: t}, nil
	}
	if t, ok := r.Rerankers[name]; ok {
		return &RerankerTool{Rr: t}, nil
	}
	if t, ok := r.LLMs[name]; ok {
		return &LLMTool{Llm: t}, nil
	}
	// Note: VectorStores, Embedders, and StateProviders are not currently adapted
	// as tools because they don't represent standalone pipeline steps.
	return nil, fmt.Errorf("tool with name '%s' not found", name)
}

// OptionsLike is a temporary struct to hold global settings during the build process.
// It will be replaced by a more robust configuration management system in the future.
type OptionsLike struct {
	TopK              int
	MaxTokens         int
	FallbackThreshold float64
	Obs               Observability
	ResourceClosers   []ResourceCloser
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

// Logger defines a vendor-neutral interface for structured logging. The
// methods follow a "message + key/value" calling convention so callers can
// attach context without committing to any specific backend semantics.
type Logger interface {
	// Debugf records verbose diagnostic information about control flow or
	// intermediate state. Arguments are interpreted as key/value pairs.
	Debugf(msg string, kv ...any)
	// Infof records high-level lifecycle events such as component start or
	// stop. Arguments are interpreted as key/value pairs.
	Infof(msg string, kv ...any)
	// Warnf records recoverable issues that deserve operator attention but
	// do not stop execution.
	Warnf(msg string, kv ...any)
	// Errorf records failures. Implementations should treat the "error"
	// key specially when present.
	Errorf(msg string, kv ...any)
	// With returns a child logger that automatically appends the supplied
	// key/value pairs to every log record.
	With(kv ...any) Logger
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

// ResourceCloser defines a cleanup callback that releases external resources.
// The provided context can be used by the closer to respect deadlines or propagate
// cancellation signals while shutting down the resource.
type ResourceCloser func(ctx context.Context) error

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
