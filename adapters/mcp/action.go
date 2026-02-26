package mcp

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit-wip/core"
	"github.com/firebase/genkit/go/ai"
)

// MCPAction wraps a Genkit AI Tool discovered from an MCP server.
type MCPAction struct {
	tool       ai.Tool
	serverName string
	name       string
	initError  error
}

// NewAction creates a new MCPAction.
// If tool is nil (e.g. startup failure), name must be provided.
func NewAction(serverName string, tool ai.Tool, name string, initError error) *MCPAction {
	actionName := name
	if tool != nil {
		// Default naming strategy: mcp_<server>_<tool>
		actionName = tool.Name()
	}

	return &MCPAction{
		tool:       tool,
		serverName: serverName,
		name:       actionName,
		initError:  initError,
	}
}

// Execute invokes the underlying MCP tool via Genkit.
func (a *MCPAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	// Check for initialization error (Health Check Pattern)
	if a.initError != nil {
		return core.Envelope{}, fmt.Errorf("tool '%s' is unavailable due to startup failure: %w", a.name, a.initError)
	}

	// Safety check
	if a.tool == nil {
		return core.Envelope{}, fmt.Errorf("tool '%s' is invalid: nil tool implementation", a.name)
	}

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
		Name: fmt.Sprintf("mcp_%s_%s", a.serverName, a.name),
		Type: "mcp_tool",
	}
}
