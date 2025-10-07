package core

import (
	"context"

	"github.com/google/mangle/ast"
)

// Stage represents the phase of the pipeline where Mangle rules are evaluated.
type Stage string

const (
	// Pre defines the rule evaluation stage that occurs before document retrieval.
	// Rules in this stage typically handle query validation, normalization, and scoping.
	Pre Stage = "pre"
	// Post defines the rule evaluation stage that occurs after document retrieval
	// and reranking, but before the final LLM call. Rules in this stage often
	// enforce entitlements, compatibility, and PII filtering on the retrieved documents.
	Post Stage = "post"
)

// SchemaSource defines a schema to be loaded and parsed into Datalog facts
// for the Mangle engine.
type SchemaSource struct {
	// Type specifies the parser to use for the schema (e.g., "jsonschema").
	Type string `yaml:"type"`
	// Path is the file path to the schema definition.
	Path string `yaml:"path"`
}

// MangleOptions provides configuration for initializing the Mangle rule engine provider.
type MangleOptions struct {
	// Path is a slice of file paths or glob patterns pointing to Mangle Datalog (.dlog) rule files.
	Path []string `yaml:"path"`
	// SchemaSources is a slice of schemas to be loaded.
	SchemaSources []SchemaSource `yaml:"schemaSources"`
	// PreProcess is a slice of Mangle transformer names to run in the "pre" stage.
	PreProcess []string
	// PostProcess is a slice of Mangle transformer names to run in the "post" stage.
	PostProcess []string
	// DefaultConverters specifies whether to include the default set of fact converters.
	DefaultConverters bool
}

// RuleSet defines the interface for a rules engine that can evaluate rules
// at different stages of the pipeline.
type RuleSet interface {
	// Evaluate runs the configured rules for a given pipeline stage.
	//
	// stage is the pipeline stage (Pre or Post) at which to evaluate rules.
	// q is the incoming query, which can be read or mutated by the rules.
	// a is the current answer, which can be read or mutated by the rules.
	// It returns a RuleResult indicating whether the pipeline should proceed and
	// an error if the evaluation fails.
	Evaluate(stage Stage, q Query, a *Answer) (RuleResult, error)
}

// RuleResult encapsulates the outcome of a rule evaluation.
type RuleResult struct {
	// Allowed indicates whether the pipeline is permitted to continue. If false,
	// the pipeline should be halted.
	Allowed bool
	// Reason provides an explanation for the outcome, especially if Allowed is false.
	Reason string
	// Mutate is an optional function that applies modifications to the query or
	// answer based on the rule evaluation.
	Mutate func(q *Query, a *Answer)
	// SkippedStages contains a set of stage names that the rules have determined should be skipped.
	SkippedStages map[string]bool
}

// Querier defines an interface for components that can run arbitrary queries
// against a knowledge base, like a Mangle Datalog engine. This is used by the
// DeclarativeOrchestrator to fetch its execution plan.
type Querier interface {
	// Query executes a Datalog query and streams the results to the onSolution callback.
	Query(ctx context.Context, query string, onSolution func(map[string]any) error) error
}

// FlowController defines the interface for an engine that can both evaluate
// policy rules and be queried for workflow definitions. It combines the RuleSet
// and Querier interfaces.
type FlowController interface {
	RuleSet
	Querier
}

// FactConverter defines the interface for components that convert Go objects
// into Mangle Datalog facts (ast.Atom). This allows runtime data to be injected
// into the Mangle engine for evaluation.
type FactConverter interface {
	// ToFacts converts the given input object into a slice of Mangle facts.
	//
	// input is the Go object to be converted.
	// It returns a slice of ast.Atom or an error if the conversion fails.
	ToFacts(input any) ([]ast.Atom, error)

	// Predicates returns a slice of ast.PredicateSym, declaring the Datalog
	// predicates that this converter can generate.
	Predicates() []ast.PredicateSym
}