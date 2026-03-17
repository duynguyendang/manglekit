package multiagent

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
)

// AgentRole represents the role of an agent
type AgentRole string

// Agent represents an agent in the system
type Agent struct {
	ID           string
	Role         AgentRole
	Capabilities []string
	Config       map[string]string
	Status       AgentStatus
}

// AgentStatus represents the current status of an agent
type AgentStatus string

const (
	StatusAvailable AgentStatus = "available"
	StatusBusy      AgentStatus = "busy"
	StatusOffline   AgentStatus = "offline"
)

// Workflow represents a workflow defined in Datalog
type Workflow struct {
	ID      string
	Name    string
	Version string
	Nodes   []WorkflowNode
	Edges   []WorkflowEdge
}

// WorkflowNode represents a node in a workflow
type WorkflowNode struct {
	ID     string
	Type   string // "agent", "action", "condition", "parallel", "merge"
	Agent  string // agent role or action name
	Config map[string]string
}

// WorkflowEdge represents an edge between workflow nodes
type WorkflowEdge struct {
	From      string
	To        string
	Condition string // Datalog query for conditional edges
}

// AgentSystem manages agents and workflows using Datalog
type AgentSystem struct {
	engine   *engine.PolicyEngine
	registry map[string]*Agent
}

// Engine returns the underlying policy engine for external access
func (s *AgentSystem) Engine() *engine.PolicyEngine {
	return s.engine
}

// NewAgentSystem creates a new agent system
func NewAgentSystem(ctx context.Context) (*AgentSystem, error) {
	eng, err := engine.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create engine: %w", err)
	}

	return &AgentSystem{
		engine:   eng,
		registry: make(map[string]*Agent),
	}, nil
}

// Query executes a Datalog query against the agent system
func (s *AgentSystem) Query(ctx context.Context, facts []string, query string) ([]map[string]string, error) {
	return s.engine.Query(ctx, facts, query)
}

// QueryWithAudit executes a Datalog query and returns results with an audit trail.
// This provides transparency into which rules were matched and from which tier.
func (s *AgentSystem) QueryWithAudit(ctx context.Context, facts []string, query string) ([]map[string]string, *core.AuditTrail, error) {
	result, err := s.engine.QueryWithAudit(ctx, facts, query)
	if err != nil {
		return nil, nil, err
	}
	return result.Results, result.AuditTrail, nil
}

// LoadAgentDefinitions loads agent definitions from Datalog
func (s *AgentSystem) LoadAgentDefinitions(ctx context.Context) error {
	// Load agent registry Datalog via PolicyEngine
	// This would typically load from files or a database
	// For now, we add the Datalog rules directly
	agentRules := `
% Agent Roles
agent_role("planner").
agent_role("executor").
agent_role("reviewer").
agent_role("researcher").
agent_role("coordinator").
agent_role("supervisor").
agent_role("orchestrator").

% Role Capabilities
role_capability("planner", "task_decomposition").
role_capability("planner", "goal_analysis").
role_capability("planner", "plan_generation").
role_capability("planner", "llm_reasoning").

role_capability("executor", "action_execution").
role_capability("executor", "llm_generation").
role_capability("executor", "tool_invocation").

role_capability("reviewer", "validation").
role_capability("reviewer", "quality_check").
role_capability("reviewer", "llm_reasoning").

role_capability("researcher", "knowledge_retrieval").
role_capability("researcher", "vector_search").

% Role Inheritance
role_inherits("supervisor", "planner").
role_inherits("supervisor", "executor").
role_inherits("supervisor", "reviewer").

% Task Requirements
task_requires_capability("generate_document", "llm_generation").
task_requires_capability("validate_output", "validation").
task_requires_capability("create_plan", "plan_generation").

% Agent Registry
agent("planner-001", "planner").
agent("executor-001", "executor").
agent("executor-002", "executor").
agent("reviewer-001", "reviewer").
agent("researcher-001", "researcher").

% Agent Capabilities
agent_capability("planner-001", "llm_reasoning").
agent_capability("planner-001", "task_decomposition").
agent_capability("executor-001", "llm_generation").
agent_capability("executor-001", "tool_invocation").
agent_capability("executor-002", "llm_generation").
agent_capability("reviewer-001", "llm_reasoning").
agent_capability("reviewer-001", "validation").
agent_capability("researcher-001", "knowledge_retrieval").

% Agent Config
agent_config("planner-001", "model", "gpt-4o").
agent_config("executor-001", "model", "gpt-4o").
agent_config("reviewer-001", "model", "gpt-4o").

% Agent Status
agent_status("planner-001", "available").
agent_status("executor-001", "available").
agent_status("executor-002", "available").
agent_status("reviewer-001", "available").
agent_status("researcher-001", "available").

% Workflows
workflow("content-pipeline", "Content Generation", "v1.0").
workflow("simple-gen", "Simple Generation", "v1.0").

% Workflow Nodes
workflow_node("content-pipeline", "research", "agent", "researcher").
workflow_node("content-pipeline", "plan", "agent", "planner").
workflow_node("content-pipeline", "execute", "agent", "executor").
workflow_node("content-pipeline", "review", "agent", "reviewer").

workflow_node("simple-gen", "generate", "agent", "executor").
workflow_node("simple-gen", "review", "agent", "reviewer").

% Workflow Edges
workflow_edge("content-pipeline", "research", "plan").
workflow_edge("content-pipeline", "plan", "execute").
workflow_edge("content-pipeline", "execute", "review").

workflow_edge("simple-gen", "generate", "review").
`
	if err := s.engine.Runtime().AddPolicy(agentRules); err != nil {
		return fmt.Errorf("failed to load agent rules: %w", err)
	}

	return nil
}

