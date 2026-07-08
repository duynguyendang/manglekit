// Package scenario provides a tiny, dependency-free test harness for
// running behavior-driven policy scenarios against a Manglekit client.
//
// A Scenario bundles a human-readable Name, a Run function that exercises
// the system under test, and an expectation about whether that run should be
// blocked (a non-nil error) or allowed (a nil error). Run executes every
// scenario and reports failures via t.Error-equivalent logging so it can be
// driven from a normal main or a test.
package scenario

import (
	"context"
	"fmt"
	"os"
)

// Scenario describes a single behavior-driven check.
//
// Exactly one of WantBlocked/WantAllowed should normally be set to assert the
// expected outcome of Run. If neither is set, Run is executed and any error is
// treated as a failure (fail-closed by default).
type Scenario struct {
	// Name is a human-readable label printed for the scenario.
	Name string

	// Run exercises the system under test and returns an error if the
	// scenario's own internal assertions fail. For policy-gated flows,
	// Run typically calls into the engine and returns an error when the
	// observed outcome differs from the expected one.
	Run func(ctx context.Context) error

	// WantBlocked asserts that Run returns a non-nil error (the action was
	// halted/blocked by policy).
	WantBlocked bool

	// WantAllowed asserts that Run returns a nil error (the action was
	// permitted by policy).
	WantAllowed bool
}

// Result summarizes the outcome of a single scenario execution.
type Result struct {
	Scenario Scenario
	Passed   bool
	Err      error
}

// Run executes each scenario in order with the provided context. It returns a
// slice with one Result per scenario. A scenario passes when its outcome
// matches the declared expectation:
//
//   - WantBlocked:  Run must return a non-nil error.
//   - WantAllowed:  Run must return a nil error.
//   - neither set:  Run must return a nil error (fail-closed default).
//
// Run never aborts early; every scenario is executed so a single failing
// scenario does not mask the rest.
func Run(ctx context.Context, scenarios []Scenario) []Result {
	results := make([]Result, 0, len(scenarios))
	for _, s := range scenarios {
		result := Result{Scenario: s}
		err := s.Run(ctx)
		result.Err = err

		switch {
		case s.WantBlocked:
			result.Passed = err != nil
		case s.WantAllowed:
			result.Passed = err == nil
		default:
			result.Passed = err == nil
		}

		results = append(results, result)
	}
	return results
}

// RunMain executes the scenarios and, if any fail, prints a summary and exits
// with a non-zero status so CI catches the regression. It is intended to be
// called from a package main.
func RunMain(ctx context.Context, scenarios []Scenario) {
	results := Run(ctx, scenarios)

	failures := 0
	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
			failures++
		}
		if r.Err != nil {
			fmt.Printf("[%s] %s: %v\n", status, r.Scenario.Name, r.Err)
		} else {
			fmt.Printf("[%s] %s\n", status, r.Scenario.Name)
		}
	}

	if failures > 0 {
		fmt.Printf("\n%d scenario(s) FAILED.\n", failures)
		os.Exit(1)
	}
	fmt.Println("\nAll scenarios passed.")
}
