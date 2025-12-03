package main

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/sdk"
	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/adapters/vector"
	"github.com/duynguyendang/manglekit/core"
	genkit_ai "github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core/api"
)

// =============================================================================
// Mock Genkit Implementations (for demonstration purposes without API keys)
// =============================================================================

// MockGenkitModel implements a minimal genkit_ai.Model interface
// for testing the Genkit adapter without actual API calls.
type MockGenkitModel struct {
	logger core.Logger
}

func (m *MockGenkitModel) Name() string {
	return "mock-genkit-model"
}

func (m *MockGenkitModel) Generate(ctx context.Context, req *genkit_ai.ModelRequest, cb genkit_ai.ModelStreamCallback) (*genkit_ai.ModelResponse, error) {
	if req == nil || len(req.Messages) == 0 {
		return nil, fmt.Errorf("invalid request: no messages provided")
	}

	// Extract prompt from the first message
	msg := req.Messages[0]
	prompt := ""
	if len(msg.Content) > 0 && msg.Content[0].Text != "" {
		prompt = msg.Content[0].Text
	}

	m.logger.Info("MockGenkitModel.Generate called", "prompt_length", len(prompt))

	// Return a fixed response
	responseText := "Final Answer: Based on the provided context, the answer is 42."

	return &genkit_ai.ModelResponse{
		Message: &genkit_ai.Message{
			Role: "model",
			Content: []*genkit_ai.Part{
				genkit_ai.NewTextPart(responseText),
			},
		},
	}, nil
}

func (m *MockGenkitModel) Register(r api.Registry) {
	// No-op for mock
}

// MockGenkitRetriever implements a minimal genkit_ai.Retriever interface
// for testing the Genkit adapter without actual vector store connections.
type MockGenkitRetriever struct {
	logger core.Logger
}

func (m *MockGenkitRetriever) Name() string {
	return "mock-genkit-retriever"
}

func (m *MockGenkitRetriever) Retrieve(ctx context.Context, req *genkit_ai.RetrieverRequest) (*genkit_ai.RetrieverResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("invalid request: no retriever request provided")
	}

	m.logger.Info("MockGenkitRetriever.Retrieve called")

	// Return fixed documents
	docs := []*genkit_ai.Document{
		{
			Content: []*genkit_ai.Part{
				genkit_ai.NewTextPart("The meaning of life, the universe, and everything is 42."),
			},
			Metadata: map[string]any{
				"source": "hitchhikers_guide.txt",
			},
		},
		{
			Content: []*genkit_ai.Part{
				genkit_ai.NewTextPart("Douglas Adams wrote The Hitchhiker's Guide to the Galaxy."),
			},
			Metadata: map[string]any{
				"source": "authors.txt",
			},
		},
	}

	return &genkit_ai.RetrieverResponse{
		Documents: docs,
	}, nil
}

func (m *MockGenkitRetriever) Register(r api.Registry) {
	// No-op for mock
}

// =============================================================================
// RAG Flow Demonstration with Genkit Adapters
// =============================================================================

func main() {
	ctx := context.Background()

	fmt.Println("=== Manglekit RAG Flow Demo with Genkit Adapters ===")
	fmt.Println()

	// ---------------------------------------------------------------------------
	// 1. Initialize Logger and Manglekit Client
	// ---------------------------------------------------------------------------
	client, err := sdk.NewDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize manglekit client: %v\n", err)
		os.Exit(1)
	}
	log := client.Logger()
	log.Info("Manglekit client initialized")

	// ---------------------------------------------------------------------------
	// 2. Create Mock Genkit Implementations
	// ---------------------------------------------------------------------------
	mockGenkitModel := &MockGenkitModel{logger: log}
	mockGenkitRetriever := &MockGenkitRetriever{logger: log}

	// ---------------------------------------------------------------------------
	// 3. Wrap Genkit Backends into Universal Actions using Adapters
	// ---------------------------------------------------------------------------
	retrieverAction := vector.NewGenkitRetrieverAction("rag-retriever", mockGenkitRetriever, nil)
	llmAction, err := ai.NewGenkitAction("rag-llm", mockGenkitModel)
	if err != nil {
		log.Error("Failed to create Genkit LLM action", "error", err)
		os.Exit(1)
	}

	log.Info("Actions created with Genkit adapters", "actions", "GenkitRetrieverAction, GenkitLLMAction")

	// ---------------------------------------------------------------------------
	// 4. Protect Actions with Governance Guard
	// ---------------------------------------------------------------------------
	safeRetriever := client.Protect(retrieverAction)
	safeLLM := client.Protect(llmAction)

	log.Info("Actions protected with governance guard")
	fmt.Println()

	// ---------------------------------------------------------------------------
	// 5. Execute the RAG Flow
	// ---------------------------------------------------------------------------
	fmt.Println("--- Executing RAG Flow ---")
	fmt.Println()

	// Step 5a: Create the initial query envelope
	userQuery := "What is the meaning of life?"
	queryEnvelope := sdk.NewEnvelope(userQuery)
	log.Info("User Query", "query", userQuery)
	fmt.Println()

	// Step 5b: Execute Retrieval (Guarded) - now via Genkit adapter
	fmt.Println("Step 1: Retrieval Phase (via Genkit)")
	retrievedEnvelope, err := safeRetriever.Execute(ctx, queryEnvelope)
	if err != nil {
		log.Error("retrieval failed", "error", err)
		os.Exit(1)
	}
	log.Info("Retrieved documents via Genkit", "doc_count", retrievedEnvelope.GetMeta("doc_count"))

	// Format retrieved docs as context for LLM
	docsJSON, ok := retrievedEnvelope.Payload.(string)
	if !ok {
		log.Error("unexpected payload type", "type", fmt.Sprintf("%T", retrievedEnvelope.Payload))
		os.Exit(1)
	}

	formattedContext, err := vector.FormatDocsAsContext(docsJSON)
	if err != nil {
		log.Error("failed to format context", "error", err)
		os.Exit(1)
	}
	log.Debug("Formatted context", "context_length", len(formattedContext))
	fmt.Printf("Formatted Context:\n%s", formattedContext)
	fmt.Println()

	// Step 5c: Compose prompt with retrieved context
	fmt.Println("Step 2: Generation Phase (via Genkit)")
	prompt := fmt.Sprintf("Context:\n%s\nQuestion: %s\nAnswer:", formattedContext, userQuery)
	llmInputEnvelope := sdk.NewEnvelope(prompt)

	// Step 5d: Execute LLM (Guarded) - now via Genkit adapter
	generatedEnvelope, err := safeLLM.Execute(ctx, llmInputEnvelope)
	if err != nil {
		log.Error("LLM generation failed", "error", err)
		os.Exit(1)
	}

	// Extract the generated answer
	generatedText, ok := generatedEnvelope.Payload.(string)
	if !ok {
		log.Error("unexpected LLM output type", "type", fmt.Sprintf("%T", generatedEnvelope.Payload))
		os.Exit(1)
	}

	log.Info("LLM generation complete via Genkit", "output_length", len(generatedText))
	fmt.Println()
	fmt.Println("Step 3: Final Answer")
	fmt.Printf("Question: %s\n", userQuery)
	fmt.Printf("Answer: %s\n", generatedText)
	fmt.Println()
	fmt.Println("=== RAG Flow with Genkit Adapters Completed Successfully ===")
}
