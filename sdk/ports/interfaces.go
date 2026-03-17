package ports

import (
	"context"

	"github.com/duynguyendang/manglekit/core"
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
	// The frame parameter is an interface to avoid circular imports.
	Generate(ctx context.Context, frame any) (any, error)
}

// GenePoolPort handles fetching and mapping crystallized logic rules.
type GenePoolPort interface {
	// LoadActiveGenes fetches the relevant domain logic bounds for the current phase.
	// The frame parameter is an interface to avoid circular imports.
	LoadActiveGenes(ctx context.Context, frame any) ([]any, error)
}

// StoragePort guarantees persistence across the Hexagonal system.
type StoragePort interface {
	// SaveTrace records the final result of a CognitiveFrame epoch.
	// The frame parameter is an interface to avoid circular imports.
	SaveTrace(ctx context.Context, frame any) error
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

// TransientStore defines the interface for storing transient coordination facts
// during OODA loop execution. Facts in this store are isolated by SessionID
// and are NOT persisted to MEB (BadgerDB).
// This is used for short-term memory: current_node, agent_status, workflow progress, etc.
type TransientStore interface {
	// Put stores a fact in the session's transient memory.
	Put(ctx context.Context, sessionID, key string, fact *TransientFact) error

	// Get retrieves a fact from the session's transient memory.
	Get(ctx context.Context, sessionID, key string) (*TransientFact, error)

	// GetAll retrieves all facts for a session.
	GetAll(ctx context.Context, sessionID string) ([]*TransientFact, error)

	// Delete removes a fact from the session's transient memory.
	Delete(ctx context.Context, sessionID, key string) error

	// ClearSession removes all facts for a session.
	ClearSession(ctx context.Context, sessionID string) error

	// ToAtoms converts all session facts to core.Atom format for Brain evaluation.
	ToAtoms(ctx context.Context, sessionID string) ([]core.Atom, error)

	// ToQuads converts all session facts to core.Quad format for storage.
	ToQuads(ctx context.Context, sessionID string) ([]core.Quad, error)
}

// TransientFact represents a fact stored in transient session memory.
type TransientFact struct {
	Subject   string
	Predicate string
	Object    string
	Graph     string
	CreatedAt string // ISO8601 timestamp
}
