package multiagent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

type NodeExecutor interface {
	Execute(ctx context.Context, node *WorkflowNode, input interface{}, agent *Agent) (interface{}, error)
}

type DefaultNodeExecutor struct{}

func (e *DefaultNodeExecutor) Execute(ctx context.Context, node *WorkflowNode, input interface{}, agent *Agent) (interface{}, error) {
	return fmt.Sprintf("output from node %s executed by agent %s", node.ID, agent.ID), nil
}

type WorkflowExecutor struct {
	agentSystem  *AgentSystem
	nodeExecutor NodeExecutor
	maxRetries   int
	timeout      time.Duration
	logger       core.Logger
}

type WorkflowResult struct {
	WorkflowID  string
	Status      WorkflowStatus
	Output      interface{}
	NodeResults map[string]*NodeResult
	Error       error
	StartTime   time.Time
	EndTime     time.Time
}

type NodeResult struct {
	NodeID     string
	AgentID    string
	Input      interface{}
	Output     interface{}
	Error      error
	StartTime  time.Time
	EndTime    time.Duration
	RetryCount int
}

type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusCompleted WorkflowStatus = "completed"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCancelled WorkflowStatus = "cancelled"
)

type WorkflowContext map[string]interface{}

func NewWorkflowExecutor(agentSystem *AgentSystem) *WorkflowExecutor {
	return &WorkflowExecutor{
		agentSystem:  agentSystem,
		nodeExecutor: &DefaultNodeExecutor{},
		maxRetries:   3,
		timeout:      5 * time.Minute,
		logger:       core.NopLogger{},
	}
}

func (e *WorkflowExecutor) WithNodeExecutor(exec NodeExecutor) *WorkflowExecutor {
	e.nodeExecutor = exec
	return e
}

func (e *WorkflowExecutor) WithMaxRetries(retries int) *WorkflowExecutor {
	e.maxRetries = retries
	return e
}

func (e *WorkflowExecutor) WithTimeout(timeout time.Duration) *WorkflowExecutor {
	e.timeout = timeout
	return e
}

func (e *WorkflowExecutor) WithLogger(logger core.Logger) *WorkflowExecutor {
	e.logger = logger
	return e
}

func (e *WorkflowExecutor) Execute(ctx context.Context, workflowName string, initialInput interface{}) (*WorkflowResult, error) {
	workflow, err := e.agentSystem.GetWorkflow(ctx, workflowName)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow %s: %w", workflowName, err)
	}

	return e.ExecuteWorkflow(ctx, workflow, initialInput)
}

