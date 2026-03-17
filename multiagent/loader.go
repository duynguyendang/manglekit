package multiagent

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

type DatalogWorkflowLoader struct {
	agentSystem *AgentSystem
}

func NewDatalogWorkflowLoader(agentSystem *AgentSystem) *DatalogWorkflowLoader {
	return &DatalogWorkflowLoader{
		agentSystem: agentSystem,
	}
}

func (l *DatalogWorkflowLoader) LoadWorkflow(ctx context.Context, workflowID string) (*core.WorkflowDef, error) {
	wf, err := l.agentSystem.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow %s: %w", workflowID, err)
	}

	return l.convertToWorkflowDef(wf), nil
}

func (l *DatalogWorkflowLoader) convertToWorkflowDef(wf *Workflow) *core.WorkflowDef {
	nodes := make(map[string]core.NodeDef)
	for _, node := range wf.Nodes {
		config := make(map[string]interface{})
		for k, v := range node.Config {
			config[k] = v
		}
		nodes[node.ID] = core.NodeDef{
			ID:        node.ID,
			AgentRole: node.Agent,
			TaskType:  node.Type,
			Config:    config,
		}
	}

	var edges []core.EdgeDef
	for _, edge := range wf.Edges {
		edges = append(edges, core.EdgeDef{
			From:      edge.From,
			To:        edge.To,
			Condition: edge.Condition,
		})
	}

	startNode := findStartNode(wf)
	rootNodeID := ""
	if startNode != nil {
		rootNodeID = startNode.ID
	}

	return &core.WorkflowDef{
		ID:         wf.ID,
		Name:       wf.Name,
		Version:    wf.Version,
		RootNodeID: rootNodeID,
		Nodes:      nodes,
		Edges:      edges,
	}
}

func findStartNode(wf *Workflow) *WorkflowNode {
	hasIncoming := make(map[string]bool)
	for _, edge := range wf.Edges {
		hasIncoming[edge.To] = true
	}

	for _, node := range wf.Nodes {
		if !hasIncoming[node.ID] {
			return &node
		}
	}

	if len(wf.Nodes) > 0 {
		return &wf.Nodes[0]
	}
	return nil
}

var _ ports.WorkflowLoader = (*DatalogWorkflowLoader)(nil)

type ConditionEvaluatorAdapter struct {
	agentSystem *AgentSystem
}

func NewConditionEvaluatorAdapter(agentSystem *AgentSystem) *ConditionEvaluatorAdapter {
	return &ConditionEvaluatorAdapter{
		agentSystem: agentSystem,
	}
}

func (e *ConditionEvaluatorAdapter) EvaluateCondition(ctx context.Context, condition string, facts map[string]interface{}) (bool, error) {
	if condition == "" {
		return true, nil
	}

	var factStrings []string
	for k, v := range facts {
		factStrings = append(factStrings, fmt.Sprintf(`context("%s", "%v").`, k, v))
	}

	results, err := e.agentSystem.Query(ctx, factStrings, condition)
	if err != nil {
		return false, err
	}

	return len(results) > 0, nil
}

var _ ports.ConditionEvaluator = (*ConditionEvaluatorAdapter)(nil)

type AgentFinderAdapter struct {
	agentSystem *AgentSystem
}

func NewAgentFinderAdapter(agentSystem *AgentSystem) *AgentFinderAdapter {
	return &AgentFinderAdapter{
		agentSystem: agentSystem,
	}
}

func (f *AgentFinderAdapter) FindAgentsByRole(ctx context.Context, role string) ([]string, error) {
	agents, err := f.agentSystem.GetAgentsByRole(ctx, role)
	if err != nil {
		return nil, err
	}

	var agentIDs []string
	for _, agent := range agents {
		agentIDs = append(agentIDs, agent.ID)
	}

	return agentIDs, nil
}

var _ ports.AgentFinder = (*AgentFinderAdapter)(nil)
