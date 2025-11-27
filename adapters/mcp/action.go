package mcp

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/firebase/genkit/go/ai"
)

// MCPAction wraps a Genkit AI Tool discovered from an MCP server.
type MCPAction struct {
	tool       ai.Tool
	serverName string
}

// NewAction creates a new MCPAction.
func NewAction(serverName string, tool ai.Tool) *MCPAction {
	return &MCPAction{
		tool:       tool,
		serverName: serverName,
	}
}

// Execute invokes the underlying MCP tool via Genkit.
func (a *MCPAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Invoke the tool using RunRaw, which handles input marshaling if needed.
	res, err := a.tool.RunRaw(ctx, input.Payload)
	if err != nil {
		return core.Envelope{}, fmt.Errorf("execution failed for mcp tool %s: %w", a.tool.Name(), err)
	}

	return core.NewEnvelope(res), nil
}

// Metadata returns the action metadata.
func (a *MCPAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: fmt.Sprintf("mcp_%s_%s", a.serverName, a.tool.Name()),
		Type: "mcp_tool",
	}
}
