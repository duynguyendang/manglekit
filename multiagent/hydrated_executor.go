package multiagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

type HydratedWorkflowExecutor struct {
	workflowDef   *core.WorkflowDef
	conditionEval ports.ConditionEvaluator
	agentFinder   ports.AgentFinder
	nodeExecutor  NodeExecutor
	sessionStore  ports.SessionStateStore
	sessionID     string
	maxRetries    int
	timeout       time.Duration
}

func NewHydratedWorkflowExecutor(workflowDef *core.WorkflowDef) *HydratedWorkflowExecutor {
	return &HydratedWorkflowExecutor{
		workflowDef:  workflowDef,
		nodeExecutor: &DefaultNodeExecutor{},
		maxRetries:   3,
		timeout:      5 * time.Minute,
	}
}

func (e *HydratedWorkflowExecutor) WithConditionEvaluator(eval ports.ConditionEvaluator) *HydratedWorkflowExecutor {
	e.conditionEval = eval
	return e
}

func (e *HydratedWorkflowExecutor) WithAgentFinder(finder ports.AgentFinder) *HydratedWorkflowExecutor {
	e.agentFinder = finder
	return e
}

func (e *HydratedWorkflowExecutor) WithNodeExecutor(exec NodeExecutor) *HydratedWorkflowExecutor {
	e.nodeExecutor = exec
	return e
}

func (e *HydratedWorkflowExecutor) WithMaxRetries(retries int) *HydratedWorkflowExecutor {
	e.maxRetries = retries
	return e
}

func (e *HydratedWorkflowExecutor) WithTimeout(timeout time.Duration) *HydratedWorkflowExecutor {
	e.timeout = timeout
	return e
}

func (e *HydratedWorkflowExecutor) WithSessionStore(store ports.SessionStateStore, sessionID string) *HydratedWorkflowExecutor {
	e.sessionStore = store
	e.sessionID = sessionID
	return e
}

