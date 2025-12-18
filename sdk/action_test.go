package sdk_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestInput struct {
	Value int
}

type TestOutput struct {
	Result int
}

func TestContextFacts(t *testing.T) {
	ctx := context.Background()
	ctx = core.WithFact(ctx, "role", "admin")
	ctx = core.WithFact(ctx, "region", "us-east")

	facts := core.ContextFacts(ctx)
	assert.Equal(t, "admin", facts["role"])
	assert.Equal(t, "us-east", facts["region"])
}

func TestDefineAndRun(t *testing.T) {
	client, err := sdk.NewDefault()
	require.NoError(t, err)

	handler := func(ctx context.Context, in TestInput) (TestOutput, error) {
		facts := core.ContextFacts(ctx)
		if facts["test_key"] != "test_value" {
			return TestOutput{}, errors.New("context facts missing")
		}

		return TestOutput{Result: in.Value * 2}, nil
	}

	action := sdk.Define(client, "double", handler)

	ctx := context.Background()
	ctx = core.WithFact(ctx, "test_key", "test_value")

	res, err := action.Run(ctx, TestInput{Value: 5})
	require.NoError(t, err)
	assert.Equal(t, 10, res.Result)
}

func TestRunError(t *testing.T) {
	client, err := sdk.NewDefault()
	require.NoError(t, err)

	handler := func(ctx context.Context, in TestInput) (TestOutput, error) {
		return TestOutput{}, errors.New("failure")
	}

	action := sdk.Define(client, "fail", handler)
	ctx := context.Background()
	_, err = action.Run(ctx, TestInput{Value: 1})
	assert.Error(t, err)
	// Guard wraps errors
	assert.True(t, strings.Contains(err.Error(), "failure"))
}
