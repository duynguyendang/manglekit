package core

import (
	"context"
	"fmt"
)

// RetrieverTool adapts a core.Retriever to the core.Tool interface.
type RetrieverTool struct {
	R Retriever
}

// Execute retrieves documents based on the input in the ExecutionContext.
func (t *RetrieverTool) Execute(ctx context.Context, execCtx *ExecutionContext) error {
	if t.R == nil {
		return fmt.Errorf("retriever is nil")
	}

	topK := 10 // Default value
	if val, ok := execCtx.CurrentStepParams["topK"]; ok {
		if intVal, ok := val.(int); ok {
			topK = intVal
		}
	}

	// The retriever operates on the raw query text.
	result, err := t.R.Retrieve(ctx, RetrieveRequest{Query: execCtx.Input, TopK: topK})
	if err != nil {
		return fmt.Errorf("retriever tool failed: %w", err)
	}
	execCtx.Documents = result.Docs
	return nil
}

// RerankerTool adapts a core.Reranker to the core.Tool interface.
type RerankerTool struct {
	Rr Reranker
}

// Execute reranks the documents currently in the ExecutionContext.
func (t *RerankerTool) Execute(ctx context.Context, execCtx *ExecutionContext) error {
	if t.Rr == nil {
		return fmt.Errorf("reranker is nil")
	}

	topK := 10 // Default value
	if val, ok := execCtx.CurrentStepParams["topK"]; ok {
		if intVal, ok := val.(int); ok {
			topK = intVal
		}
	}

	// The reranker needs the original query text and the documents from a previous step.
	rerankedDocs, err := t.Rr.Rerank(ctx, RerankRequest{Query: execCtx.Input, Docs: execCtx.Documents, TopK: topK})
	if err != nil {
		return fmt.Errorf("reranker tool failed: %w", err)
	}
	// Convert []ScoredDoc back to []Doc
	docs := make([]Doc, len(rerankedDocs))
	for i, sd := range rerankedDocs {
		docs[i] = sd.Doc
	}
	execCtx.Documents = docs
	return nil
}

// LLMTool adapts a core.LLMClient to the core.Tool interface.
type LLMTool struct {
	Llm LLMClient
}

// Execute generates an answer using the LLM based on the current ExecutionContext.
func (t *LLMTool) Execute(ctx context.Context, execCtx *ExecutionContext) error {
	if t.Llm == nil {
		return fmt.Errorf("llm client is nil")
	}
	// Convert documents to strings for the LLM context.
	contexts := make([]string, len(execCtx.Documents))
	for i, doc := range execCtx.Documents {
		contexts[i] = doc.Text
	}

	// The LLM generates an answer from the query and the retrieved/reranked documents.
	response, err := t.Llm.Complete(ctx, LLMRequest{
		Prompt:  execCtx.Input, // Simplified: using input as prompt
		Context: contexts,
		Data:    execCtx.Query.Meta,
	})
	if err != nil {
		return fmt.Errorf("llm tool failed: %w", err)
	}
	execCtx.Answer = Answer{
		Text: response.Text,
		Meta: map[string]any{"token_usage": response.Usage},
	}
	return nil
}