func (e *HydratedWorkflowExecutor) Execute(ctx context.Context, initialInput interface{}) (*core.WorkflowResult, error) {
	if e.workflowDef == nil {
		return nil, fmt.Errorf("workflow definition is nil")
	}

	result := &core.WorkflowResult{
		WorkflowID:  e.workflowDef.ID,
		Status:      core.WorkflowStatusRunning,
		NodeResults: make(map[string]*core.NodeResult),
		StartTime:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	nodeResults := make(map[string]interface{})
	currentInput := initialInput
	currentNode := e.workflowDef.FindStartNode()

	if currentNode == nil {
		result.Status = core.WorkflowStatusFailed
		result.Error = fmt.Errorf("no start node found")
		result.EndTime = time.Now()
		return result, result.Error
	}

	visited := make(map[string]int)
	var lastError error

	for currentNode != nil {
		if ctx.Err() != nil {
			result.Status = core.WorkflowStatusCancelled
			result.Error = ctx.Err()
			break
		}

		visited[currentNode.ID]++
		if visited[currentNode.ID] > e.maxRetries {
			result.Status = core.WorkflowStatusFailed
			result.Error = fmt.Errorf("max retries exceeded for node %s", currentNode.ID)
			break
		}

		nodeResult := e.executeNode(ctx, currentNode, currentInput, nodeResults)
		result.NodeResults[currentNode.ID] = nodeResult

		if nodeResult.Error != nil {
			lastError = nodeResult.Error

			errorEdge := e.findErrorEdge(currentNode.ID)
			if errorEdge == nil {
				result.Status = core.WorkflowStatusFailed
				break
			}
			currentNode = e.workflowDef.GetNode(errorEdge.To)
			continue
		}

		nodeResults[currentNode.ID] = nodeResult.Output
		currentInput = nodeResult.Output

		nextNode, err := e.findNextNode(ctx, currentNode.ID, nodeResult.Output, nodeResults)
		if err != nil {
			lastError = err
			result.Status = core.WorkflowStatusFailed
			break
		}

		if nextNode == nil {
			break
		}

		currentNode = nextNode
	}

	result.EndTime = time.Now()

	if result.Status == core.WorkflowStatusRunning {
		if lastError != nil {
			result.Status = core.WorkflowStatusFailed
			result.Error = lastError
		} else {
			result.Status = core.WorkflowStatusCompleted
			result.Output = currentInput
		}
	}

	return result, nil
}

func (e *HydratedWorkflowExecutor) ExecuteWithSession(ctx context.Context, initialInput interface{}) (*core.WorkflowResult, *core.WorkflowInstance, error) {
	if e.workflowDef == nil {
		return nil, nil, fmt.Errorf("workflow definition is nil")
	}

	if e.sessionStore == nil || e.sessionID == "" {
		result, err := e.Execute(ctx, initialInput)
		return result, nil, err
	}

	result := &core.WorkflowResult{
		WorkflowID:  e.workflowDef.ID,
		Status:      core.WorkflowStatusRunning,
		NodeResults: make(map[string]*core.NodeResult),
		StartTime:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	sessionKey := e.sessionID + ":" + e.workflowDef.ID
	instance, err := e.sessionStore.Get(ctx, sessionKey)
	if err != nil {
		instance = core.NewWorkflowInstance(e.workflowDef.ID, e.sessionID)
		instance.Status = core.WorkflowInstanceStatusRunning
		if err := e.sessionStore.Create(ctx, instance); err != nil {
			return nil, nil, fmt.Errorf("failed to create session: %w", err)
		}
	}

	nodeResults := make(map[string]interface{})
	for k, v := range instance.Variables {
		nodeResults[k] = v
	}

	currentInput := initialInput
	if instance.CurrentNodeID != "" {
		currentNode := e.workflowDef.GetNode(instance.CurrentNodeID)
		if currentNode != nil {
			currentInput = instance.GetVariable("current_input")
		}
	} else {
		startNode := e.workflowDef.FindStartNode()
		if startNode == nil {
			result.Status = core.WorkflowStatusFailed
			result.Error = fmt.Errorf("no start node found")
			result.EndTime = time.Now()
			return result, instance, result.Error
		}
		instance.SetCurrentNode(startNode.ID)
	}

	visited := make(map[string]int)
	for _, nodeID := range instance.CompletedNodes {
		visited[nodeID] = 1
	}

	var lastError error

	for {
		if ctx.Err() != nil {
			result.Status = core.WorkflowStatusCancelled
			result.Error = ctx.Err()
			instance.Status = core.WorkflowInstanceStatusPaused
			e.sessionStore.Update(ctx, instance)
			break
		}

		currentNodeID := instance.CurrentNodeID
		if currentNodeID == "" {
			break
		}

		currentNode := e.workflowDef.GetNode(currentNodeID)
		if currentNode == nil {
			break
		}

		visited[currentNodeID]++
		if visited[currentNodeID] > e.maxRetries {
			result.Status = core.WorkflowStatusFailed
			result.Error = fmt.Errorf("max retries exceeded for node %s", currentNodeID)
			instance.Status = core.WorkflowInstanceStatusFailed
			break
		}

		instance.SetVariable("current_input", currentInput)
		e.sessionStore.Update(ctx, instance)

		nodeResult := e.executeNode(ctx, currentNode, currentInput, nodeResults)
		result.NodeResults[currentNode.ID] = nodeResult

		if nodeResult.Error != nil {
			lastError = nodeResult.Error
			instance.MarkNodeFailed(currentNode.ID)

			errorEdge := e.findErrorEdge(currentNode.ID)
			if errorEdge == nil {
				result.Status = core.WorkflowStatusFailed
				instance.Status = core.WorkflowInstanceStatusFailed
				break
			}
			instance.SetCurrentNode(errorEdge.To)
			e.sessionStore.Update(ctx, instance)
			continue
		}

		nodeResults[currentNode.ID] = nodeResult.Output
		instance.SetVariable(currentNode.ID+"_output", nodeResult.Output)
		instance.MarkNodeCompleted(currentNode.ID)
		currentInput = nodeResult.Output

		nextNode, err := e.findNextNode(ctx, currentNode.ID, nodeResult.Output, nodeResults)
		if err != nil {
			lastError = err
			result.Status = core.WorkflowStatusFailed
			instance.Status = core.WorkflowInstanceStatusFailed
			break
		}

		if nextNode == nil {
			instance.Status = core.WorkflowInstanceStatusCompleted
			break
		}

		instance.SetCurrentNode(nextNode.ID)
		e.sessionStore.Update(ctx, instance)
		currentNode = nextNode
	}

	result.EndTime = time.Now()

	if result.Status == core.WorkflowStatusRunning {
		if lastError != nil {
			result.Status = core.WorkflowStatusFailed
			result.Error = lastError
			instance.Status = core.WorkflowInstanceStatusFailed
		} else {
			result.Status = core.WorkflowStatusCompleted
			result.Output = currentInput
			instance.Status = core.WorkflowInstanceStatusCompleted
		}
	}

	e.sessionStore.Update(ctx, instance)
	return result, instance, nil
}

func (e *HydratedWorkflowExecutor) executeNode(ctx context.Context, node *core.NodeDef, input interface{}, previousResults map[string]interface{}) *core.NodeResult {
	result := &core.NodeResult{
		NodeID:    node.ID,
		Input:     input,
		StartTime: time.Now(),
	}

	var agentID string
	if e.agentFinder != nil {
		agents, err := e.agentFinder.FindAgentsByRole(ctx, node.AgentRole)
		if err == nil && len(agents) > 0 {
			agentID = agents[0]
		}
	}
	result.AgentID = agentID

	var execErr error
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		result.RetryCount = attempt

		output, err := e.nodeExecutor.Execute(ctx, toWorkflowNode(node), input, &Agent{ID: agentID, Role: AgentRole(node.AgentRole)})
		if err == nil {
			result.Output = output
			result.Duration = time.Since(result.StartTime)
			return result
		}

		execErr = err
		if attempt < e.maxRetries {
			timer := time.NewTimer(time.Duration(attempt+1) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				result.Error = ctx.Err()
				return result
			case <-timer.C:
				timer.Stop()
			}
		}
	}

	result.Error = execErr
	result.Duration = time.Since(result.StartTime)
	return result
}

func (e *HydratedWorkflowExecutor) findNextNode(ctx context.Context, currentNodeID string, nodeOutput interface{}, previousResults map[string]interface{}) (*core.NodeDef, error) {
	edges := e.workflowDef.GetOutgoingEdges(currentNodeID)

	if len(edges) == 0 {
		return nil, nil
	}

	for _, edge := range edges {
		if edge.Condition == "" {
			return e.workflowDef.GetNode(edge.To), nil
		}

		if e.conditionEval != nil {
			facts := make(map[string]interface{})
			facts["current_output"] = nodeOutput
			for k, v := range previousResults {
				facts[k] = v
			}

			approved, err := e.conditionEval.EvaluateCondition(ctx, edge.Condition, facts)
			if err != nil {
				return nil, err
			}

			if approved {
				return e.workflowDef.GetNode(edge.To), nil
			}
		}
	}

	return nil, nil
}

func (e *HydratedWorkflowExecutor) findErrorEdge(nodeID string) *core.EdgeDef {
	for _, edge := range e.workflowDef.Edges {
		if edge.From == nodeID && strings.HasPrefix(edge.Condition, "error") {
			return &edge
		}
	}
	return nil
}

func toWorkflowNode(node *core.NodeDef) *WorkflowNode {
	config := make(map[string]string)
	for k, v := range node.Config {
		config[k] = fmt.Sprintf("%v", v)
	}
	return &WorkflowNode{
		ID:     node.ID,
		Type:   node.TaskType,
		Agent:  node.AgentRole,
		Config: config,
	}
}
