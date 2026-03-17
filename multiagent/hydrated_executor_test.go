package multiagent

import (
	"context"
	"fmt"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

func TestHydratedWorkflowExecutor_NoDatalog(t *testing.T) {
	ctx := context.Background()

	workflowDef := &core.WorkflowDef{
		ID:      "test-workflow",
		Name:    "Test Workflow",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"start": {
				ID:        "start",
				AgentRole: "planner",
				TaskType:  "agent",
			},
			"execute": {
				ID:        "execute",
				AgentRole: "executor",
				TaskType:  "agent",
			},
			"end": {
				ID:        "end",
				AgentRole: "reviewer",
				TaskType:  "agent",
			},
		},
		Edges: []core.EdgeDef{
			{From: "start", To: "execute"},
			{From: "execute", To: "end"},
		},
	}

	executor := NewHydratedWorkflowExecutor(workflowDef)

	result, err := executor.Execute(ctx, "test input")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if result.Status != core.WorkflowStatusCompleted {
		t.Errorf("expected completed, got %s", result.Status)
	}

	if len(result.NodeResults) != 3 {
		t.Errorf("expected 3 node results, got %d", len(result.NodeResults))
	}

	t.Logf("Workflow completed: %s", result.Status)
	for nodeID, nodeResult := range result.NodeResults {
		t.Logf("  Node %s: output=%v, agent=%s", nodeID, nodeResult.Output, nodeResult.AgentID)
	}
}

func TestHydratedWorkflowExecutor_WithConditionals(t *testing.T) {
	ctx := context.Background()

	workflowDef := &core.WorkflowDef{
		ID:      "conditional-workflow",
		Name:    "Conditional Workflow",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"start": {
				ID:        "start",
				AgentRole: "planner",
				TaskType:  "agent",
			},
			"branch_a": {
				ID:        "branch_a",
				AgentRole: "executor",
				TaskType:  "agent",
			},
			"branch_b": {
				ID:        "branch_b",
				AgentRole: "reviewer",
				TaskType:  "agent",
			},
			"end": {
				ID:        "end",
				AgentRole: "reviewer",
				TaskType:  "agent",
			},
		},
		Edges: []core.EdgeDef{
			{From: "start", To: "branch_a"},
			{From: "start", To: "branch_b"},
			{From: "branch_a", To: "end", Condition: `context("output") = "a"`},
			{From: "branch_b", To: "end", Condition: `context("output") = "b"`},
		},
	}

	mockEvaluator := &MockConditionEvaluator{
		results: map[string]bool{
			`context("output") = "a"`: true,
		},
	}

	executor := NewHydratedWorkflowExecutor(workflowDef).
		WithConditionEvaluator(mockEvaluator)

	result, err := executor.Execute(ctx, "test input")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	t.Logf("Conditional workflow status: %s", result.Status)
}

func TestHydratedWorkflowExecutor_Validation(t *testing.T) {
	workflowDef := &core.WorkflowDef{
		ID:      "validation-test",
		Name:    "Validation Test",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"node1": {ID: "node1", AgentRole: "planner"},
			"node2": {ID: "node2", AgentRole: "executor"},
		},
		Edges: []core.EdgeDef{
			{From: "node1", To: "node2"},
		},
	}

	err := workflowDef.Validate()
	if err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestHydratedWorkflowExecutor_ValidationWithBadEdge(t *testing.T) {
	workflowDef := &core.WorkflowDef{
		ID:      "bad-edge-test",
		Name:    "Bad Edge Test",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"node1": {ID: "node1", AgentRole: "planner"},
		},
		Edges: []core.EdgeDef{
			{From: "node1", To: "nonexistent"},
		},
	}

	err := workflowDef.Validate()
	if err == nil {
		t.Error("expected validation error for bad edge")
	}
}

func TestHydratedWorkflowExecutor_FindStartNode(t *testing.T) {
	workflowDef := &core.WorkflowDef{
		ID:      "start-node-test",
		Name:    "Start Node Test",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"a": {ID: "a", AgentRole: "planner"},
			"b": {ID: "b", AgentRole: "executor"},
			"c": {ID: "c", AgentRole: "reviewer"},
		},
		Edges: []core.EdgeDef{
			{From: "a", To: "b"},
			{From: "b", To: "c"},
		},
	}

	startNode := workflowDef.FindStartNode()
	if startNode == nil {
		t.Fatal("expected start node")
	}

	if startNode.ID != "a" {
		t.Errorf("expected 'a' as start, got %s", startNode.ID)
	}
}

func TestHydratedWorkflowExecutor_GetOutgoingEdges(t *testing.T) {
	workflowDef := &core.WorkflowDef{
		ID:      "edges-test",
		Name:    "Edges Test",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"a": {ID: "a", AgentRole: "planner"},
			"b": {ID: "b", AgentRole: "executor"},
			"c": {ID: "c", AgentRole: "reviewer"},
		},
		Edges: []core.EdgeDef{
			{From: "a", To: "b"},
			{From: "a", To: "c"},
		},
	}

	edges := workflowDef.GetOutgoingEdges("a")
	if len(edges) != 2 {
		t.Errorf("expected 2 outgoing edges from 'a', got %d", len(edges))
	}
}

