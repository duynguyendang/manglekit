package mcp

import (
	"context"
	"testing"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockClient mocks the MCP Client.
type MockClient struct {
	mock.Mock
}

func (m *MockClient) GetActiveTools(ctx context.Context, g *genkit.Genkit) ([]ai.Tool, error) {
	args := m.Called(ctx, g)
	return args.Get(0).([]ai.Tool), args.Error(1)
}

// MockFactory mocks the ClientFactory.
type MockFactory struct {
	mock.Mock
}

func (m *MockFactory) CreateClient(ctx context.Context, cfg config.MCPServerConfig) (Client, error) {
	args := m.Called(ctx, cfg)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(Client), args.Error(1)
}

func TestLoad(t *testing.T) {
	// Setup mocks
	mockClient := new(MockClient)
	mockFactory := new(MockFactory)
	ctx := context.Background()

	// Define tool using ai.NewTool
	testTool := ai.NewTool[any, any](
		"calculator",
		"A calculator",
		func(ctx *ai.ToolContext, input any) (any, error) {
			return "42", nil
		},
	)

	// Expect factory to create client
	cfg := config.MCPServerConfig{
		Name:      "test_server",
		Transport: "stdio",
	}
	mockFactory.On("CreateClient", ctx, cfg).Return(mockClient, nil)

	// Expect client to return tools
	mockClient.On("GetActiveTools", ctx, mock.Anything).Return([]ai.Tool{testTool}, nil)

	// Execute Load
	actions, err := LoadWithFactory(ctx, []config.MCPServerConfig{cfg}, mockFactory, core.NopLogger{})

	// Verify
	assert.NoError(t, err)
	assert.Len(t, actions, 1)
	assert.Equal(t, "mcp_test_server_calculator", actions[0].Metadata().Name)

	// Verify Execution
	res, err := actions[0].Execute(ctx, core.NewEnvelope("calculate"))
	assert.NoError(t, err)
	assert.Equal(t, "42", res.Payload)

	mockFactory.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}
