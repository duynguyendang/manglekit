package sdk_test

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAction for testing
type MockAction struct {
	Func func(ctx context.Context, env core.Envelope) (core.Envelope, error)
}

func (m *MockAction) Execute(ctx context.Context, env core.Envelope) (core.Envelope, error) {
	return m.Func(ctx, env)
}

func (m *MockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "mock"}
}

type TestReq struct {
	Val string `json:"val"`
}

type TestResp struct {
	Result string `json:"result"`
}

func TestExecute_Typed_FastPath(t *testing.T) {
	ctx := context.Background()
	client, err := sdk.NewClient(context.Background())
	require.NoError(t, err)

	handle := sdk.DefineAction[TestReq, TestResp]("test_fast")

	// Register action that returns the struct directly
	client.RegisterAction("test_fast", &MockAction{
		Func: func(ctx context.Context, env core.Envelope) (core.Envelope, error) {
			return core.NewEnvelope(TestResp{Result: "fast"}), nil
		},
	})

	input := TestReq{Val: "in"}
	resp, err := sdk.Execute(ctx, client, handle, input)
	require.NoError(t, err)
	assert.Equal(t, "fast", resp.Result)
}

func TestExecute_Typed_SlowPath(t *testing.T) {
	ctx := context.Background()
	client, err := sdk.NewClient(context.Background())
	require.NoError(t, err)

	handle := sdk.DefineAction[TestReq, TestResp]("test_slow")

	// Register action that returns a map (simulating JSON/Serialization)
	client.RegisterAction("test_slow", &MockAction{
		Func: func(ctx context.Context, env core.Envelope) (core.Envelope, error) {
			return core.NewEnvelope(map[string]any{
				"result": "slow",
			}), nil
		},
	})

	input := TestReq{Val: "in"}
	resp, err := sdk.Execute(ctx, client, handle, input)
	require.NoError(t, err)
	assert.Equal(t, "slow", resp.Result)
}

func TestExecute_Typed_ExecutionError(t *testing.T) {
	ctx := context.Background()
	client, err := sdk.NewClient(context.Background())
	require.NoError(t, err)

	handle := sdk.DefineAction[TestReq, TestResp]("test_error")

	client.RegisterAction("test_error", &MockAction{
		Func: func(ctx context.Context, env core.Envelope) (core.Envelope, error) {
			return core.Envelope{}, assert.AnError
		},
	})

	input := TestReq{Val: "in"}
	_, err = sdk.Execute(ctx, client, handle, input)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
}

func TestExecute_Typed_ConversionError(t *testing.T) {
	ctx := context.Background()
	client, err := sdk.NewClient(context.Background())
	require.NoError(t, err)

	handle := sdk.DefineAction[TestReq, TestResp]("test_bad_type")

	// Return data that cannot be marshaled into TestResp
	// e.g. "result" field is expected to be string, but we give a number
	client.RegisterAction("test_bad_type", &MockAction{
		Func: func(ctx context.Context, env core.Envelope) (core.Envelope, error) {
			return core.NewEnvelope(map[string]any{
				"result": 12345,
			}), nil
		},
	})

	input := TestReq{Val: "in"}
	_, err = sdk.Execute(ctx, client, handle, input)
	assert.Error(t, err)
	// The error comes from json.Unmarshal because 12345 cannot be unmarshaled into string field "result"
	// (unless using loose unmarshal, but standard json is strict about types if not string/int confusion)
	// Actually 12345 to string might fail.
}
