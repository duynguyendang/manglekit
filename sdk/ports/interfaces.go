package ports

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ooda"
)

// KnowledgeStore defines the interface for retrieving knowledge facts from a persistent store.
// This is used for the OODA Orient phase to hydrate context from the knowledge base.
type KnowledgeStore interface {
	// Recall retrieves the top-K most relevant facts for a query.
	// Uses graphID to scope the search to a specific knowledge tier/graph.
	Recall(ctx context.Context, query string, topK int, graphID string) ([]core.Atom, error)

	// GetFacts retrieves facts matching a specific pattern (subject/predicate/object).
	// Uses graphID to scope the search to a specific knowledge tier/graph.
	GetFacts(ctx context.Context, subject, predicate, object, graphID string) ([]core.Quad, error)

	// StreamFacts streams facts matching a pattern using zero-copy iteration.
	// Uses graphID to scope the search to a specific knowledge tier/graph.
	StreamFacts(ctx context.Context, subject, predicate, object, graphID string) func(func(core.Quad) bool)
}

// ReasoningPort provides pure Datalog evaluation capabilities against the underlying graph store.
type ReasoningPort interface {
	// VerifyExecutes evaluates a Datalog query and returns structural Audit results.
	VerifyWithDatalog(ctx context.Context, datalogQuery string) ([]map[string]string, error)
}

// GenerativePort encapsulates interactions with Large Language Models.
type GenerativePort interface {
	// Generate produces structured output based on the assembled CognitiveFrame.
	Generate(ctx context.Context, frame *ooda.CognitiveFrame) (any, error)
}

// GenePoolPort handles fetching and mapping crystallized logic rules.
type GenePoolPort interface {
	// LoadActiveGenes fetches the relevant domain logic bounds for the current phase.
	LoadActiveGenes(ctx context.Context, frame *ooda.CognitiveFrame) ([]ooda.DomainGene, error)
}

// StoragePort guarantees persistence across the Hexagonal system.
type StoragePort interface {
	// SaveTrace records the final result of a CognitiveFrame epoch.
	SaveTrace(ctx context.Context, frame *ooda.CognitiveFrame) error
}

// WorkflowLoader defines the interface for loading workflow definitions.
// This decouples the workflow source (Datalog, YAML, JSON, etc.) from execution.
type WorkflowLoader interface {
	// LoadWorkflow loads a workflow definition by ID.
	LoadWorkflow(ctx context.Context, workflowID string) (*core.WorkflowDef, error)
}

// ConditionEvaluator evaluates workflow edge conditions against current facts.
type ConditionEvaluator interface {
	// EvaluateCondition evaluates a condition string against current context.
	EvaluateCondition(ctx context.Context, condition string, facts map[string]interface{}) (bool, error)
}

// AgentFinder finds available agents by capability/role.
type AgentFinder interface {
	// FindAgentsByRole finds available agents by role.
	FindAgentsByRole(ctx context.Context, role string) ([]string, error)
}

// SessionStateStore defines the interface for storing workflow execution state in memory.
// This is separate from MEB (persistent storage) - state here is transient.
type SessionStateStore interface {
	// Create creates a new workflow instance in session memory.
	Create(ctx context.Context, instance *core.WorkflowInstance) error

	// Get retrieves a workflow instance by session key.
	Get(ctx context.Context, sessionKey string) (*core.WorkflowInstance, error)

	// Update updates an existing workflow instance.
	Update(ctx context.Context, instance *core.WorkflowInstance) error

	// Delete removes a workflow instance from session memory.
	Delete(ctx context.Context, sessionKey string) error

	// Exists checks if a workflow instance exists.
	Exists(ctx context.Context, sessionKey string) bool

	// List returns all workflow instances for a session.
	List(ctx context.Context, sessionID string) ([]*core.WorkflowInstance, error)

	// ClearSession removes all instances for a session (cleanup).
	ClearSession(ctx context.Context, sessionID string) error
}
