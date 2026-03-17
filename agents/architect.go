// Package agents provides the Architect Agent implementation
// using the OODA Chassis SDK with Dual Memory architecture.
package agents

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/duynguyendang/manglekit/agents/tools"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/multiagent"
	"github.com/duynguyendang/manglekit/sdk/ooda"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

// ArchitectAgent implements the Architect Agent using OODA Chassis
type ArchitectAgent struct {
	frame          *ooda.CognitiveFrame
	engine         *multiagent.AgentSystem
	workflowID     string
	sessionID      string
	transientStore ports.TransientStore
	knowledgeStore ports.KnowledgeStore
	registry       *ooda.Registry
}

// NewArchitectAgent creates a new Architect Agent
func NewArchitectAgent(ctx context.Context, engine *multiagent.AgentSystem, workflowID string) (*ArchitectAgent, error) {
	// Create session ID
	sessionID := fmt.Sprintf("architect-%d", time.Now().UnixNano())

	// Create registry and register tools
	registry := ooda.NewRegistry()
	tools.RegisterArchitectTools(registry)

	// Create transient store for session state
	transientStore := tools.NewTransientStore()

	// Create the cognitive frame using Builder
	frame := ooda.NewBuilder().
		WithSessionID(sessionID).
		WithWorkflowID(workflowID).
		WithRegistry(registry).
		Build()

	return &ArchitectAgent{
		frame:          frame,
		engine:         engine,
		workflowID:     workflowID,
		sessionID:      sessionID,
		transientStore: transientStore,
		registry:       registry,
	}, nil
}

// Run executes the Architect Agent workflow
func (a *ArchitectAgent) Run(ctx context.Context, input core.Envelope) (*core.WorkflowResult, error) {
	// Load workflow definition from Datalog
	loader := multiagent.NewDatalogWorkflowLoader(a.engine)
	workflowDef, err := loader.LoadWorkflow(ctx, a.workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	slog.Info("loaded workflow", "workflow", workflowDef.Name, "nodes", len(workflowDef.Nodes))

	// Create the hydrated executor
	executor := multiagent.NewHydratedWorkflowExecutor(workflowDef).
		WithSessionStore(newSessionStoreAdapter(a.transientStore), a.sessionID)

	// Execute the workflow with session
	result, instance, err := executor.ExecuteWithSession(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	slog.Info("workflow completed",
		"status", result.Status,
		"session", a.sessionID,
		"completed_nodes", len(instance.CompletedNodes),
	)

	return result, nil
}

// RunOODA executes a single OODA cycle (for step-by-step execution)
func (a *ArchitectAgent) RunOODA(ctx context.Context, input string) (*ooda.CognitiveFrame, error) {
	// Build the cognitive frame using existing frame
	a.frame.Input = input

	// Run the OODA loop
	_, err := ooda.Run(ctx, a.frame)
	if err != nil {
		return a.frame, fmt.Errorf("OODA loop failed: %w", err)
	}

	// Log audit summary
	slog.Info("OODA cycle completed",
		"phase", a.frame.Phase,
		"audit", a.frame.GetAuditSummary(),
	)

	return a.frame, nil
}

// GetSessionState returns the current session state from TransientStore
func (a *ArchitectAgent) GetSessionState(ctx context.Context) (map[string]interface{}, error) {
	state := make(map[string]interface{})

	// Get all facts from transient store
	facts, err := a.transientStore.GetAll(ctx, a.sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session state: %w", err)
	}

	for _, fact := range facts {
		state[fact.Subject+":"+fact.Predicate] = fact.Object
	}

	return state, nil
}

// ClearSession clears the session state
func (a *ArchitectAgent) ClearSession(ctx context.Context) error {
	return a.transientStore.ClearSession(ctx, a.sessionID)
}

// SetKnowledgeStore sets the knowledge store (MEB adapter)
func (a *ArchitectAgent) SetKnowledgeStore(store ports.KnowledgeStore) {
	a.knowledgeStore = store
	a.frame.KnowledgeStore = store
}

// GetAuditTrail returns the audit trail from the last run
func (a *ArchitectAgent) GetAuditTrail() string {
	if a.frame == nil {
		return "No frame available"
	}
	return a.frame.GetAuditSummary()
}

// sessionStoreAdapter adapts TransientStore to SessionStateStore
type sessionStoreAdapter struct {
	store ports.TransientStore
}

func newSessionStoreAdapter(store ports.TransientStore) *sessionStoreAdapter {
	return &sessionStoreAdapter{store: store}
}

func (s *sessionStoreAdapter) Create(ctx context.Context, instance *core.WorkflowInstance) error {
	return nil // Transient store doesn't need explicit create
}

func (s *sessionStoreAdapter) Get(ctx context.Context, sessionKey string) (*core.WorkflowInstance, error) {
	// Parse sessionKey to get sessionID and workflowID
	// For now, return a simple instance
	return core.NewWorkflowInstance(sessionKey, sessionKey), nil
}

func (s *sessionStoreAdapter) Update(ctx context.Context, instance *core.WorkflowInstance) error {
	return nil
}

func (s *sessionStoreAdapter) Delete(ctx context.Context, sessionKey string) error {
	return nil
}

func (s *sessionStoreAdapter) Exists(ctx context.Context, sessionKey string) bool {
	return true
}

func (s *sessionStoreAdapter) List(ctx context.Context, sessionID string) ([]*core.WorkflowInstance, error) {
	return nil, nil
}

func (s *sessionStoreAdapter) ClearSession(ctx context.Context, sessionID string) error {
	return s.store.ClearSession(ctx, sessionID)
}
