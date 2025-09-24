// Package agent provides the core agent implementation for Manglekit.
// It implements a simplified agent that uses the Sandwich pattern for backward compatibility.
package agent

import (
	"context"
	"fmt"
	"strings"

	"ndduy.dev/manglekit/internal/llm"
	"ndduy.dev/manglekit/internal/types"
)

// DemoAgent provides a simplified agent implementation for backward compatibility.
// For new code, use the Orchestrator directly.
type DemoAgent struct {
	llm llm.LLM
}

// NewDemoAgent creates a new demo agent with the provided LLM client.
func NewDemoAgent(llmClient llm.LLM) *DemoAgent {
	return &DemoAgent{llm: llmClient}
}

// Answer processes a query using a simplified version of the Sandwich pattern.
// This method is kept for backward compatibility.
func (a *DemoAgent) Answer(user, query string) (string, error) {
	// Mangle-Pre (simplified): normalize + basic filters
	norm := strings.ToLower(query)
	filters := map[string]string{"visibility": "public_or_tenant"}

	// Retrieval (simplified)
	docs := a.retrieve(norm, filters)

	// Mangle-Post (simplified): drop disallowed docs
	vetted := []string{}
	for _, d := range docs {
		if a.allowDoc(user, d) {
			vetted = append(vetted, d.Text)
		}
	}

	// LLM synthesis
	return a.llm.Answer(context.Background(), query, vetted)
}

// doc represents a simple document structure for the demo agent.
type doc struct {
	ID         string
	Tenant     string
	Visibility string
	Title      string
	Text       string
}

// retrieve performs simple document retrieval for the demo agent.
func (a *DemoAgent) retrieve(q string, filters map[string]string) []doc {
	// Mock corpus for demonstration
	corpus := []doc{
		{
			ID:         "d1",
			Tenant:     "t42",
			Visibility: "tenant",
			Title:      "Fix PDF export crash",
			Text:       "Workaround for PDF export crash on Ubuntu 22.04: Update the PDF library to version 2.1.3 and restart the service.",
		},
		{
			ID:         "d2",
			Tenant:     "public",
			Visibility: "public",
			Title:      "General export guide",
			Text:       "How export works: The system supports multiple export formats including PDF, CSV, and JSON. Use the /export endpoint with the format parameter.",
		},
		{
			ID:         "d3",
			Tenant:     "t42",
			Visibility: "internal",
			Title:      "Debug export issues",
			Text:       "Internal debugging guide for export failures. Check logs in /var/log/app/ and verify database connections.",
		},
	}

	var out []doc
	for _, d := range corpus {
		// Simple text search
		searchText := strings.ToLower(d.Title + " " + d.Text)
		if strings.Contains(searchText, q) {
			out = append(out, d)
		}
	}

	return out
}

// allowDoc performs simple access control for the demo agent.
func (a *DemoAgent) allowDoc(user string, d doc) bool {
	// Simple access control logic
	switch d.Visibility {
	case "public":
		return true
	case "tenant":
		// In a real system, you'd check if user belongs to tenant
		return d.Tenant == "t42" // Simplified for demo
	case "internal":
		// In a real system, you'd check user permissions
		return user == "admin" // Simplified for demo
	default:
		return false
	}
}

// Gateway wraps an LLM client to implement the Gateway interface.
type Gateway struct {
	llm llm.LLM
}

// NewGateway creates a new Gateway wrapper around an LLM client.
func NewGateway(llmClient llm.LLM) types.Gateway {
	// If the LLM client already implements Gateway, return it directly
	if gateway, ok := llmClient.(types.Gateway); ok {
		return gateway
	}

	// Otherwise, wrap it
	return &Gateway{llm: llmClient}
}

// Generate implements the Gateway interface by wrapping the LLM interface.
func (g *Gateway) Generate(ctx context.Context, prompt string, chunks []*types.Chunk) (*types.Response, error) {
	// Convert chunks to simple strings for the legacy interface
	contextDocs := make([]string, len(chunks))
	citations := make([]string, 0, len(chunks))

	for i, chunk := range chunks {
		contextDocs[i] = chunk.Text
		if chunk.ID != "" {
			citations = append(citations, chunk.ID)
		}
	}

	// Extract question from prompt (simplified)
	// In a real implementation, you'd parse the prompt more carefully
	question := prompt
	if strings.Contains(prompt, "Question:") {
		parts := strings.Split(prompt, "Question:")
		if len(parts) > 1 {
			question = strings.TrimSpace(parts[len(parts)-1])
		}
	}

	// Call the legacy LLM interface
	answer, err := g.llm.Answer(ctx, question, contextDocs)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Build response
	response := &types.Response{
		Answer:    answer,
		Citations: citations,
		Metadata: map[string]interface{}{
			"chunks_used": len(chunks),
		},
	}

	return response, nil
}
