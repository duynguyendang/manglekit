package main

import (
	"context"
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/adapters/vector"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/logger"
)

// =============================================================================
// Mock Implementations (for demonstration purposes)
// =============================================================================

// MockTextGenerator implements ai.TextGenerator with fixed responses.
type MockTextGenerator struct {
	logger core.Logger
}

func (m *MockTextGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	m.logger.Info("MockLLM received prompt", "prompt_length", len(prompt))
	return fmt.Sprintf("Final Answer: Based on the provided context, the answer is 42."), nil
}

// MockDocumentRetriever implements vector.DocumentRetriever with fixed documents.
type MockDocumentRetriever struct {
	logger core.Logger
}

func (m *MockDocumentRetriever) Retrieve(ctx context.Context, query string) ([]vector.Document, error) {
	m.logger.Info("MockRetriever received query", "query", query)
	return []vector.Document{
		{
			Content: "The meaning of life, the universe, and everything is 42.",
			Source:  "hitchhikers_guide.txt",
		},
		{
			Content: "Douglas Adams wrote The Hitchhiker's Guide to the Galaxy.",
			Source:  "authors.txt",
		},
	}, nil
}

// =============================================================================
// RAG Flow Demonstration
// =============================================================================

func main() {
	ctx := context.Background()

	fmt.Println("=== Manglekit RAG Flow Demo ===")
	fmt.Println()

	// ---------------------------------------------------------------------------
	// 1. Initialize Logger and Manglekit Client
	// ---------------------------------------------------------------------------
	log := logger.NewStdLogger()

	client, err := manglekit.NewClient(ctx, "", manglekit.WithLogger(log))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize manglekit client: %v\n", err)
		os.Exit(1)
	}
	log.Info("Manglekit client initialized")

	// ---------------------------------------------------------------------------
	// 2. Create Mock Backends (Simulating Genkit/Vector Store)
	// ---------------------------------------------------------------------------
	mockRetriever := &MockDocumentRetriever{logger: log}
	mockGenerator := &MockTextGenerator{logger: log}

	// ---------------------------------------------------------------------------
	// 3. Wrap Backends into Universal Actions
	// ---------------------------------------------------------------------------
	retrieverAction := vector.NewRetrieverAction("rag-retriever", mockRetriever)
	llmAction := ai.NewLLMAction("rag-llm", mockGenerator)

	log.Info("Actions created", "actions", "RetrieverAction, LLMAction")

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
	queryEnvelope := manglekit.NewEnvelope(userQuery)
	log.Info("User Query", "query", userQuery)
	fmt.Println()

	// Step 5b: Execute Retrieval (Guarded)
	fmt.Println("Step 1: Retrieval Phase")
	retrievedEnvelope, err := safeRetriever.Execute(ctx, queryEnvelope)
	if err != nil {
		log.Error("retrieval failed", "error", err)
		os.Exit(1)
	}
	log.Info("Retrieved documents", "doc_count", retrievedEnvelope.GetMeta("doc_count"))

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
	fmt.Println("Step 2: Generation Phase")
	prompt := fmt.Sprintf("Context:\n%s\nQuestion: %s\nAnswer:", formattedContext, userQuery)
	llmInputEnvelope := manglekit.NewEnvelope(prompt)

	// Step 5d: Execute LLM (Guarded)
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

	log.Info("LLM generation complete", "output_length", len(generatedText))
	fmt.Println()
	fmt.Println("Step 3: Final Answer")
	fmt.Printf("Question: %s\n", userQuery)
	fmt.Printf("Answer: %s\n", generatedText)
	fmt.Println()
	fmt.Println("=== RAG Flow Completed Successfully ===")
}
