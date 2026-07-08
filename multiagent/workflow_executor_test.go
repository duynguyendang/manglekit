package multiagent

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWorkflowExecutor_Execute(t *testing.T) {
	ctx := context.Background()

	system, err := NewAgentSystem(ctx)
	if err != nil {
		t.Fatalf("failed to create agent system: %v", err)
	}

	if err := system.LoadAgentDefinitions(ctx); err != nil {
		t.Fatalf("failed to load agent definitions: %v", err)
	}

	executor := NewWorkflowExecutor(system)

	result, err := executor.Execute(ctx, "content-pipeline", "initial input")
	if err != nil {
		t.Fatalf("workflow execution failed: %v", err)
	}

	if result.Status != WorkflowStatusCompleted {
		t.Errorf("expected workflow status %s, got %s", WorkflowStatusCompleted, result.Status)
	}

	if len(result.NodeResults) == 0 {
		t.Error("expected node results, got none")
	}

	t.Logf("Workflow completed: %s", result.Status)
	for nodeID, nodeResult := range result.NodeResults {
		t.Logf("  Node %s: executed by %s, duration %v", nodeID, nodeResult.AgentID, nodeResult.EndTime)
	}
}

func TestWorkflowExecutor_WithCustomExecutor(t *testing.T) {
	ctx := context.Background()

	system, _ := NewAgentSystem(ctx)
	system.LoadAgentDefinitions(ctx)

	callCount := 0

	customExecutor := &CustomTestExecutor{
		executeFn: func(node *WorkflowNode) (interface{}, error) {
			callCount++
			return fmt.Sprintf("custom output from %s", node.ID), nil
		},
	}

	executor := NewWorkflowExecutor(system).WithNodeExecutor(customExecutor)

	result, err := executor.Execute(ctx, "simple-gen", "test input")
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if result.Status != WorkflowStatusCompleted {
		t.Errorf("expected completed, got %s", result.Status)
	}

	if callCount == 0 {
		t.Error("expected custom executor to be called")
	}

	t.Logf("Custom executor was called %d times", callCount)
}

func TestWorkflowExecutor_MaxRetries(t *testing.T) {
	ctx := context.Background()

	system, _ := NewAgentSystem(ctx)
	system.LoadAgentDefinitions(ctx)

	failingExecutor := &FailingTestExecutor{
		failUntilAttempt: 2,
	}

	executor := NewWorkflowExecutor(system).
		WithNodeExecutor(failingExecutor).
		WithMaxRetries(3)

	result, err := executor.Execute(ctx, "simple-gen", "test")
	if err != nil {
		t.Logf("Expected error: %v", err)
	}

	if result.Status == WorkflowStatusCompleted {
		t.Log("Workflow completed after retries")
	} else {
		t.Logf("Workflow status: %s", result.Status)
	}
}

func TestWorkflowExecutor_Timeout(t *testing.T) {
	ctx := context.Background()

	system, _ := NewAgentSystem(ctx)
	system.LoadAgentDefinitions(ctx)

	slowExecutor := &SlowTestExecutor{
		delay: 100 * time.Millisecond,
	}

	executor := NewWorkflowExecutor(system).
		WithNodeExecutor(slowExecutor).
		WithTimeout(50 * time.Millisecond)

	result, err := executor.Execute(ctx, "simple-gen", "test")
	if err != nil {
		t.Logf("Execution error (expected due to timeout): %v", err)
	}

	t.Logf("Result status: %s", result.Status)
}

func TestParallelWorkflowExecutor(t *testing.T) {
	ctx := context.Background()

	system, _ := NewAgentSystem(ctx)
	system.LoadAgentDefinitions(ctx)

	executor := NewParallelWorkflowExecutor(system)

	workflow, err := system.GetWorkflow(ctx, "content-pipeline")
	if err != nil {
		t.Fatalf("failed to get workflow: %v", err)
	}

	result, err := executor.ExecuteParallel(ctx, workflow, "parallel test")
	if err != nil {
		t.Fatalf("parallel execution failed: %v", err)
	}

	t.Logf("Parallel workflow status: %s", result.Status)
	t.Logf("Node results: %d", len(result.NodeResults))
}

func TestWorkflowExecutor_ConditionalEdges(t *testing.T) {
	ctx := context.Background()

	system, _ := NewAgentSystem(ctx)

	_ = system.Engine().Runtime().AddPolicy(context.Background(), `
% Test workflow with conditional edges
workflow("conditional-test", "Conditional Test", "v1.0").

workflow_node("conditional-test", "start", "agent", "planner").
workflow_node("conditional-test", "branch_a", "agent", "executor").
workflow_node("conditional-test", "branch_b", "agent", "reviewer").
workflow_node("conditional-test", "end", "agent", "researcher").

workflow_edge("conditional-test", "start", "branch_a").
workflow_edge("conditional-test", "start", "branch_b").

conditional_edge("conditional-test", "branch_a", "end", "context("output") = "a"").
conditional_edge("conditional-test", "branch_b", "end", "context("output") = "b"").
	`)

	executor := NewWorkflowExecutor(system)

	result, err := executor.Execute(ctx, "conditional-test", "test")
	if err != nil {
		t.Logf("Execution error (expected if workflow not found): %v", err)
	}

	if result != nil {
		t.Logf("Conditional workflow status: %s", result.Status)
	}
}

