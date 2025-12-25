package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	function "github.com/duynguyendang/manglekit/adapters/func"
	"github.com/duynguyendang/manglekit/adapters/vector"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/google"
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

// Document represents a knowledge base item
type Document struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// QueryRequest defines the input payload for our action
type QueryRequest struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type MockLLM struct{}

func (m *MockLLM) Complete(ctx context.Context, prompt string) (string, error) {
	return "I read the context", nil
}
func (m *MockLLM) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	return &core.LLMResponse{
		Text:  "I read the context: [Mock Content]",
		Usage: map[string]int{"prompt": 10, "completion": 5},
	}, nil
}
func (m *MockLLM) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	ch := make(chan string)
	close(ch)
	return ch, nil
}

// CustomHybridMemory wraps the standard HybridMemory to inject "memory_hit" facts.
type CustomHybridMemory struct {
	*sdk.HybridMemory
	vectorStore core.VectorStore
}

// RecallWithFacts implements the optional interface to return metadata
func (m *CustomHybridMemory) RecallWithFacts(ctx context.Context, query string) (string, map[string]any, error) {
	// 1. Vector Search
	docIDs, err := m.vectorStore.Search(ctx, query, 3)
	if err != nil {
		return "", nil, err
	}

	var contextParts []string
	var hits []string

	for _, id := range docIDs {
		content, err := m.vectorStore.Get(ctx, id)
		if err == nil {
			contextParts = append(contextParts, fmt.Sprintf("[DocID:%s] %s", id, content))
			hits = append(hits, id)
		}
	}

	meta := make(map[string]any)
	if len(hits) > 0 {
		meta["memory_hit"] = hits // Will be converted to meta("memory_hit", "docID") for each
	}

	return strings.Join(contextParts, "\n\n"), meta, nil
}

func main() {
	ctx := context.Background()
	_ = godotenv.Load()

	// 1. Setup Components
	var embedder core.Embedder
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		if os.Getenv("GO_TEST") == "" {
			fmt.Println("Warning: GOOGLE_API_KEY not set, using Mock Embedder")
		}
		embedder = &MockEmbedder{}
	} else {
		g, err := google.NewEmbedder(ctx, apiKey, "text-embedding-004")
		if err != nil {
			log.Fatalf("Failed to init Google Embedder: %v", err)
		}
		embedder = g
	}

	// Vector Store
	vecStore := vector.NewSimpleStore(embedder) // Using SimpleStore as HNSW is internal/unavailable or same interface

	// Load Knowledge Base
	kbData, err := os.ReadFile("examples/hybrid_rag/data/knowledge.json")
	if err != nil {
		log.Fatalf("Failed to read knowledge.json: %v", err)
	}
	var docs []Document
	if err := json.Unmarshal(kbData, &docs); err != nil {
		log.Fatalf("Failed to parse knowledge.json: %v", err)
	}
	for _, doc := range docs {
		if err := vecStore.Upsert(ctx, doc.ID, doc.Content); err != nil {
			log.Fatalf("Failed to upsert doc %s: %v", doc.ID, err)
		}
	}

	// Hybrid Memory
	baseMem := sdk.NewHybridMemory(&core.NopStore{}, vecStore, embedder)
	customMem := &CustomHybridMemory{
		HybridMemory: baseMem,
		vectorStore:  vecStore,
	}

	// 2. Configure Client
	client, err := sdk.NewClient(ctx,
		sdk.WithMemory(customMem),
		sdk.WithFailMode(sdk.FailModeOpen), // Allow system errors, block alignment errors
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	client.SetLLM(&MockLLM{})

	// Load Policy
	policyData, err := os.ReadFile("examples/hybrid_rag/policy.dl")
	if err != nil {
		log.Fatalf("Failed to read policy.dl: %v", err)
	}
	if err := client.Engine().LoadPolicy(ctx, string(policyData)); err != nil {
		log.Fatalf("Failed to load policy: %v", err)
	}

	// Load Graph Facts
	graphData, err := os.ReadFile("examples/hybrid_rag/data/access_graph.nq")
	if err != nil {
		log.Fatalf("Failed to read access_graph.nq: %v", err)
	}
	var facts []string
	lines := strings.Split(string(graphData), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			s := strings.Trim(parts[0], "<>")
			p := strings.Trim(parts[1], "<>")
			o := strings.Trim(parts[2], "\"")
			fact := fmt.Sprintf("triple(\"%s\", \"%s\", \"%s\")", s, p, o)
			facts = append(facts, fact)
		}
	}
	if err := client.LoadFacts(facts); err != nil {
		log.Fatalf("Failed to load graph facts: %v", err)
	}

	// Register Action
	act := function.New("simulate_llm", func(ctx context.Context, req QueryRequest) (string, error) {
		return "Processed Query: " + req.Text, nil
	})
	safeAct := client.Supervise(act)
	client.RegisterAction("simulate_llm", safeAct)

	// 3. Run Scenarios
	runScenario(ctx, client, "Scenario A (Level 1 User)", "level_1", "What are the launch codes for Project X?", true)
	runScenario(ctx, client, "Scenario B (Level 3 User)", "level_3", "What are the launch codes for Project X?", false)
}

func runScenario(ctx context.Context, client *sdk.Client, name, userLevel, query string, expectBlock bool) {
	fmt.Printf("\n--- Running %s ---\n", name)

	req := QueryRequest{Type: "query", Text: query}

	_, err := client.ExecuteByName(ctx, "simulate_llm", req,
		sdk.WithMetadata("user_level", userLevel),
	)

	if err != nil {
		if expectBlock {
			if strings.Contains(err.Error(), "Access Denied") {
				fmt.Println("PASS: Request was blocked as expected.")
			} else {
				fmt.Printf("FAIL: Request blocked but with wrong reason: %v\n", err)
			}
		} else {
			fmt.Printf("FAIL: Request should have succeeded: %v\n", err)
		}
	} else {
		if expectBlock {
			fmt.Println("FAIL: Request should have been blocked.")
		} else {
			fmt.Println("PASS: Request succeeded as expected.")
		}
	}
}

// MockEmbedder for testing without API Key
type MockEmbedder struct{}

func (m *MockEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if strings.Contains(text, "launch") || strings.Contains(text, "Project X") {
		return []float32{0.9, 0.1}, nil
	}
	return []float32{0.1, 0.9}, nil
}
func (m *MockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	var res [][]float32
	for _, t := range texts {
		e, _ := m.Embed(ctx, t)
		res = append(res, e)
	}
	return res, nil
}
func (m *MockEmbedder) Dimension() int { return 2 }