func TestHydratedWorkflowExecutor_WithMockAgentFinder(t *testing.T) {
	ctx := context.Background()

	workflowDef := &core.WorkflowDef{
		ID:      "mock-agent-test",
		Name:    "Mock Agent Test",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"start": {
				ID:        "start",
				AgentRole: "planner",
				TaskType:  "agent",
			},
		},
		Edges: []core.EdgeDef{},
	}

	mockAgentFinder := &MockAgentFinder{
		agents: map[string][]string{
			"planner":  {"planner-001", "planner-002"},
			"executor": {"executor-001"},
		},
	}

	executor := NewHydratedWorkflowExecutor(workflowDef).
		WithAgentFinder(mockAgentFinder)

	result, err := executor.Execute(ctx, "test input")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if result.NodeResults["start"].AgentID != "planner-001" {
		t.Errorf("expected agentID 'planner-001', got %s", result.NodeResults["start"].AgentID)
	}
}

type MockConditionEvaluator struct {
	results map[string]bool
}

func (m *MockConditionEvaluator) EvaluateCondition(ctx context.Context, condition string, facts map[string]interface{}) (bool, error) {
	if result, ok := m.results[condition]; ok {
		return result, nil
	}
	return false, nil
}

type MockAgentFinder struct {
	agents map[string][]string
}

func (m *MockAgentFinder) FindAgentsByRole(ctx context.Context, role string) ([]string, error) {
	if agents, ok := m.agents[role]; ok {
		return agents, nil
	}
	return []string{}, nil
}

func TestHydratedWorkflowExecutor_WithSessionStore(t *testing.T) {
	ctx := context.Background()

	workflowDef := &core.WorkflowDef{
		ID:      "session-workflow",
		Name:    "Session Workflow",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"start": {
				ID:        "start",
				AgentRole: "planner",
				TaskType:  "agent",
			},
			"execute": {
				ID:        "execute",
				AgentRole: "executor",
				TaskType:  "agent",
			},
			"end": {
				ID:        "end",
				AgentRole: "reviewer",
				TaskType:  "agent",
			},
		},
		Edges: []core.EdgeDef{
			{From: "start", To: "execute"},
			{From: "execute", To: "end"},
		},
	}

	sessionStore := ports.NewInMemorySessionStore()
	sessionID := "test-session-001"

	executor := NewHydratedWorkflowExecutor(workflowDef).
		WithSessionStore(sessionStore, sessionID)

	result, instance, err := executor.ExecuteWithSession(ctx, "test input")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if result.Status != core.WorkflowStatusCompleted {
		t.Errorf("expected completed, got %s", result.Status)
	}

	if instance == nil {
		t.Fatal("expected workflow instance")
	}

	if instance.SessionID != sessionID {
		t.Errorf("expected sessionID %s, got %s", sessionID, instance.SessionID)
	}

	if instance.WorkflowID != workflowDef.ID {
		t.Errorf("expected workflowID %s, got %s", workflowDef.ID, instance.WorkflowID)
	}

	if instance.Status != core.WorkflowInstanceStatusCompleted {
		t.Errorf("expected instance status completed, got %s", instance.Status)
	}

	if len(instance.CompletedNodes) != 3 {
		t.Errorf("expected 3 completed nodes, got %d", len(instance.CompletedNodes))
	}

	t.Logf("Workflow completed with session: %s", result.Status)
	t.Logf("Instance status: %s", instance.Status)
	t.Logf("Completed nodes: %v", instance.CompletedNodes)
}

func TestHydratedWorkflowExecutor_ResumeSession(t *testing.T) {
	ctx := context.Background()

	workflowDef := &core.WorkflowDef{
		ID:      "resume-workflow",
		Name:    "Resume Workflow",
		Version: "v1.0",
		Nodes: map[string]core.NodeDef{
			"start": {
				ID:        "start",
				AgentRole: "planner",
				TaskType:  "agent",
			},
			"execute": {
				ID:        "execute",
				AgentRole: "executor",
				TaskType:  "agent",
			},
			"end": {
				ID:        "end",
				AgentRole: "reviewer",
				TaskType:  "agent",
			},
		},
		Edges: []core.EdgeDef{
			{From: "start", To: "execute"},
			{From: "execute", To: "end"},
		},
	}

	sessionStore := ports.NewInMemorySessionStore()
	sessionID := "test-session-002"

	existingInstance := core.NewWorkflowInstance(workflowDef.ID, sessionID)
	existingInstance.Status = core.WorkflowInstanceStatusPaused
	existingInstance.SetCurrentNode("execute")
	existingInstance.SetVariable("start_output", "completed output")
	existingInstance.MarkNodeCompleted("start")

	err := sessionStore.Create(ctx, existingInstance)
	if err != nil {
		t.Fatalf("failed to create existing instance: %v", err)
	}

	executor := NewHydratedWorkflowExecutor(workflowDef).
		WithSessionStore(sessionStore, sessionID)

	result, instance, err := executor.ExecuteWithSession(ctx, "resume input")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if result.Status != core.WorkflowStatusCompleted {
		t.Errorf("expected completed, got %s", result.Status)
	}

	if instance == nil {
		t.Fatal("expected workflow instance")
	}

	if len(instance.CompletedNodes) != 3 {
		t.Errorf("expected 3 completed nodes, got %d", len(instance.CompletedNodes))
	}

	hasStart := false
	for _, n := range instance.CompletedNodes {
		if n == "start" {
			hasStart = true
		}
	}
	if !hasStart {
		t.Error("expected 'start' to be in completed nodes (resumed from session)")
	}

	t.Logf("Workflow resumed and completed: %s", result.Status)
	t.Logf("All completed nodes: %v", instance.CompletedNodes)
}

var _ = fmt.Printf