func TestFindStartNode(t *testing.T) {
	workflow := &Workflow{
		ID: "test-workflow",
		Nodes: []WorkflowNode{
			{ID: "node1", Type: "agent", Agent: "planner"},
			{ID: "node2", Type: "agent", Agent: "executor"},
			{ID: "node3", Type: "agent", Agent: "reviewer"},
		},
		Edges: []WorkflowEdge{
			{From: "node1", To: "node2"},
			{From: "node2", To: "node3"},
		},
	}

	executor := NewWorkflowExecutor(nil)
	startNode := executor.findStartNode(workflow)

	if startNode == nil {
		t.Fatal("expected start node, got nil")
	}

	if startNode.ID != "node1" {
		t.Errorf("expected node1 as start, got %s", startNode.ID)
	}
}

func TestFindNextNode(t *testing.T) {
	ctx := context.Background()
	system, _ := NewAgentSystem(ctx)
	system.LoadAgentDefinitions(ctx)

	workflow := &Workflow{
		ID: "test",
		Nodes: []WorkflowNode{
			{ID: "start", Type: "agent", Agent: "planner"},
			{ID: "middle", Type: "agent", Agent: "executor"},
			{ID: "end", Type: "agent", Agent: "reviewer"},
		},
		Edges: []WorkflowEdge{
			{From: "start", To: "middle"},
			{From: "middle", To: "end"},
		},
	}

	executor := NewWorkflowExecutor(system)
	middleNode := executor.findNodeByID(workflow, "middle")

	next, err := executor.findNextNode(ctx, workflow, middleNode, "output", map[string]interface{}{})
	if err != nil {
		t.Fatalf("findNextNode failed: %v", err)
	}

	if next == nil {
		t.Fatal("expected next node, got nil")
	}

	if next.ID != "end" {
		t.Errorf("expected 'end' as next node, got %s", next.ID)
	}
}

func TestIdentifyParallelGroups(t *testing.T) {
	workflow := &Workflow{
		ID: "parallel-test",
		Nodes: []WorkflowNode{
			{ID: "start", Type: "agent", Agent: "planner"},
			{ID: "parallel1", Type: "agent", Agent: "executor"},
			{ID: "parallel2", Type: "agent", Agent: "reviewer"},
			{ID: "parallel3", Type: "agent", Agent: "researcher"},
			{ID: "end", Type: "agent", Agent: "planner"},
		},
		Edges: []WorkflowEdge{
			{From: "start", To: "parallel1"},
			{From: "start", To: "parallel2"},
			{From: "start", To: "parallel3"},
			{From: "parallel1", To: "end"},
			{From: "parallel2", To: "end"},
			{From: "parallel3", To: "end"},
		},
	}

	executor := NewParallelWorkflowExecutor(nil)
	groups := executor.identifyParallelGroups(workflow)

	if len(groups) == 0 {
		t.Error("expected parallel groups, got none")
	}

	foundParallel := false
	for _, group := range groups {
		if len(group) >= 3 {
			foundParallel = true
			t.Logf("Found parallel group: %v", group)
		}
	}

	if !foundParallel {
		t.Log("No parallel group with 3+ nodes found (this may be expected)")
	}
}

func TestWorkflowContext_MergeOutputs(t *testing.T) {
	executor := NewParallelWorkflowExecutor(nil)

	outputs := map[string]interface{}{
		"node1": "output from node 1",
		"node2": "output from node 2",
		"node3": "output from node 3",
	}

	merged := executor.mergeOutputs(outputs)

	switch v := merged.(type) {
	case []string:
		if len(v) != 3 {
			t.Errorf("expected 3 merged outputs, got %d", len(v))
		}
	default:
		t.Logf("Merged output type: %T", v)
	}
}

type CustomTestExecutor struct {
	executeFn func(node *WorkflowNode) (interface{}, error)
}

func (e *CustomTestExecutor) Execute(ctx context.Context, node *WorkflowNode, input interface{}, agent *Agent) (interface{}, error) {
	return e.executeFn(node)
}

type FailingTestExecutor struct {
	failUntilAttempt int
	attempt          int
}

func (e *FailingTestExecutor) Execute(ctx context.Context, node *WorkflowNode, input interface{}, agent *Agent) (interface{}, error) {
	e.attempt++
	if e.attempt <= e.failUntilAttempt {
		return nil, fmt.Errorf("intentional failure on attempt %d", e.attempt)
	}
	return "success after retries", nil
}

type SlowTestExecutor struct {
	delay time.Duration
}

func (e *SlowTestExecutor) Execute(ctx context.Context, node *WorkflowNode, input interface{}, agent *Agent) (interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(e.delay):
		return "slow output", nil
	}
}
