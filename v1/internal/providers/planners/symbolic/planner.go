// Package symbolic provides a symbolic planner implementation that uses
// a reasoner to generate multi-step plans.
package symbolic

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/duynguyendang/manglekit/v1/core"
)

// SymbolicPlanner implements core.Planner using a symbolic reasoner.
// It converts queries into logical facts, executes a reasoner, and
// constructs a structured plan from the reasoner's output.
type SymbolicPlanner struct {
	log      core.Logger
	reasoner core.Reasoner
}

// Plan generates a step-by-step plan by executing a symbolic reasoner.
// It converts the query into input facts, executes the reasoner, and
// parses the output facts into a structured core.Plan.
//
// The reasoner is expected to return facts in the format:
// - plan_step(Order, ToolName, ParamsJSON, Reason)
// Where Order is an integer indicating the step sequence.
func (p *SymbolicPlanner) Plan(ctx context.Context, q core.Query) (core.Plan, error) {
	p.log.Debugf("Generating plan for query", "query", q.Text)

	// Convert Query to Facts
	// Transform the query into input facts for the reasoner
	inputFacts := make(map[string]any)
	inputFacts["query_text"] = q.Text

	// Include query metadata if present
	if q.Meta != nil {
		for k, v := range q.Meta {
			inputFacts[fmt.Sprintf("query_meta_%s", k)] = v
		}
	}

	// Create Reasoner Request
	req := core.ReasonerRequest{
		Input: inputFacts,
	}

	// Execute Reasoner
	p.log.Debugf("Executing reasoner", "input_facts", len(inputFacts))
	resp, err := p.reasoner.Execute(ctx, req)
	if err != nil {
		p.log.Errorf("Reasoner execution failed", "error", err)
		return core.Plan{}, fmt.Errorf("reasoner execution failed: %w", err)
	}

	// Parse Response
	// The reasoner's output is expected to contain facts that define the plan
	// Expected format: plan_step_<order>, plan_tool_<order>, plan_params_<order>, plan_reason_<order>
	steps, err := p.parseSteps(resp.Output)
	if err != nil {
		p.log.Errorf("Failed to parse reasoner output", "error", err)
		return core.Plan{}, fmt.Errorf("failed to parse reasoner output: %w", err)
	}

	if len(steps) == 0 {
		p.log.Warnf("Reasoner returned no plan steps")
		return core.Plan{}, fmt.Errorf("reasoner returned no plan steps")
	}

	p.log.Infof("Generated plan with %d steps", len(steps))
	return core.Plan{Steps: steps}, nil
}

// planStep is a temporary structure for sorting and building the final plan.
type planStep struct {
	Order  int
	Tool   string
	Params map[string]any
	Reason string
}

// parseSteps extracts plan steps from the reasoner's output map.
// It looks for keys like plan_step_0, plan_tool_0, etc. and constructs
// a sorted list of core.Step.
func (p *SymbolicPlanner) parseSteps(output map[string]any) ([]core.Step, error) {
	// Collect plan steps by order
	stepMap := make(map[int]*planStep)

	for key, value := range output {
		// Parse plan_step_<order> keys
		var order int
		var fieldType string

		// Try to parse plan_step_<order>
		n, err := fmt.Sscanf(key, "plan_step_%d", &order)
		if n == 1 && err == nil {
			fieldType = "step"
		} else {
			// Try plan_tool_<order>
			n, err = fmt.Sscanf(key, "plan_tool_%d", &order)
			if n == 1 && err == nil {
				fieldType = "tool"
			} else {
				// Try plan_params_<order>
				n, err = fmt.Sscanf(key, "plan_params_%d", &order)
				if n == 1 && err == nil {
					fieldType = "params"
				} else {
					// Try plan_reason_<order>
					n, err = fmt.Sscanf(key, "plan_reason_%d", &order)
					if n == 1 && err == nil {
						fieldType = "reason"
					} else {
						// Not a plan-related key, skip
						continue
					}
				}
			}
		}

		// Ensure step entry exists
		if _, exists := stepMap[order]; !exists {
			stepMap[order] = &planStep{Order: order}
		}

		// Set the appropriate field
		switch fieldType {
		case "step":
			// This might be redundant but captures order explicitly
			if orderStr, ok := value.(string); ok {
				if parsedOrder, err := strconv.Atoi(orderStr); err == nil {
					stepMap[order].Order = parsedOrder
				}
			}
		case "tool":
			if toolName, ok := value.(string); ok {
				stepMap[order].Tool = toolName
			}
		case "params":
			// Parse params - can be JSON string or map
			switch v := value.(type) {
			case string:
				var params map[string]any
				if err := json.Unmarshal([]byte(v), &params); err != nil {
					p.log.Warnf("Failed to parse params JSON for step %d: %v", order, err)
					params = map[string]any{"raw": v}
				}
				stepMap[order].Params = params
			case map[string]any:
				stepMap[order].Params = v
			default:
				p.log.Warnf("Unexpected params type for step %d: %T", order, v)
			}
		case "reason":
			if reason, ok := value.(string); ok {
				stepMap[order].Reason = reason
			}
		}
	}

	if len(stepMap) == 0 {
		return nil, fmt.Errorf("no plan steps found in reasoner output")
	}

	// Convert to sorted slice
	var steps []core.Step
	orders := make([]int, 0, len(stepMap))
	for order := range stepMap {
		orders = append(orders, order)
	}
	sort.Ints(orders)

	for _, order := range orders {
		ps := stepMap[order]
		if ps.Tool == "" {
			p.log.Warnf("Step %d missing tool name, skipping", order)
			continue
		}
		steps = append(steps, core.Step{
			Tool:   ps.Tool,
			Params: ps.Params,
			Reason: ps.Reason,
		})
	}

	return steps, nil
}