// GetAgentsByRole retrieves all agents with the specified role
func (s *AgentSystem) GetAgentsByRole(ctx context.Context, role string) ([]*Agent, error) {
	query := fmt.Sprintf(`agent(AgentID, "%s"), agent_status(AgentID, "available").`, role)
	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var agents []*Agent
	seen := make(map[string]bool)
	for _, result := range results {
		id := result["AgentID"]
		if !seen[id] {
			seen[id] = true
			agent, err := s.GetAgent(ctx, id)
			if err != nil {
				continue
			}
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

// GetAgent retrieves an agent by ID
func (s *AgentSystem) GetAgent(ctx context.Context, agentID string) (*Agent, error) {
	// Check cache
	if agent, ok := s.registry[agentID]; ok {
		return agent, nil
	}

	// Query agent
	query := fmt.Sprintf(`agent("%s", Role).`, agentID)
	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	role := results[0]["Role"]

	// Query capabilities
	capsQuery := fmt.Sprintf(`agent_capability("%s", Capability).`, agentID)
	capsResults, err := s.engine.Query(ctx, []string{}, capsQuery)
	var capabilities []string
	for _, r := range capsResults {
		capabilities = append(capabilities, r["Capability"])
	}

	// Query config
	configQuery := fmt.Sprintf(`agent_config("%s", Key, Value).`, agentID)
	configResults, err := s.engine.Query(ctx, []string{}, configQuery)
	config := make(map[string]string)
	for _, r := range configResults {
		config[r["Key"]] = r["Value"]
	}

	// Query status
	statusQuery := fmt.Sprintf(`agent_status("%s", Status).`, agentID)
	statusResults, err := s.engine.Query(ctx, []string{}, statusQuery)
	status := StatusAvailable
	if err == nil && len(statusResults) > 0 {
		status = AgentStatus(statusResults[0]["Status"])
	}

	agent := &Agent{
		ID:           agentID,
		Role:         AgentRole(role),
		Capabilities: capabilities,
		Config:       config,
		Status:       status,
	}

	s.registry[agentID] = agent
	return agent, nil
}

// FindAgentsForTask finds agents capable of performing a task
func (s *AgentSystem) FindAgentsForTask(ctx context.Context, task string) ([]*Agent, error) {
	// Query capabilities for task
	query := fmt.Sprintf(`
		agent(AgentID, _),
		agent_capability(AgentID, Capability),
		agent_status(AgentID, "available"),
		task_requires_capability("%s", Capability).`, task)

	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var agents []*Agent
	seen := make(map[string]bool)
	for _, result := range results {
		id := result["AgentID"]
		if !seen[id] {
			seen[id] = true
			agent, err := s.GetAgent(ctx, id)
			if err != nil {
				continue
			}
			agents = append(agents, agent)
		}
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents found for task: %s", task)
	}

	return agents, nil
}

// GetRoleCapabilities returns capabilities for a given role
func (s *AgentSystem) GetRoleCapabilities(ctx context.Context, role string) ([]string, error) {
	// First check direct capabilities
	query := fmt.Sprintf(`role_capability("%s", Capability).`, role)
	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil {
		return nil, err
	}

	capabilities := make(map[string]bool)
	for _, r := range results {
		capabilities[r["Capability"]] = true
	}

	// Then check inherited roles
	inheritQuery := fmt.Sprintf(`role_inherits("%s", InheritedRole).`, role)
	inheritResults, err := s.engine.Query(ctx, []string{}, inheritQuery)
	if err == nil {
		for _, r := range inheritResults {
			inheritedRole := r["InheritedRole"]
			subQuery := fmt.Sprintf(`role_capability("%s", Capability).`, inheritedRole)
			subResults, err := s.engine.Query(ctx, []string{}, subQuery)
			if err == nil {
				for _, sr := range subResults {
					capabilities[sr["Capability"]] = true
				}
			}
		}
	}

	var caps []string
	for cap := range capabilities {
		caps = append(caps, cap)
	}

	return caps, nil
}

// GetWorkflow retrieves a workflow by name
func (s *AgentSystem) GetWorkflow(ctx context.Context, workflowName string) (*Workflow, error) {
	// Query workflow metadata
	query := fmt.Sprintf(`workflow("%s", Name, Version).`, workflowName)
	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil || len(results) == 0 {
		return nil, fmt.Errorf("workflow not found: %s", workflowName)
	}

	workflow := &Workflow{
		ID:      workflowName,
		Name:    results[0]["Name"],
		Version: results[0]["Version"],
	}

	// Query nodes
	nodeQuery := fmt.Sprintf(`workflow_node("%s", NodeID, Type, Agent).`, workflowName)
	nodeResults, err := s.engine.Query(ctx, []string{}, nodeQuery)
	if err == nil {
		for _, r := range nodeResults {
			node := WorkflowNode{
				ID:    r["NodeID"],
				Type:  r["Type"],
				Agent: r["Agent"],
			}

			// Query node config
			configQuery := fmt.Sprintf(`node_config("%s", "%s", Key, Value).`, workflowName, node.ID)
			configResults, err := s.engine.Query(ctx, []string{}, configQuery)
			if err == nil {
				node.Config = make(map[string]string)
				for _, cr := range configResults {
					node.Config[cr["Key"]] = cr["Value"]
				}
			}

			workflow.Nodes = append(workflow.Nodes, node)
		}
	}

	// Query edges
	edgeQuery := fmt.Sprintf(`workflow_edge("%s", From, To).`, workflowName)
	edgeResults, err := s.engine.Query(ctx, []string{}, edgeQuery)
	if err == nil {
		for _, r := range edgeResults {
			workflow.Edges = append(workflow.Edges, WorkflowEdge{
				From: r["From"],
				To:   r["To"],
			})
		}
	}

	// Query conditional edges
	condQuery := fmt.Sprintf(`conditional_edge("%s", From, To, Condition).`, workflowName)
	condResults, err := s.engine.Query(ctx, []string{}, condQuery)
	if err == nil {
		for _, r := range condResults {
			workflow.Edges = append(workflow.Edges, WorkflowEdge{
				From:      r["From"],
				To:        r["To"],
				Condition: r["Condition"],
			})
		}
	}

	return workflow, nil
}

// GetWorkflows retrieves all available workflows
func (s *AgentSystem) GetWorkflows(ctx context.Context) ([]*Workflow, error) {
	query := `workflow(ID, Name, Version).`
	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil {
		return nil, err
	}

	var workflows []*Workflow
	for _, r := range results {
		wf, err := s.GetWorkflow(ctx, r["ID"])
		if err != nil {
			continue
		}
		workflows = append(workflows, wf)
	}

	return workflows, nil
}

// GetAgentRoles retrieves all defined agent roles
func (s *AgentSystem) GetAgentRoles(ctx context.Context) ([]string, error) {
	query := `agent_role(Role).`
	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil {
		return nil, err
	}

	var roles []string
	for _, r := range results {
		roles = append(roles, r["Role"])
	}

	return roles, nil
}

// SetAgentStatus updates the status of an agent
func (s *AgentSystem) SetAgentStatus(ctx context.Context, agentID string, status AgentStatus) error {
	// Update in-memory registry
	if agent, ok := s.registry[agentID]; ok {
		agent.Status = status
	}

	// Note: In a real implementation, this would update the Datalog facts
	// For now, we only update in-memory
	return nil
}

// GetWorkflowEdges returns edges for a specific node
func (s *AgentSystem) GetWorkflowEdges(ctx context.Context, workflowName, nodeID string) ([]WorkflowEdge, error) {
	query := fmt.Sprintf(`workflow_edge("%s", "%s", To).`, workflowName, nodeID)
	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil {
		return nil, err
	}

	var edges []WorkflowEdge
	for _, r := range results {
		edges = append(edges, WorkflowEdge{
			From: nodeID,
			To:   r["To"],
		})
	}

	// Check conditional edges
	condQuery := fmt.Sprintf(`conditional_edge("%s", "%s", To, Condition).`, workflowName, nodeID)
	condResults, err := s.engine.Query(ctx, []string{}, condQuery)
	if err == nil {
		for _, r := range condResults {
			edges = append(edges, WorkflowEdge{
				From:      nodeID,
				To:        r["To"],
				Condition: r["Condition"],
			})
		}
	}

	return edges, nil
}

// EvaluateCondition evaluates a conditional edge
func (s *AgentSystem) EvaluateCondition(ctx context.Context, condition string, contextData map[string]string) (bool, error) {
	if condition == "" {
		return true, nil
	}

	// Build facts from context
	var facts []string
	for k, v := range contextData {
		facts = append(facts, fmt.Sprintf(`context("%s", "%s").`, k, escapeDatalog(v)))
	}

	query := condition
	results, err := s.engine.Query(ctx, facts, query)
	if err != nil {
		return false, err
	}

	return len(results) > 0, nil
}

// escapeDatalog escapes special characters for Datalog
func escapeDatalog(s string) string {
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, `\`, `\\`)
	return s
}

// Envelope wraps the input/output for agent execution
type Envelope = core.Envelope
