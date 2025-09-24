// Package orchestrator implements the Genkit Orchestrator component.
// It coordinates the end-to-end flow: intake, pre-processing, retrieval, post-processing, and synthesis.

package agent

import (
	"context"
	"fmt"
	"strings"

	"ndduy.dev/manglekit/internal/types"
)

// OrchestratorImpl is the concrete implementation of the Orchestrator interface.
type OrchestratorImpl struct {
	mangle    types.Processor
	retriever types.Retriever
	llm       types.Gateway
}

// NewOrchestrator creates a new Orchestrator with dependencies.
func NewOrchestrator(mangle types.Processor, retriever types.Retriever, llm types.Gateway) types.Orchestrator {
	return &OrchestratorImpl{
		mangle:    mangle,
		retriever: retriever,
		llm:       llm,
	}
}

// RunFlow executes the core Sandwich pattern flow.
func (o *OrchestratorImpl) RunFlow(ctx context.Context, input *types.QueryInput) (*types.Response, error) {
	// Step 1: Mangle-Pre: Normalize, constrain, expand
	expanded, err := o.mangle.PreProcess(input)
	if err != nil {
		return nil, fmt.Errorf("pre-process failed: %w", err)
	}

	// Step 2: Retrieval: Hybrid search with filters
	chunks, err := o.retriever.Search(ctx, expanded, expanded.Filters)
	if err != nil {
		// Fallback: empty chunks for direct LLM
		chunks = []*types.Chunk{}
	}

	userCtx := &types.Context{UserContext: input.UserContext}

	// Step 3: Mangle-Post: Validate, redact, annotate
	postChunks, explanations := o.mangle.PostProcess(chunks, userCtx)

	// Step 4: LLM Synthesis: Generate response with context
	// Build Genkit-style prompt template with query, context, user info
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("User query: %s\n\n", input.Query)) // Use original query
	sb.WriteString("Relevant context (with relevance scores):\n")
	for _, chunk := range postChunks {
		sb.WriteString(fmt.Sprintf("- %s (score: %.2f)\n", chunk.Text, chunk.Score))
	}
	sb.WriteString("\nUser context: ")
	for k, v := range userCtx.UserContext {
		sb.WriteString(fmt.Sprintf("%s=%v; ", k, v))
	}
	sb.WriteString("\n\nAnswer the query using the provided context. Be concise and cite sources.\n")
	prompt := sb.String()
	// TODO: Mangle the prompt for final optimization (e.g., mangle.PostProcessPrompt(prompt))

	resp, err := o.llm.Generate(ctx, prompt, postChunks)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Step 5: Assemble response with citations and explanations
	if explanations != nil {
		resp.Explanations = *explanations
	}
	// TODO: Add metadata (latency, scores)

	return resp, nil
}
