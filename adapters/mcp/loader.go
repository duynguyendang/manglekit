package mcp

import (
	"context"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/mcp"
)

// Client defines the interface for an MCP client wrapper.
// This allows us to mock the client in tests.
type Client interface {
	GetActiveTools(ctx context.Context, g *genkit.Genkit) ([]ai.Tool, error)
}

// ClientFactory is an interface for creating MCP clients.
type ClientFactory interface {
	CreateClient(ctx context.Context, cfg config.MCPServerConfig) (Client, error)
}

// DefaultFactory implements ClientFactory using the real Genkit MCP plugin.
type DefaultFactory struct{}

// CreateClient creates a new GenkitMCPClient.
func (f *DefaultFactory) CreateClient(ctx context.Context, cfg config.MCPServerConfig) (Client, error) {
	opts := mcp.MCPClientOptions{
		Name: cfg.Name,
	}

	switch cfg.Transport {
	case "stdio":
		opts.Stdio = &mcp.StdioConfig{
			Command: cfg.Command,
			Args:    cfg.Args,
			Env:     cfg.Env,
		}
	case "sse":
		// Use Command as BaseURL for SSE if provided
		opts.SSE = &mcp.SSEConfig{
			BaseURL: cfg.Command,
		}
	}

	return mcp.NewGenkitMCPClient(opts)
}

// Load discovers and creates actions from configured MCP servers.
// It initializes a Genkit instance, connects to servers, and wraps their tools.
func Load(ctx context.Context, configs []config.MCPServerConfig, logger core.Logger) ([]core.Action, error) {
	return LoadWithFactory(ctx, configs, &DefaultFactory{}, logger)
}

// LoadWithFactory allows injection of a custom ClientFactory for testing.
func LoadWithFactory(ctx context.Context, configs []config.MCPServerConfig, factory ClientFactory, logger core.Logger) ([]core.Action, error) {
	var actions []core.Action

	// Ensure logger is not nil
	if logger == nil {
		logger = core.NopLogger{}
	}

	// Initialize Genkit context
	g := genkit.Init(ctx)

	for _, cfg := range configs {
		client, err := factory.CreateClient(ctx, cfg)
		if err != nil {
			// Log error but continue loading other servers
			logger.Error("Error connecting to MCP server", "server", cfg.Name, "error", err)
			continue
		}

		tools, err := client.GetActiveTools(ctx, g)
		if err != nil {
			logger.Error("Error listing tools for MCP server", "server", cfg.Name, "error", err)
			continue
		}

		for _, tool := range tools {
			action := NewAction(cfg.Name, tool)
			actions = append(actions, action)
			// Log discovery
			logger.Info("Discovered MCP Tool", "name", action.Metadata().Name)
		}
	}

	return actions, nil
}
