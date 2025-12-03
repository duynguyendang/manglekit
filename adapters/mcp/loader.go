package mcp

import (
	"context"
	"fmt"

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

// Loader handles the initialization of an MCP server.
type Loader struct {
	config  config.MCPServerConfig
	factory ClientFactory
}

// NewLoader creates a new Loader for the given configuration.
func NewLoader(cfg config.MCPServerConfig) *Loader {
	return &Loader{
		config:  cfg,
		factory: &DefaultFactory{},
	}
}

// WithFactory allows overriding the default ClientFactory (useful for testing).
func (l *Loader) WithFactory(f ClientFactory) *Loader {
	l.factory = f
	return l
}

// Load connects to the MCP server and returns the discovered actions.
// It returns an error if the connection or tool discovery fails.
func (l *Loader) Load(ctx context.Context) ([]core.Action, error) {
	// Initialize Genkit context
	g := genkit.Init(ctx)

	client, err := l.factory.CreateClient(ctx, l.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client for %s: %w", l.config.Name, err)
	}

	tools, err := client.GetActiveTools(ctx, g)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools for MCP server %s: %w", l.config.Name, err)
	}

	var actions []core.Action
	for _, tool := range tools {
		action := NewAction(l.config.Name, tool)
		actions = append(actions, action)
	}

	return actions, nil
}

// Load is a convenience function for backward compatibility or bulk loading.
// Deprecated: Use NewLoader(cfg).Load(ctx) instead for better error handling.
func Load(ctx context.Context, configs []config.MCPServerConfig, logger core.Logger) ([]core.Action, error) {
	var allActions []core.Action

	// Ensure logger is not nil
	if logger == nil {
		logger = core.NopLogger{}
	}

	for _, cfg := range configs {
		loader := NewLoader(cfg)
		// We use the default factory here. If tests needed injection, they should use NewLoader().WithFactory().
		// However, since LoadWithFactory was public, we should probably keep supporting it via a helper if needed,
		// but since we are refactoring, we can assume Load is legacy.

		actions, err := loader.Load(ctx)
		if err != nil {
			// Legacy behavior: Log error but continue
			logger.Error("Error connecting to MCP server", "server", cfg.Name, "error", err)
			continue
		}

		for _, action := range actions {
			allActions = append(allActions, action)
			logger.Info("Discovered MCP Tool", "name", action.Metadata().Name)
		}
	}

	return allActions, nil
}

// LoadWithFactory allows injection of a custom ClientFactory for testing.
// Deprecated: Use NewLoader(cfg).WithFactory(f).Load(ctx) instead.
func LoadWithFactory(ctx context.Context, configs []config.MCPServerConfig, factory ClientFactory, logger core.Logger) ([]core.Action, error) {
	var allActions []core.Action

	// Ensure logger is not nil
	if logger == nil {
		logger = core.NopLogger{}
	}

	for _, cfg := range configs {
		loader := NewLoader(cfg).WithFactory(factory)
		actions, err := loader.Load(ctx)
		if err != nil {
			// Legacy behavior: Log error but continue
			logger.Error("Error connecting to MCP server", "server", cfg.Name, "error", err)
			continue
		}

		for _, action := range actions {
			allActions = append(allActions, action)
			logger.Info("Discovered MCP Tool", "name", action.Metadata().Name)
		}
	}

	return allActions, nil
}
