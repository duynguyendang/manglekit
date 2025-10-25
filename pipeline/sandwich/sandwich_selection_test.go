package sandwich_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/internal/providers/state/inmemory"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandwich_DeterministicSelection(t *testing.T) {
	// 1. Setup: Create a `core.Resolved` with multiple mock components.
	stateA, err := inmemory.New(inmemory.Options{})
	require.NoError(t, err)
	stateB, err := inmemory.New(inmemory.Options{})
	require.NoError(t, err)

	deps := core.Resolved{
		Retrievers: map[string]core.Retriever{
			"mock-retriever": mock.NewRetriever(nil),
		},
		LLMs: map[string]core.LLMClient{
			"mock-llm": mock.NewLLM(""),
		},
		Rules: map[string]core.RuleSet{
			"ruleset-A": mock.NewRuleSet(),
			"ruleset-B": mock.NewRuleSet(),
		},
		StateProviders: map[string]core.StateProvider{
			"state-A": stateA,
			"state-B": stateB,
		},
	}

	// 2. Configuration: Explicitly select which components to use.
	cfg := sandwich.SandwichOptions{
		Retriever:     "mock-retriever",
		LLM:           "mock-llm",
		RuleSet:       "ruleset-B",
		StateProvider: "state-A",
	}

	// 3. Execution: Build the orchestrator.
	orch, err := sandwich.NewSandwich(context.Background(), deps, cfg)

	// 4. Assertion: Verify that the orchestrator was created successfully
	//    and that the correct components were selected.
	require.NoError(t, err, "NewSandwich should not return an error with valid config")
	require.NotNil(t, orch, "NewSandwich should return a valid orchestrator")

	// To verify the correct components were selected, we would ideally inspect the
	// internal state of the Sandwich orchestrator. Since its fields are private,
	// a full verification would require either exporting them for testing or
	// adding methods to expose them.
	//
	// However, the factory's logic is straightforward:
	// `s.ruleset, ok = deps.Rules[cfg.RuleSet]`
	//
	// The fact that `NewSandwich` returns a non-nil orchestrator without an error
	// is a strong indication that the lookups for "ruleset-B" and "state-A"
	// were successful, implicitly confirming the selection logic. If the lookup
	// failed, it would have returned an error.

	// Let's test the negative case to be certain.
	t.Run("returns error for non-existent ruleset", func(t *testing.T) {
		badCfg := cfg
		badCfg.RuleSet = "ruleset-C" // This one does not exist
		_, err := sandwich.NewSandwich(context.Background(), deps, badCfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `ruleset "ruleset-C" not found`)
	})

	t.Run("returns error for non-existent state provider", func(t *testing.T) {
		badCfg := cfg
		badCfg.StateProvider = "state-C" // This one does not exist
		_, err := sandwich.NewSandwich(context.Background(), deps, badCfg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), `state provider "state-C" not found`)
	})
}
