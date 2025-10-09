// The agent command provides a runnable example of a MangleKit server.
// It uses the Genkit server plugin to expose a MangleKit orchestrator as an
// HTTP endpoint.
//
// This example demonstrates how to:
//  1. Load environment variables from a .env file.
//  2. Programmatically configure and build a MangleKit orchestrator using the
//     fluent Builder API.
//  3. Inject complex dependencies like an `ai.Embedder` into components that need it.
//  4. Define a Genkit flow (`/answer`) that wraps the orchestrator's Run method.
//  5. Start an HTTP server to listen for requests to the flow.
//
// Provider implementations are included via blank imports to ensure their `init()`
// functions run, registering them with the MangleKit registry.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/server"
	"github.com/joho/godotenv"

	// Blank import to register all standard providers.
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file, using environment variables: %v", err)
	}
	ctx := context.Background()

	// In a real application, you would use a config file or flags to determine
	// which providers to use. For this example, we'll hardcode them.
	// We still need to initialize genkit for the Google embedder.
	g := genkit.Init(ctx, genkit.WithPlugins(&googlegenai.GoogleAI{APIKey: os.Getenv("GOOGLE_API_KEY")}))
	embedder := googlegenai.GoogleAIEmbedder(g, "text-embedding-004")

	// Use the builder to construct the orchestrator programmatically.
	// We first provide the pre-built embedder, and the builder will automatically
	// inject it into components that need it (like the reranker).
	orch, err := manglekit.NewBuilder().
		WithEmbedder(embedder).
		WithRetriever(&retrieve.BM25Options{Path: "mangle/knowledge_base", TopK: 10}).
		WithReranker(&rerank.CosineOptions{TopK: 5}).
		WithLLM(&llm.OpenAIOptions{Model: "gpt-4o-mini"}).
		WithTopK(8).
		WithMaxTokens(512).
		WithFallbackThreshold(0.5).
		Build()
	if err != nil {
		log.Fatalf("failed to build orchestrator: %v", err)
	}

	// Define and run the Genkit flow.
	answerFlow := genkit.DefineFlow(g, "answer", func(ctx context.Context, query core.Query) (core.Answer, error) {
		return orch.Run(ctx, query)
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /"+answerFlow.Name(), genkit.Handler(answerFlow))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	fmt.Printf("Server listening on 127.0.0.1:%s\n", port)
	log.Fatal(server.Start(ctx, "127.0.0.1:"+port, mux))
}
