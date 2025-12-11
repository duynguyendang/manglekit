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
	found, err := eng.ExecuteQuery(ctx, nil, `action("reflect_test")`)
	require.NoError(t, err)
	assert.True(t, found, "action fact not found")

	// Check has_input
	found, err = eng.ExecuteQuery(ctx, nil, `has_input("reflect_test", "ReflectReq")`)
	require.NoError(t, err)
	assert.True(t, found, "has_input fact not found")

	// Check has_output
	found, err = eng.ExecuteQuery(ctx, nil, `has_output("reflect_test", "ReflectRes")`)
	require.NoError(t, err)
	assert.True(t, found, "has_output fact not found")
}