func (e *WorkflowExecutor) ExecuteWorkflow(ctx context.Context, workflow *Workflow, initialInput interface{}) (*WorkflowResult, error) {
	result := &WorkflowResult{
		WorkflowID:  workflow.ID,
		Status:      WorkflowStatusRunning,
		NodeResults: make(map[string]*NodeResult),
		StartTime:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	nodeResults := make(map[string]interface{})
	currentInput := initialInput

	currentNode := e.findStartNode(workflow)
	if currentNode == nil {
		result.Status = WorkflowStatusFailed
		result.Error = fmt.Errorf("no start node found for workflow %s", workflow.ID)
		result.EndTime = time.Now()
		return result, result.Error
	}

	visited := make(map[string]int)
	var lastError error

	e.logger.Info("workflow started", "workflow_id", workflow.ID)

	for currentNode != nil {
		if ctx.Err() != nil {
			result.Status = WorkflowStatusCancelled
			result.Error = ctx.Err()
			break
		}

		visited[currentNode.ID]++

		if visited[currentNode.ID] > e.maxRetries {
			result.Status = WorkflowStatusFailed
			result.Error = fmt.Errorf("max retries exceeded for node %s", currentNode.ID)
			e.logger.Error("max retries exceeded", "node_id", currentNode.ID)
			break
		}

		e.logger.Debug("executing node", "node_id", currentNode.ID, "attempt", visited[currentNode.ID])

		nodeResult := e.executeNode(ctx, currentNode, currentInput, nodeResults)
		result.NodeResults[currentNode.ID] = nodeResult

		if nodeResult.Error != nil {
			lastError = nodeResult.Error
			e.logger.Warn("node execution failed", "node_id", currentNode.ID, "error", nodeResult.Error)

			errorEdge := e.findErrorEdge(workflow, currentNode)
			if errorEdge == nil {
				result.Status = WorkflowStatusFailed
				break
			}
			currentNode = e.findNodeByID(workflow, errorEdge.To)
			continue
		}

		nodeResults[currentNode.ID] = nodeResult.Output
		currentInput = nodeResult.Output

		e.logger.Debug("node completed", "node_id", currentNode.ID, "duration_ms", nodeResult.EndTime.Milliseconds())

		nextNode, err := e.findNextNode(ctx, workflow, currentNode, nodeResult.Output, nodeResults)
		if err != nil {
			lastError = err
			result.Status = WorkflowStatusFailed
			e.logger.Error("failed to find next node", "error", err)
			break
		}

		if nextNode == nil {
			break
		}

		currentNode = nextNode
	}

	result.EndTime = time.Now()

	if result.Status == WorkflowStatusRunning {
		if lastError != nil {
			result.Status = WorkflowStatusFailed
			result.Error = lastError
		} else {
			result.Status = WorkflowStatusCompleted
			result.Output = currentInput
		}
	}

	e.logger.Info("workflow completed", "workflow_id", workflow.ID, "status", result.Status, "duration_ms", result.EndTime.Sub(result.StartTime).Milliseconds())

	return result, nil
}

func (e *WorkflowExecutor) executeNode(ctx context.Context, node *WorkflowNode, input interface{}, previousResults map[string]interface{}) *NodeResult {
	result := &NodeResult{
		NodeID:    node.ID,
		Input:     input,
		StartTime: time.Now(),
	}

	agents, err := e.agentSystem.FindAgentsForTask(ctx, node.Agent)
	if err != nil || len(agents) == 0 {
		result.Error = fmt.Errorf("no agents found for role %s", node.Agent)
		return result
	}

	agent := agents[0]
	result.AgentID = agent.ID

	var execErr error
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		result.RetryCount = attempt

		output, err := e.nodeExecutor.Execute(ctx, node, input, agent)
		if err == nil {
			result.Output = output
			result.EndTime = time.Since(result.StartTime)
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
	result.EndTime = time.Since(result.StartTime)
	return result
}

func (e *WorkflowExecutor) findStartNode(workflow *Workflow) *WorkflowNode {
	for i := range workflow.Nodes {
		if e.isStartNode(workflow, &workflow.Nodes[i]) {
			return &workflow.Nodes[i]
		}
	}
	if len(workflow.Nodes) > 0 {
		return &workflow.Nodes[0]
	}
	return nil
}

func (e *WorkflowExecutor) isStartNode(workflow *Workflow, node *WorkflowNode) bool {
	for _, edge := range workflow.Edges {
		if edge.To == node.ID && edge.Condition == "" {
			return false
		}
	}
	return true
}

func (e *WorkflowExecutor) findNextNode(ctx context.Context, workflow *Workflow, currentNode *WorkflowNode, nodeOutput interface{}, previousResults map[string]interface{}) (*WorkflowNode, error) {
	var nextEdges []WorkflowEdge

	for i := range workflow.Edges {
		if workflow.Edges[i].From == currentNode.ID {
			nextEdges = append(nextEdges, workflow.Edges[i])
		}
	}

	if len(nextEdges) == 0 {
		return nil, nil
	}

	for _, edge := range nextEdges {
		if edge.Condition == "" {
			return e.findNodeByID(workflow, edge.To), nil
		}

		approved, err := e.evaluateCondition(ctx, edge.Condition, nodeOutput, previousResults)
		if err != nil {
			return nil, err
		}

		if approved {
			return e.findNodeByID(workflow, edge.To), nil
		}
	}

	return nil, nil
}

func (e *WorkflowExecutor) findErrorEdge(workflow *Workflow, node *WorkflowNode) *WorkflowEdge {
	for i := range workflow.Edges {
		if workflow.Edges[i].From == node.ID && strings.HasPrefix(workflow.Edges[i].Condition, "error") {
			return &workflow.Edges[i]
		}
	}
	return nil
}

func (e *WorkflowExecutor) findNodeByID(workflow *Workflow, nodeID string) *WorkflowNode {
	for i := range workflow.Nodes {
		if workflow.Nodes[i].ID == nodeID {
			return &workflow.Nodes[i]
		}
	}
	return nil
}

func (e *WorkflowExecutor) evaluateCondition(ctx context.Context, condition string, nodeOutput interface{}, previousResults map[string]interface{}) (bool, error) {
	if condition == "" {
		return true, nil
	}

	facts := []string{
		fmt.Sprintf(`context("current_output", "%v").`, formatValue(nodeOutput)),
	}

	for k, v := range previousResults {
		facts = append(facts, fmt.Sprintf(`context("%s", "%v").`, k, formatValue(v)))
	}

	results, err := e.agentSystem.Query(ctx, facts, condition)
	if err != nil {
		return false, err
	}

	return len(results) > 0, nil
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strings.ReplaceAll(val, `"`, `\"`)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", val)
	}
}

type ParallelWorkflowExecutor struct {
	executor    *WorkflowExecutor
	maxParallel int
	barrierSync bool
}

func NewParallelWorkflowExecutor(agentSystem *AgentSystem) *ParallelWorkflowExecutor {
	return &ParallelWorkflowExecutor{
		executor:    NewWorkflowExecutor(agentSystem),
		maxParallel: 3,
		barrierSync: false,
	}
}

func (e *ParallelWorkflowExecutor) WithMaxParallel(max int) *ParallelWorkflowExecutor {
	e.maxParallel = max
	return e
}

func (e *ParallelWorkflowExecutor) WithBarrierSync(barrier bool) *ParallelWorkflowExecutor {
	e.barrierSync = barrier
	return e
}

func (e *ParallelWorkflowExecutor) ExecuteParallel(ctx context.Context, workflow *Workflow, initialInput interface{}) (*WorkflowResult, error) {
	result := &WorkflowResult{
		WorkflowID:  workflow.ID,
		Status:      WorkflowStatusRunning,
		NodeResults: make(map[string]*NodeResult),
		StartTime:   time.Now(),
	}

	parallelGroups := e.identifyParallelGroups(workflow)
	if len(parallelGroups) == 0 {
		return e.executor.ExecuteWorkflow(ctx, workflow, initialInput)
	}

	nodeResults := make(map[string]interface{})
	currentInput := initialInput

	for _, group := range parallelGroups {
		groupResult, err := e.executeParallelGroup(ctx, workflow, group, currentInput, nodeResults)
		if err != nil {
			result.Status = WorkflowStatusFailed
			result.Error = err
			result.EndTime = time.Now()
			return result, err
		}

		for k, v := range groupResult {
			nodeResults[k] = v
		}

		currentInput = e.mergeOutputs(groupResult)
	}

	result.Status = WorkflowStatusCompleted
	result.Output = currentInput
	result.EndTime = time.Now()

	return result, nil
}

func (e *ParallelWorkflowExecutor) identifyParallelGroups(workflow *Workflow) [][]string {
	inDegree := make(map[string]int)

	for _, node := range workflow.Nodes {
		inDegree[node.ID] = 0
	}

	for _, edge := range workflow.Edges {
		if edge.Condition == "" {
			inDegree[edge.To]++
		}
	}

	var groups [][]string
	var current []string

	for {
		current = nil
		for nodeID, degree := range inDegree {
			if degree == 0 {
				current = append(current, nodeID)
			}
		}

		if len(current) == 0 {
			break
		}

		if len(current) > 1 {
			groups = append(groups, current)
		}

		for _, nodeID := range current {
			for _, edge := range workflow.Edges {
				if edge.From == nodeID && edge.Condition == "" {
					inDegree[edge.To]--
				}
			}
			delete(inDegree, nodeID)
		}
	}

	return groups
}

func (e *ParallelWorkflowExecutor) executeParallelGroup(ctx context.Context, workflow *Workflow, nodeIDs []string, input interface{}, previousResults map[string]interface{}) (map[string]interface{}, error) {
	type result struct {
		nodeID string
		output interface{}
		err    error
	}

	resultChan := make(chan result, len(nodeIDs))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, e.maxParallel)

	for _, nodeID := range nodeIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			node := e.executor.findNodeByID(workflow, id)
			if node == nil {
				resultChan <- result{nodeID: id, err: fmt.Errorf("node not found: %s", id)}
				return
			}

			nodeResult := e.executor.executeNode(ctx, node, input, previousResults)
			resultChan <- result{nodeID: id, output: nodeResult.Output, err: nodeResult.Error}
		}(nodeID)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	outputs := make(map[string]interface{})
	var errs []error
	errMu := sync.Mutex{}

	for r := range resultChan {
		if r.err != nil {
			errMu.Lock()
			errs = append(errs, fmt.Errorf("node %s: %w", r.nodeID, r.err))
			errMu.Unlock()
		}
		outputs[r.nodeID] = r.output
	}

	if len(errs) > 0 {
		return outputs, errors.Join(errs...)
	}
	return outputs, nil
}

func (e *ParallelWorkflowExecutor) mergeOutputs(outputs map[string]interface{}) interface{} {
	if len(outputs) == 0 {
		return nil
	}

	var merged []string
	for _, v := range outputs {
		if s, ok := v.(string); ok {
			merged = append(merged, s)
		}
	}

	if len(merged) > 0 {
		return merged
	}
	return outputs
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
