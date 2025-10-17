package core

import (
	"context"

	"github.com/google/mangle/ast"
)

// Stage represents a distinct phase of the processing pipeline where Mangle
// rules can be evaluated. This allows for different sets of logic to be applied
// at different points in the workflow.
type Stage string

const (
	// Pre defines the rule evaluation stage that occurs *before* document retrieval.
	// Rules in this stage typically handle query validation, normalization, expansion,
	// and scoping (e.g., adding metadata filters based on the user's identity).
	Pre Stage = "pre"
	// Post defines the rule evaluation stage that occurs *after* document retrieval
	// and reranking, but *before* the final LLM call. Rules in this stage often
	// enforce entitlements, compatibility, and PII filtering on the retrieved documents.
	Post Stage = "post"
)

// SchemaSource defines a schema to be loaded and parsed into Datalog facts
// for the Mangle engine. This allows rules to query the structure of external
// data sources.
type SchemaSource struct {
	// Type specifies the registered name of the SchemaParser to use (e.g., "jsonschema").
	Type string `yaml:"type"`
	// Path is the file system path to the schema definition file.
	Path string `yaml:"path" path:"resolve"`
}

// MangleOptions provides configuration for initializing the Mangle rule engine provider.
type MangleOptions struct {
	// Path is a slice of file paths or glob patterns pointing to Mangle Datalog
	// (.dlog) rule files that should be loaded into the engine.
	Path []string `yaml:"path" path:"resolve"`
	// SchemaSources is a slice of schemas to be loaded, which will be parsed
	// into facts and added to the engine's knowledge base.
	SchemaSources []SchemaSource `yaml:"schemaSources"`
	// PreProcess is a slice of Mangle transformer names to run in the "pre" stage.
	// Transformers are special rules that can modify the set of available facts.
	PreProcess []string `yaml:"preProcess"`
	// PostProcess is a slice of Mangle transformer names to run in the "post" stage.
	PostProcess []string `yaml:"postProcess"`
	// DefaultConverters specifies whether to include the default set of built-in
	// fact converters, which handle common data types.
	DefaultConverters bool `yaml:"defaultConverters"`
	// FileFirst, if true, indicates that facts from files should be loaded before
	// facts from converters, which can affect rule evaluation if there are conflicts.
	FileFirst bool `yaml:"fileFirst"`
	// Logger receives structured diagnostics from the rules engine while it boots.
	Logger Logger `yaml:"-"`
}

func (o MangleOptions) ProviderName() string { return "mangle" }
func (o MangleOptions) ProviderKind() Kind   { return KindRules }

// RuleSet defines the interface for a rules engine that can evaluate rules
// at different stages of the pipeline. It is the primary point of interaction
// between the orchestrator and the policy engine.
type RuleSet interface {
	// Evaluate runs the configured rules for a given pipeline stage.
	//
	// stage is the pipeline stage (Pre or Post) at which to evaluate rules.
	// q is the incoming query. Rules can read its contents and metadata.
	// a is the current answer object. In the 'pre' stage, this is typically empty.
	// In the 'post' stage, it contains the LLM-generated text and citations.
	// It returns a RuleResult indicating the outcome of the evaluation and
	// an error if the evaluation itself fails.
	Evaluate(stage Stage, q Query, a *Answer) (RuleResult, error)
}

// RuleResult encapsulates the outcome of a rule evaluation. It communicates
// whether the pipeline should proceed, why a decision was made, and what
// changes (mutations) should be applied.
type RuleResult struct {
	// Allowed indicates whether the pipeline is permitted to continue. If false,
	// the orchestrator should halt processing and return a denial.
	Allowed bool
	// Reason provides a human-readable explanation for the outcome, especially
	// if Allowed is false.
	Reason string
	// Mutate is an optional function that applies modifications to the query or
	// answer based on the rule evaluation. This is how rules can add metadata,
	// filter citations, or rewrite queries.
	Mutate func(q *Query, a *Answer)
	// SkippedStages contains a set of stage names that the rules have determined
	// should be skipped in the declarative orchestrator.
	SkippedStages map[string]bool
}

// Querier defines an interface for components that can run arbitrary queries
// against a knowledge base, such as a Mangle Datalog engine. This is a key
// component of the declarative orchestrator, which uses Datalog queries to
// determine its execution flow.
type Querier interface {
	// Query executes a Datalog query string. For each solution found, it invokes
	// the onSolution callback with a map representing the bound variables.
	//
	// ctx is the context for the query execution.
	// query is the Datalog query to be executed (e.g., "foo(X, Y), bar(Y)").
	// onSolution is a callback function that will be called for each result.
	// It returns an error if the query execution fails.
	Query(ctx context.Context, query string, onSolution func(map[string]any) error) error
}

// FlowController defines the interface for an engine that can both evaluate
// policy rules and be queried for workflow definitions. It combines the RuleSet
// and Querier interfaces, making it the core of the declarative orchestrator.
type FlowController interface {
	RuleSet
	Querier
}

// PostRuleEvaluator defines the optional extension interface for rule engines
// that can evaluate post-retrieval logic prior to the LLM stage. Declarative
// orchestrations rely on this hook to enforce policy before generation.
type PostRuleEvaluator interface {
	// Post evaluates post-retrieval rules and returns a PostRuleResult describing
	// any mutations, denials, or metadata emitted by the rule engine.
	//
	// ctx is the evaluation context.
	// q is the normalized query.
	// evidence contains the documents that will be passed to the LLM if allowed.
	// meta carries additional execution metadata such as retrieval scores.
	Post(ctx context.Context, q Query, evidence []Doc, meta map[string]any) (PostRuleResult, error)
}

// PostRuleResult captures the outcome of a PostRuleEvaluator run.
type PostRuleResult struct {
	// Filtered contains the evidence that survived rule evaluation. This slice
	// may be shorter than the input if documents were dropped or redacted.
	Filtered []Doc
	// Denied indicates the pipeline must stop before calling the LLM.
	Denied bool
	// Reason provides the human-readable denial explanation when Denied is true.
	Reason string
	// Meta contains any additional data generated during evaluation, for example
	// audit records or lists of rules that fired.
	Meta map[string]any
}

// FactConverter defines the interface for components that convert Go objects
// into Mangle Datalog facts (ast.Atom). This is a powerful mechanism for
// injecting runtime data—such as the incoming query, user information, or
// retrieved documents—into the Mangle engine so that rules can reason about it.
type FactConverter interface {
	// ToFacts converts the given input object into a slice of Mangle facts.
	//
	// input is the Go object to be converted (e.g., a Query or Answer struct).
	// It returns a slice of ast.Atom representing the facts derived from the
	// input, or an error if the conversion fails.
	ToFacts(input any) ([]ast.Atom, error)

	// Predicates returns a slice of ast.PredicateSym, declaring the Datalog
	// predicates that this converter can generate. This is required for Mangle's
	// static analysis to validate rules.
	Predicates() []ast.PredicateSym
}
