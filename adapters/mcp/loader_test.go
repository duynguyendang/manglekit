package mcp

import (
	"context"
	"fmt"
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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

func TestLoader_Load(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid Server", func(t *testing.T) {
		mockClient := new(MockClient)
		mockFactory := new(MockFactory)
		cfg := config.MCPServerConfig{Name: "valid-server"}

		testTool := ai.NewTool[any, any]("tool", "desc", func(ctx *ai.ToolContext, input any) (any, error) { return nil, nil })

		mockFactory.On("CreateClient", ctx, cfg).Return(mockClient, nil)
		mockClient.On("GetActiveTools", ctx, mock.Anything).Return([]ai.Tool{testTool}, nil)

		loader := NewLoader(cfg).WithFactory(mockFactory)
		actions, err := loader.Load(ctx)

		assert.NoError(t, err)
		assert.Len(t, actions, 1)
	})

	t.Run("Error Server FailOnStartup True", func(t *testing.T) {
		mockFactory := new(MockFactory)
		cfg := config.MCPServerConfig{
			Name:          "error-server-fatal",
			FailOnStartup: true,
		}

		mockFactory.On("CreateClient", ctx, cfg).Return(nil, fmt.Errorf("connect error"))

		loader := NewLoader(cfg).WithFactory(mockFactory)
		actions, err := loader.Load(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load MCP server")
		assert.Nil(t, actions)
	})

	t.Run("Error Server FailOnStartup False (Default)", func(t *testing.T) {
		mockFactory := new(MockFactory)
		cfg := config.MCPServerConfig{
			Name:          "error-server-soft",
			FailOnStartup: false,
			Tools:         []string{"weather"}, // Expected tool
		}

		mockFactory.On("CreateClient", ctx, cfg).Return(nil, fmt.Errorf("connect error"))

		loader := NewLoader(cfg).WithFactory(mockFactory)
		actions, err := loader.Load(ctx)

		// Should NOT return error
		assert.NoError(t, err)
		// Should return 1 unhealthy action
		assert.Len(t, actions, 1)

		unhealthyAction := actions[0]
		assert.Equal(t, "mcp_error-server-soft_weather", unhealthyAction.Metadata().Name)

		// Execute should return unavailable error
		res, err := unhealthyAction.Execute(ctx, core.NewEnvelope("test"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "is unavailable due to startup failure")
		assert.Contains(t, err.Error(), "connect error")
		assert.Empty(t, res.Payload)
	})

	t.Run("Error Server FailOnStartup False No Tools", func(t *testing.T) {
		mockFactory := new(MockFactory)
		cfg := config.MCPServerConfig{
			Name:          "error-server-empty",
			FailOnStartup: false,
			// No tools defined
		}

		mockFactory.On("CreateClient", ctx, cfg).Return(nil, fmt.Errorf("connect error"))

		loader := NewLoader(cfg).WithFactory(mockFactory)
		actions, err := loader.Load(ctx)

		assert.NoError(t, err)
		assert.Empty(t, actions)
	})
}
