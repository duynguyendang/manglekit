package mcp

import (
	"context"
	"fmt"

	adapterai "github.com/duynguyendang/manglekit/adapters/ai"
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
	logger  core.Logger
}

// NewLoader creates a new Loader for the given configuration.
func NewLoader(cfg config.MCPServerConfig) *Loader {
	return &Loader{
		config:  cfg,
		factory: &DefaultFactory{},
		logger:  core.NopLogger{},
	}
}

// WithFactory allows overriding the default ClientFactory (useful for testing).
func (l *Loader) WithFactory(f ClientFactory) *Loader {
	l.factory = f
	return l
}

// WithLogger allows injecting a logger.
func (l *Loader) WithLogger(logger core.Logger) *Loader {
	l.logger = logger
	return l
}

// Load connects to the MCP server and returns the discovered actions.
// It implements the Driver Resilience (Health Check) pattern.
// If FailOnStartup is true, it returns connection errors.
// If FailOnStartup is false (default), it logs a warning and returns "Unhealthy" actions
// for expected tools defined in the config.
func (l *Loader) Load(ctx context.Context) ([]core.Action, error) {
	g := adapterai.GetGenkit(ctx)

	client, err := l.factory.CreateClient(ctx, l.config)
	if err == nil {
		// Try to list tools
		var tools []ai.Tool
		tools, err = client.GetActiveTools(ctx, g)
		if err == nil {
			var actions []core.Action
			for _, tool := range tools {
				// Success case: Create healthy action
				action := NewAction(l.config.Name, tool, "", nil)
				actions = append(actions, action)
			}
			return actions, nil
		}
	}

	// Failure Case Handling

	if l.config.FailOnStartup {
		// Critical failure: Bubble up the error to stop initialization
		return nil, fmt.Errorf("failed to load MCP server %s: %w", l.config.Name, err)
	}

	// Soft failure: Log warning and continue (Graceful Degradation)
	l.logger.Warn("MCP Tool failed to connect, marking as unhealthy", "tool", l.config.Name, "err", err)

	// Create "Unhealthy" actions for expected tools
	var actions []core.Action
	for _, toolName := range l.config.Tools {
		// Create unhealthy action with initError
		action := NewAction(l.config.Name, nil, toolName, err)
		actions = append(actions, action)
	}

	// If no expected tools are defined, we register nothing but return success (nil error)
	// so the SDK continues.
	return actions, nil
}
