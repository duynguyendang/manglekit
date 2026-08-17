package multiagent

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
)

//go:embed assets/agent_registry.dlog
var defaultAgentRegistry string

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
	mu       sync.RWMutex
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

// LoadAgentDefinitions loads agent definitions from Datalog.
// The default registry is embedded in the binary from assets/agent_registry.dlog.
// Callers may also load custom rules by calling Engine().Runtime().AddPolicy()
// directly after this call.
func (s *AgentSystem) LoadAgentDefinitions(ctx context.Context) error {
	if err := s.engine.Runtime().AddPolicy(ctx, defaultAgentRegistry); err != nil {
		return fmt.Errorf("failed to load agent rules: %w", err)
	}

	return nil
}

// GetAgentsByRole retrieves all agents with the specified role
func (s *AgentSystem) GetAgentsByRole(ctx context.Context, role string) ([]*Agent, error) {
	query := fmt.Sprintf(`agent(AgentID, "%s"), agent_status(AgentID, "available").`, escapeDatalog(role))
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
	s.mu.RLock()
	agent, ok := s.registry[agentID]
	s.mu.RUnlock()

	if ok {
		return agent, nil
	}

	// Fetch role, capabilities, config, and status in parallel instead of
	// issuing 4 sequential Datalog queries. An agent that doesn't exist will
	// surface as an empty role on the first query.
	safeID := escapeDatalog(agentID)

	var (
		wg           sync.WaitGroup
		queryErr     error
		errOnce      sync.Once
		setErr       = func(e error) { errOnce.Do(func() { queryErr = e }) }
		role         string
		capabilities []string
		config       = make(map[string]string)
		status       = StatusAvailable
	)

	wg.Add(4)
	go func() {
		defer wg.Done()
		results, err := s.engine.Query(ctx, []string{}, fmt.Sprintf(`agent("%s", Role).`, safeID))
		if err != nil {
			setErr(fmt.Errorf("agent query failed: %w", err))
			return
		}
		if len(results) > 0 {
			role = results[0]["Role"]
		}
	}()
	go func() {
		defer wg.Done()
		results, err := s.engine.Query(ctx, []string{}, fmt.Sprintf(`agent_capability("%s", Capability).`, safeID))
		if err != nil {
			setErr(fmt.Errorf("capability query failed: %w", err))
			return
		}
		for _, r := range results {
			capabilities = append(capabilities, r["Capability"])
		}
	}()
	go func() {
		defer wg.Done()
		results, err := s.engine.Query(ctx, []string{}, fmt.Sprintf(`agent_config("%s", Key, Value).`, safeID))
		if err != nil {
			setErr(fmt.Errorf("config query failed: %w", err))
			return
		}
		for _, r := range results {
			config[r["Key"]] = r["Value"]
		}
	}()
	go func() {
		defer wg.Done()
		results, err := s.engine.Query(ctx, []string{}, fmt.Sprintf(`agent_status("%s", Status).`, safeID))
		if err != nil {
			// Status is optional; keep default and don't fail the call.
			return
		}
		if len(results) > 0 {
			status = AgentStatus(results[0]["Status"])
		}
	}()
	wg.Wait()

	if queryErr != nil {
		return nil, queryErr
	}
	if role == "" {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	agent = &Agent{
		ID:           agentID,
		Role:         AgentRole(role),
		Capabilities: capabilities,
		Config:       config,
		Status:       status,
	}

	s.mu.Lock()
	s.registry[agentID] = agent
	s.mu.Unlock()

	return agent, nil
}

// FindAgentsForTask finds agents capable of performing a task
func (s *AgentSystem) FindAgentsForTask(ctx context.Context, task string) ([]*Agent, error) {
	// Query capabilities for task
	query := fmt.Sprintf(`
		agent(AgentID, _),
		agent_capability(AgentID, Capability),
		agent_status(AgentID, "available"),
		task_requires_capability("%s", Capability).`, escapeDatalog(task))

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
	query := fmt.Sprintf(`role_capability("%s", Capability).`, escapeDatalog(role))
	results, err := s.engine.Query(ctx, []string{}, query)
	if err != nil {
		return nil, err
	}

	capabilities := make(map[string]bool)
	for _, r := range results {
		capabilities[r["Capability"]] = true
	}

	// Then check inherited roles
	inheritQuery := fmt.Sprintf(`role_inherits("%s", InheritedRole).`, escapeDatalog(role))
	inheritResults, err := s.engine.Query(ctx, []string{}, inheritQuery)
	if err == nil {
		for _, r := range inheritResults {
			inheritedRole := r["InheritedRole"]
			subQuery := fmt.Sprintf(`role_capability("%s", Capability).`, escapeDatalog(inheritedRole))
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
	query := fmt.Sprintf(`workflow("%s", Name, Version).`, escapeDatalog(workflowName))
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
	nodeQuery := fmt.Sprintf(`workflow_node("%s", NodeID, Type, Agent).`, escapeDatalog(workflowName))
	nodeResults, err := s.engine.Query(ctx, []string{}, nodeQuery)
	if err == nil {
		for _, r := range nodeResults {
			node := WorkflowNode{
				ID:    r["NodeID"],
				Type:  r["Type"],
				Agent: r["Agent"],
			}

			// Query node config
			configQuery := fmt.Sprintf(`node_config("%s", "%s", Key, Value).`, escapeDatalog(workflowName), escapeDatalog(node.ID))
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
	edgeQuery := fmt.Sprintf(`workflow_edge("%s", From, To).`, escapeDatalog(workflowName))
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
	condQuery := fmt.Sprintf(`conditional_edge("%s", From, To, Condition).`, escapeDatalog(workflowName))
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if agent, ok := s.registry[agentID]; ok {
		agent.Status = status
	}

	return nil
}

// GetWorkflowEdges returns edges for a specific node
func (s *AgentSystem) GetWorkflowEdges(ctx context.Context, workflowName, nodeID string) ([]WorkflowEdge, error) {
	query := fmt.Sprintf(`workflow_edge("%s", "%s", To).`, escapeDatalog(workflowName), escapeDatalog(nodeID))
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
	condQuery := fmt.Sprintf(`conditional_edge("%s", "%s", To, Condition).`, escapeDatalog(workflowName), escapeDatalog(nodeID))
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
		facts = append(facts, fmt.Sprintf(`context("%s", "%s").`, escapeDatalog(k), escapeDatalog(v)))
	}

	query := condition
	results, err := s.engine.Query(ctx, facts, query)
	if err != nil {
		return false, err
	}

	return len(results) > 0, nil
}

func escapeDatalog(s string) string {
	return engine.EscapeString(s)
}

// Envelope wraps the input/output for agent execution
type Envelope = core.Envelope
