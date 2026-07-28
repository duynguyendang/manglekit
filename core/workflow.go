package core

import (
	"time"
)

type WorkflowDef struct {
	ID         string
	Name       string
	Version    string
	RootNodeID string
	Nodes      map[string]NodeDef
	Edges      []EdgeDef
}

type NodeDef struct {
	ID        string
	AgentRole string
	TaskType  string
	Config    map[string]interface{}
}

type EdgeDef struct {
	From      string
	To        string
	Condition string
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
	Duration   time.Duration
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

func (w *WorkflowDef) GetNode(nodeID string) *NodeDef {
	if w == nil {
		return nil
	}
	node, ok := w.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

func (w *WorkflowDef) GetOutgoingEdges(nodeID string) []EdgeDef {
	var edges []EdgeDef
	for _, edge := range w.Edges {
		if edge.From == nodeID {
			edges = append(edges, edge)
		}
	}
	return edges
}

func (w *WorkflowDef) FindStartNode() *NodeDef {
	if w.RootNodeID != "" {
		if node := w.GetNode(w.RootNodeID); node != nil {
			return node
		}
	}

	hasIncoming := make(map[string]bool)
	for _, edge := range w.Edges {
		hasIncoming[edge.To] = true
	}

	var best *NodeDef
	for id, node := range w.Nodes {
		if !hasIncoming[id] {
			if best == nil || id < best.ID {
				n := node
				best = &n
			}
		}
	}
	if best != nil {
		return best
	}

	for id, node := range w.Nodes {
		if best == nil || id < best.ID {
			n := node
			best = &n
		}
	}
	return best
}

func (w *WorkflowDef) Validate() error {
	if w == nil {
		return nil
	}

	if len(w.Nodes) == 0 {
		return nil
	}

	nodeIDs := make(map[string]bool)
	for id := range w.Nodes {
		nodeIDs[id] = true
	}

	for _, edge := range w.Edges {
		if !nodeIDs[edge.From] {
			return &ValidationError{Field: "edge", Message: "edge references unknown node: " + edge.From}
		}
		if !nodeIDs[edge.To] {
			return &ValidationError{Field: "edge", Message: "edge references unknown node: " + edge.To}
		}
	}

	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// WorkflowInstance represents the dynamic state of a workflow execution.
// This is separate from WorkflowDef (which is immutable once hydrated).
// WorkflowInstance is stored in Session Memory (RAM) and NOT persisted to MEB.
type WorkflowInstance struct {
	WorkflowID     string                 // Reference to WorkflowDef
	SessionID      string                 // Unique session identifier
	CurrentNodeID  string                 // Current node being executed
	Variables      map[string]interface{} // Session variables (transient)
	Metadata       map[string]string      // Execution metadata
	LoopCounters   map[string]int         // Loop/iteration counters
	CompletedNodes []string               // Nodes that have completed
	FailedNodes    []string               // Nodes that failed
	Status         WorkflowInstanceStatus
	StartedAt      time.Time
	UpdatedAt      time.Time
}

type WorkflowInstanceStatus string

const (
	WorkflowInstanceStatusPending   WorkflowInstanceStatus = "pending"
	WorkflowInstanceStatusRunning   WorkflowInstanceStatus = "running"
	WorkflowInstanceStatusPaused    WorkflowInstanceStatus = "paused"
	WorkflowInstanceStatusCompleted WorkflowInstanceStatus = "completed"
	WorkflowInstanceStatusFailed    WorkflowInstanceStatus = "failed"
)

// NewWorkflowInstance creates a new workflow instance with default values.
func NewWorkflowInstance(workflowID, sessionID string) *WorkflowInstance {
	now := time.Now()
	return &WorkflowInstance{
		WorkflowID:   workflowID,
		SessionID:    sessionID,
		Variables:    make(map[string]interface{}),
		Metadata:     make(map[string]string),
		LoopCounters: make(map[string]int),
		Status:       WorkflowInstanceStatusPending,
		StartedAt:    now,
		UpdatedAt:    now,
	}
}

// SetCurrentNode updates the current node and marks the previous node as completed.
func (wi *WorkflowInstance) SetCurrentNode(nodeID string) {
	if wi.CurrentNodeID != "" && !wi.IsNodeCompleted(wi.CurrentNodeID) {
		wi.CompletedNodes = append(wi.CompletedNodes, wi.CurrentNodeID)
	}
	wi.CurrentNodeID = nodeID
	wi.UpdatedAt = time.Now()
}

// MarkNodeCompleted marks a node as completed.
func (wi *WorkflowInstance) MarkNodeCompleted(nodeID string) {
	if !wi.IsNodeCompleted(nodeID) {
		wi.CompletedNodes = append(wi.CompletedNodes, nodeID)
	}
	wi.UpdatedAt = time.Now()
}

// MarkNodeFailed marks a node as failed.
func (wi *WorkflowInstance) MarkNodeFailed(nodeID string) {
	wi.FailedNodes = append(wi.FailedNodes, nodeID)
	wi.UpdatedAt = time.Now()
}

// IncrementCounter increments a loop counter.
func (wi *WorkflowInstance) IncrementCounter(name string) int {
	wi.LoopCounters[name]++
	wi.UpdatedAt = time.Now()
	return wi.LoopCounters[name]
}

// GetCounter returns the current value of a counter.
func (wi *WorkflowInstance) GetCounter(name string) int {
	return wi.LoopCounters[name]
}

// SetVariable sets a session variable.
func (wi *WorkflowInstance) SetVariable(key string, value interface{}) {
	wi.Variables[key] = value
	wi.UpdatedAt = time.Now()
}

// GetVariable returns a session variable.
func (wi *WorkflowInstance) GetVariable(key string) interface{} {
	return wi.Variables[key]
}

// IsNodeCompleted checks if a node has been completed.
func (wi *WorkflowInstance) IsNodeCompleted(nodeID string) bool {
	for _, n := range wi.CompletedNodes {
		if n == nodeID {
			return true
		}
	}
	return false
}

// SessionKey returns the key for storing this instance in session memory.
func (wi *WorkflowInstance) SessionKey() string {
	return wi.SessionID + ":" + wi.WorkflowID
}
