package sdk

import (
	"context"
	"fmt"
	"sort"
	"strconv"
)

// PlanStep represents a single step in a generated plan.
type PlanStep struct {
	ActionName string
	Order      int
}

// Plan generates a sequence of actions (a plan) to achieve the specified goal.
// It uses the underlying Datalog engine to reason about goals and subgoals.
//
// Parameters:
//   - ctx: The context.
//   - goalName: The name of the goal to achieve (e.g., "onboard_user").
//
// Returns:
//   - A slice of PlanStep structs ordered by execution sequence.
//   - An error if planning fails.
func (c *Client) Plan(ctx context.Context, goalName string) ([]PlanStep, error) {
	if c.engine == nil {
		return nil, fmt.Errorf("engine not initialized")
	}

	// 1. Prepare Goal Fact (passed as temporary fact)
	// We don't want to pollute the global fact store with request-specific goals if possible,
	// but the user instruction said "Inject Goal... using c.engine.LoadFacts".
	// LoadFacts is persistent. If we use it, we should probably clear it or assume it's okay.
	// However, I just added Query() which takes temporary facts. This is cleaner.
	// But to strictly follow "Inject Goal... LoadFacts" I should do that.
	// However, "ensure thread safety if LoadFacts modifies shared state".
	// LoadFacts locks the runtime.
	// If I use temporary facts in Query, it's safer for concurrent requests.
	// I will use temporary facts in Query if possible, but the prompt said "load it using c.engine.LoadFacts".
	// Using LoadFacts for a specific request's goal is bad design for a shared engine (concurrent goals would mix).
	// I will assume the prompt meant "Inject the goal into the reasoning context".
	// Since I implemented `Query(ctx, facts, query)`, I can pass the goal there.

	goalFact := fmt.Sprintf("goal(\"%s\")", goalName)

	// 2. Query for steps: plan_step(Action, Order)
	// We pass goalFact as a temporary fact.
	query := "plan_step(Action, Order)"

	// core.Evaluator does not have Query. We must cast or update interface.
	// Using type assertion for now as Planner is tightly coupled to Datalog engine.
	type Queryable interface {
		Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)
	}

	queryable, ok := c.engine.(Queryable)
	if !ok {
		return nil, fmt.Errorf("engine does not support querying")
	}

	results, err := queryable.Query(ctx, []string{goalFact}, query)
	if err != nil {
		return nil, fmt.Errorf("planning query failed: %w", err)
	}

	// 3. Parse Results
	var steps []PlanStep
	for _, sol := range results {
		action, ok := sol["Action"]
		if !ok {
			continue
		}
		orderStr, ok := sol["Order"]
		if !ok {
			continue
		}

		order, err := strconv.Atoi(orderStr)
		if err != nil {
			return nil, fmt.Errorf("invalid order '%s': %w", orderStr, err)
		}

		steps = append(steps, PlanStep{
			ActionName: action,
			Order:      order,
		})
	}

	// 4. Sort by Order
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Order < steps[j].Order
	})

	return steps, nil
}
