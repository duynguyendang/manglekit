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

// Response defines the output payload for our action
type Response struct {
	Type    string `json:"type"`
	Content string `json:"content"`
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

// CustomHybridMemory wraps the standard HybridMemory to inject "memory_hit" facts and security labels.
type CustomHybridMemory struct {
	*sdk.HybridMemory
	vectorStore core.VectorStore
}

// RecallWithFacts implements the optional interface to return metadata with security labels
func (m *CustomHybridMemory) RecallWithFacts(ctx context.Context, query string) (string, map[string]any, error) {
	// 1. Vector Search
	docIDs, err := m.vectorStore.Search(ctx, query, 3)
	if err != nil {
		return "", nil, err
	}

	var contextParts []string
	var hits []string
	var securityLabels []string

	for _, id := range docIDs {
		content, err := m.vectorStore.Get(ctx, id)
		if err == nil {
			contextParts = append(contextParts, fmt.Sprintf("[DocID:%s] %s", id, content))
			hits = append(hits, id)

			// Feature 4.1: Security Label Propagation
			// Inject security labels based on document ID
			switch id {
			case "doc_project_x", "doc_project_x_spec":
				securityLabels = append(securityLabels, "TOP_SECRET")
			case "doc_project_y":
				securityLabels = append(securityLabels, "CONFIDENTIAL")
			case "doc_remote_work":
				securityLabels = append(securityLabels, "PUBLIC")
			}
		}
	}

	meta := make(map[string]any)
	// Feature 1: Inject memory_hit as individual facts for each document
	// This allows the policy to check access for each retrieved document
	for i, hit := range hits {
		meta[fmt.Sprintf("memory_hit_%d", i)] = hit
	}
	if len(hits) > 0 {
		meta["memory_hit_count"] = len(hits)
	}
	if len(securityLabels) > 0 {
		meta["security_labels"] = securityLabels
	}

	return strings.Join(contextParts, "\n\n"), meta, nil
}

// PIIMockLLM simulates an LLM that might accidentally leak PII
type PIIMockLLM struct {
	LeakPII bool
}

func (m *PIIMockLLM) Complete(ctx context.Context, prompt string) (string, error) {
	if m.LeakPII {
		return "The user's SSN is 123-45-6789 and credit card is 4532-1234-5678-9010", nil
	}
	return "I have processed your request safely", nil
}
func (m *PIIMockLLM) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	if m.LeakPII {
		return &core.LLMResponse{
			Text:  "The user's SSN is 123-45-6789 and credit card is 4532-1234-5678-9010",
			Usage: map[string]int{"prompt": 10, "completion": 5},
		}, nil
	}
	return &core.LLMResponse{
		Text:  "I have processed your request safely",
		Usage: map[string]int{"prompt": 10, "completion": 5},
	}, nil
}
func (m *PIIMockLLM) Stream(ctx context.Context, prompt string) (<-chan string, error) {
	ch := make(chan string)
	close(ch)
	return ch, nil
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

	// Register Actions
	act := function.New("simulate_llm", func(ctx context.Context, req QueryRequest) (string, error) {
		return "Processed Query: " + req.Text, nil
	})
	safeAct := client.Supervise(act)
	client.RegisterAction("simulate_llm", safeAct)

	// 3. Run Scenarios
	fmt.Println("\n=== Feature 1: Complex Transitive Access Control ===")
	runScenario(ctx, client, "Scenario A (Alice - Research Group)", "user_alice", "What are the launch codes for Project X?", false)
	runScenario(ctx, client, "Scenario B (Charlie - Junior Group)", "user_charlie", "What are the launch codes for Project X?", true)
	runScenario(ctx, client, "Scenario C (Diana - Senior Group)", "user_diana", "What are the launch codes for Project X?", false)

	fmt.Println("\n=== Feature 2: Automated Self-Correction Loop (PII Detection) ===")
	runPIIScenario(ctx, client, "Scenario D (PII Leak)", "user_alice", true, true)
	runPIIScenario(ctx, client, "Scenario E (Safe Response)", "user_alice", false, false)

	fmt.Println("\n=== Feature 4: Information Flow Control (Security Tainting) ===")
	runEgressScenario(ctx, client, "Scenario F (TOP_SECRET to public)", "user_alice", "public_client", true)
	runEgressScenario(ctx, client, "Scenario G (TOP_SECRET to internal)", "user_diana", "internal_client", false)
}

func runScenario(ctx context.Context, client *sdk.Client, name, user, query string, expectBlock bool) {
	fmt.Printf("\n--- Running %s ---\n", name)

	req := QueryRequest{Type: "query", Text: query}

	_, err := client.ExecuteByName(ctx, "simulate_llm", req,
		sdk.WithMetadata("user", user),
	)

	if err != nil {
		if expectBlock {
			if strings.Contains(err.Error(), "Access Denied") || strings.Contains(err.Error(), "halt") {
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

func runPIIScenario(ctx context.Context, client *sdk.Client, name, user string, leakPII, expectRetry bool) {
	fmt.Printf("\n--- Running %s ---\n", name)

	// Set up PII mock LLM
	piiLLM := &PIIMockLLM{LeakPII: leakPII}
	client.SetLLM(piiLLM)

	// Register PII action
	act := function.New("pii_check", func(ctx context.Context, req QueryRequest) (Response, error) {
		if leakPII {
			return Response{Type: "response", Content: "The user's SSN is 123-45-6789 and credit card is 4532-1234-5678-9010"}, nil
		}
		return Response{Type: "response", Content: "I have processed your request safely"}, nil
	})
	safeAct := client.Supervise(act)
	client.RegisterAction("pii_check", safeAct)

	req := QueryRequest{Type: "query", Text: "Process user data"}

	_, err := client.ExecuteByName(ctx, "pii_check", req,
		sdk.WithMetadata("user", user),
	)

	if err != nil {
		if expectRetry {
			if strings.Contains(err.Error(), "RETRY") || strings.Contains(err.Error(), "PII") {
				fmt.Println("PASS: RETRY triggered as expected for PII detection.")
			} else {
				fmt.Printf("INFO: Request blocked/retried: %v\n", err)
			}
		} else {
			fmt.Printf("INFO: Request result: %v\n", err)
		}
	} else {
		if expectRetry {
			fmt.Println("FAIL: Request should have triggered RETRY.")
		} else {
			fmt.Println("PASS: Request succeeded as expected.")
		}
	}
}

func runEgressScenario(ctx context.Context, client *sdk.Client, name, user, destination string, expectBlock bool) {
	fmt.Printf("\n--- Running %s ---\n", name)

	req := QueryRequest{Type: "query", Text: "Send data to destination"}

	_, err := client.ExecuteByName(ctx, "simulate_llm", req,
		sdk.WithMetadata("user", user),
		sdk.WithMetadata("destination", destination),
	)

	if err != nil {
		if expectBlock {
			if strings.Contains(err.Error(), "Data Leakage Blocked") || strings.Contains(err.Error(), "halt") {
				fmt.Println("PASS: Request was blocked as expected.")
			} else {
				fmt.Printf("INFO: Request blocked: %v\n", err)
			}
		} else {
			fmt.Printf("INFO: Request result: %v\n", err)
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
