package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/localvec"
	"gopkg.in/yaml.v3"
	"ndduy.dev/manglekit/internal/logger"
	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/orchestrator"
	"ndduy.dev/manglekit/internal/reranker"
	"ndduy.dev/manglekit/internal/retrieval"
	"ndduy.dev/manglekit/internal/types"
)

func main() {
	ctx := context.Background()

	// 1. Initialize Genkit and a logger
	g := genkit.Init(ctx)
	appLogger, err := logger.New()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer appLogger.Sync()

	// 2. Configure and initialize the embedder
	embedder := googlegenai.GoogleAIEmbedder(g, "text-embedding-004")
	if embedder == nil {
		log.Fatal("Failed to create embedder")
	}

	// 3. Set up the vector store
	if err := localvec.Init(); err != nil {
		log.Fatalf("Failed to initialize localvec: %v", err)
	}

	// 4. Load documents from the data directory
	docs, err := loadDocuments("examples/simple/data")
	if err != nil {
		log.Fatalf("Failed to load documents: %v", err)
	}

	// 5. Create retrievers (BM25 for keywords, Dense for semantics)
	bm25Retriever, err := retrieval.NewBM25(ctx, "examples/simple/data")
	if err != nil {
		log.Fatalf("Failed to create BM25 retriever: %v", err)
	}
	denseRetriever, err := retrieval.NewDense(ctx, g, embedder, docs)
	if err != nil {
		log.Fatalf("Failed to create Dense retriever: %v", err)
	}
	hybridRetriever, err := retrieval.NewHybridRetriever(bm25Retriever, denseRetriever)
	if err != nil {
		log.Fatalf("Failed to create Hybrid retriever: %v", err)
	}

	// 6. Create the reranker
	docReranker, err := reranker.New(embedder)
	if err != nil {
		log.Fatalf("Failed to create reranker: %v", err)
	}

	// 7. Configure and create the orchestrator
	orchConfig := orchestrator.Config{
		Mangle: mangle.Config{
			RulesFile: "examples/simple/mangle",
		},
		LLM: types.LLMConfig{
			Provider: "googlegenai",
			Model:    "gemini-1.5-flash-latest",
		},
	}
	orch, err := orchestrator.New(ctx, g, orchConfig, hybridRetriever, docReranker, appLogger)
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	// 8. Define a query and run the flow
	query := &types.QueryInput{
		Query: "What is manglekit v1.0?",
	}
	fmt.Printf("Running query: '%s'\n", query.Query)
	result, err := orch.RunFlow(ctx, query)
	if err != nil {
		log.Fatalf("Flow failed: %v", err)
	}

	// 9. Print the result
	fmt.Println("\n---\nAnswer:")
	fmt.Println(result.Answer)
	fmt.Println("\nCitations:")
	for _, citation := range result.Citations {
		fmt.Printf("- %s\n", citation)
	}
	fmt.Println("\n---")
}

// loadDocuments scans a directory for markdown files and loads them into memory.
func loadDocuments(path string) ([]*ai.Document, error) {
	var documents []*ai.Document
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			metadata, contentStr := parseFrontMatter(content)
			if metadata == nil {
				metadata = make(map[string]any)
			}
			metadata["source"] = path // Add source for citation
			doc := ai.DocumentFromText(contentStr, metadata)
			documents = append(documents, doc)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return documents, nil
}

// parseFrontMatter extracts YAML front matter from a markdown file.
func parseFrontMatter(fileContent []byte) (map[string]any, string) {
	const separator = "---\n"
	if !bytes.HasPrefix(fileContent, []byte(separator)) {
		return nil, string(fileContent)
	}
	parts := bytes.SplitN(fileContent, []byte(separator), 3)
	if len(parts) < 3 {
		return nil, string(fileContent)
	}
	var metadata map[string]any
	if err := yaml.Unmarshal(parts[1], &metadata); err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse front matter: %v", err)
		return nil, string(fileContent)
	}
	return metadata, string(parts[2])
}