package multiagent

import (
	"context"
	"fmt"
	"testing"
)

func TestAgentSystem(t *testing.T) {
	ctx := context.Background()

	// Create agent system
	system, err := NewAgentSystem(ctx)
	if err != nil {
		t.Fatalf("failed to create agent system: %v", err)
	}

	// Load agent definitions from Datalog
	if err := system.LoadAgentDefinitions(ctx); err != nil {
		t.Fatalf("failed to load agent definitions: %v", err)
	}

	// Test: Get all agent roles
	roles, err := system.GetAgentRoles(ctx)
	if err != nil {
		t.Fatalf("failed to get agent roles: %v", err)
	}
	fmt.Printf("Agent Roles: %v\n", roles)

	// Test: Get agents by role
	executors, err := system.GetAgentsByRole(ctx, "executor")
	if err != nil {
		t.Fatalf("failed to get executors: %v", err)
	}
	fmt.Printf("Executors: %v\n", executors)

	// Test: Find agents for task
	agents, err := system.FindAgentsForTask(ctx, "generate_document")
	if err != nil {
		t.Fatalf("failed to find agents for task: %v", err)
	}
	fmt.Printf("Agents for 'generate_document': %v\n", agents)

	// Test: Get workflow
	workflow, err := system.GetWorkflow(ctx, "content-pipeline")
	if err != nil {
		t.Fatalf("failed to get workflow: %v", err)
	}
	fmt.Printf("Workflow: %s - %s (v%s)\n", workflow.ID, workflow.Name, workflow.Version)
	fmt.Printf("Nodes: %v\n", workflow.Nodes)
	fmt.Printf("Edges: %v\n", workflow.Edges)

	// Test: Get role capabilities (including inherited)
	caps, err := system.GetRoleCapabilities(ctx, "supervisor")
	if err != nil {
		t.Fatalf("failed to get supervisor capabilities: %v", err)
	}
	fmt.Printf("Supervisor capabilities (with inheritance): %v\n", caps)
}

func ExampleAgentSystem() {
	ctx := context.Background()

	system, _ := NewAgentSystem(ctx)
	system.LoadAgentDefinitions(ctx)

	// Query: Which agents can generate a document?
	agents, _ := system.FindAgentsForTask(ctx, "generate_document")
	fmt.Println("=== Agents for generate_document ===")
	for _, a := range agents {
		fmt.Printf("- %s (%s): %v\n", a.ID, a.Role, a.Capabilities)
	}

	// Query: What can a supervisor do?
	caps, _ := system.GetRoleCapabilities(ctx, "supervisor")
	fmt.Println("\n=== Supervisor capabilities ===")
	for _, c := range caps {
		fmt.Printf("- %s\n", c)
	}

	// Query: Get workflow structure
	wf, _ := system.GetWorkflow(ctx, "content-pipeline")
	fmt.Printf("\n=== Workflow: %s ===\n", wf.Name)
	for _, n := range wf.Nodes {
		fmt.Printf("Node: %s (type: %s, agent: %s)\n", n.ID, n.Type, n.Agent)
	}
}
