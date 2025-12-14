package sdk

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ReflectReq struct{}
type ReflectRes struct{}

func TestActionReflection(t *testing.T) {
	// Create a dummy policy file to initialize the engine runtime
	tmpFile, err := os.CreateTemp("", "policy*.dl")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write a valid but empty rule/fact to initialize
	_, err = tmpFile.WriteString("init(\"true\").")
	require.NoError(t, err)
	tmpFile.Close()

	client, err := NewClient(context.Background(), WithBlueprintPath(tmpFile.Name()))
	require.NoError(t, err)

	handler := func(ctx context.Context, in ReflectReq) (ReflectRes, error) {
		return ReflectRes{}, nil
	}

	// Define calls RegisterAction which calls RegisterActionMetadata
	Define(client, "reflect_test", handler)

	// Access internal engine directly since we are in package sdk
	eng := client.engine

	ctx := context.Background()

	// Check action("reflect_test")
	// Type assert to verify engine specific query methods
	// We use the same Queryable interface trick or just verify interface compliance
	// if we added ExecuteQuery to Evaluator. We didn't.
	// We need to type assert to *engine.PolicyEngine or an interface that has ExecuteQuery.
	// But `engine` package is internal. sdk_test package (if it was separate) couldn't see it.
	// `package sdk` can see `internal/engine` because we import it in client.go.
	// However, `Client.engine` is now `core.Evaluator`.

	type QueryExecutor interface {
		ExecuteQuery(ctx context.Context, facts []any, queryStr string) (bool, error)
	}
	// Note: ExecuteQuery signature in PolicyEngine uses []ast.Atom, which is internal to Mangle.
	// We cannot easily interface it without importing ast.Atom or using `any`.
	// But wait, the test imports `eng.ExecuteQuery`.
	// `eng` is `core.Evaluator`.
	// Let's check if we can cast `eng` to something useful.
	// Since we are inside `sdk`, we can import `github.com/duynguyendang/manglekit/internal/engine` (already imported in client.go).

	// But `Client.engine` is private. We accessed it via `client.engine`.
	// So we can type assert it.

	// pe, ok := eng.(interface {
	// 	ExecuteQuery(ctx context.Context, facts []any, queryStr string) (bool, error)
	// })
	// Actually PolicyEngine.ExecuteQuery signature is:
	// func (e *PolicyEngine) ExecuteQuery(ctx context.Context, facts []ast.Atom, queryStr string) (bool, error)
	// ast.Atom comes from "github.com/google/mangle/ast".
	// The test doesn't import that.
	// If we want to fix this test, we should probably update PolicyEngine to use a friendlier signature or import ast.
	//
	// Alternatively, use `Query` which returns map[string]string solutions.
	// Query signature: Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)
	// We should add `Query` to `core.Evaluator` interface? No, it's specific.
	//
	// Let's use the `Queryable` interface we used in Planner.
	type Queryable interface {
		Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error)
	}
	qEng, ok := eng.(Queryable)
	require.True(t, ok, "engine must support Query")

	// Check action("reflect_test")
	res, err := qEng.Query(ctx, nil, `action("reflect_test")`)
	require.NoError(t, err)
	assert.NotEmpty(t, res, "action fact not found")

	// Check has_input
	res, err = qEng.Query(ctx, nil, `has_input("reflect_test", "ReflectReq")`)
	require.NoError(t, err)
	assert.NotEmpty(t, res, "has_input fact not found")

	// Check has_output
	res, err = qEng.Query(ctx, nil, `has_output("reflect_test", "ReflectRes")`)
	require.NoError(t, err)
	assert.NotEmpty(t, res, "has_output fact not found")
}
