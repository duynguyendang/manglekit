// Package main for 09-genkit-tool shows how to integrate Manglekit as a tool
// within a larger Genkit-based application.
//
// In this example, the entire Manglekit orchestrator (built from its own YAML
// configuration) is wrapped into a single Genkit `Tool`. This tool,
// `manglekitKnowledgeBase`, can then be used within a Genkit `Flow`. This
// demonstrates the composability of Manglekit, allowing its powerful, rules-driven
// RAG capabilities to be seamlessly embedded as a component in other systems.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/joho/godotenv"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	_ "github.com/duynguyendang/manglekit/providers/all"
	"github.com/duynguyendang/manglekit/retrieve"
)

var mangleOrch core.Orchestrator

func main() {
	ctx := context.Background()

	if err := godotenv.Load(); err != nil {
		log.Printf("[Init] warning: unable to load .env file: %v", err)
	} else {
		log.Println("[Init] loaded environment variables from .env")
	}

	configPath, err := resolveConfigPath()
	if err != nil {
		log.Fatalf("[Init] failed to resolve config.yaml: %v", err)
	}

	builder, err := manglekit.NewBuilderFromYAML(configPath)
	if err != nil {
		log.Fatalf("[Init] failed to create builder from YAML: %v", err)
	}

	orch, err := builder.Build(ctx)
	if err != nil {
		log.Fatalf("[Init] failed to build orchestrator: %v", err)
	}
	defer func() {
		if cerr := orch.Close(ctx); cerr != nil {
			log.Printf("[Shutdown] error closing orchestrator: %v", cerr)
		}
	}()
	mangleOrch = orch

	if err := seedSampleDocuments(ctx, orch); err != nil {
		log.Fatalf("[Init] failed to seed sample documents: %v", err)
	}

	g := genkit.Init(ctx)

	knowledgeTool := genkit.DefineTool(g,
		"manglekitKnowledgeBase",
		"Queries the internal knowledge base to answer complex questions.",
		func(toolCtx *ai.ToolContext, query string) (string, error) {
			log.Printf("[Tool] manglekitKnowledgeBase invoked with query: %q", query)
			if mangleOrch == nil {
				return "", fmt.Errorf("mangle orchestrator is not initialized")
			}

			answer, err := mangleOrch.Run(toolCtx, core.Query{Text: query})
			if err != nil {
				return "", fmt.Errorf("mangle orchestrator failed: %w", err)
			}
			log.Printf("[Tool] manglekitKnowledgeBase returning answer: %q", answer.Text)
			return answer.Text, nil
		},
	)

	chatFlow := genkit.DefineFlow(g,
		"mainChatFlow",
		func(flowCtx context.Context, userQuestion string) (string, error) {
			log.Printf("[Flow] mainChatFlow received question: %q", userQuestion)

			// The original genkitshim provided a helper for this; we now use
			// the underlying genkit.Run helper directly.
			answer, err := genkit.Run(flowCtx, "call-manglekitKnowledgeBase", func() (string, error) {
				raw, err := knowledgeTool.RunRaw(flowCtx, userQuestion)
				if err != nil {
					return "", err
				}
				typed, ok := raw.(string)
				if !ok {
					return "", fmt.Errorf("genkit: tool %q returned %T, expected string", knowledgeTool.Name(), raw)
				}
				return typed, nil
			})
			if err != nil {
				return "", fmt.Errorf("tool invocation failed: %w", err)
			}

			log.Printf("[Flow] mainChatFlow returning answer: %q", answer)
			return answer, nil
		},
	)

	sampleQuestion := "What is Manglekit?"
	log.Printf("[Main] executing mainChatFlow with sample question: %q", sampleQuestion)

	response, err := chatFlow.Run(ctx, sampleQuestion)
	if err != nil {
		log.Fatalf("[Main] flow execution failed: %v", err)
	}

	fmt.Println("Final Response:", response)
}

func resolveConfigPath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("unable to determine current file path")
	}

	dir := filepath.Dir(currentFile)
	for {
		candidate := filepath.Join(dir, "config.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat %s: %w", candidate, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("config.yaml not found when searching upward from %s", currentFile)
}

func seedSampleDocuments(ctx context.Context, orch core.Orchestrator) error {
	retriever := orch.Retriever()
	if retriever == nil {
		return fmt.Errorf("orchestrator does not expose a retriever")
	}

	updatable, ok := retriever.(retrieve.Updatable)
	if !ok {
		log.Println("[Init] configured retriever does not support updates; skipping sample seeding")
		return nil
	}

	docs := []core.Doc{
		{
			ID:     "sample-doc-1",
			Source: "demo",
			URI:    "inmemory://sample-doc-1",
			Text:   "Manglekit combines Mangle rules with Genkit orchestration to build grounded AI workflows.",
			Meta: map[string]any{
				"category": "overview",
			},
		},
		{
			ID:     "sample-doc-2",
			Source: "demo",
			URI:    "inmemory://sample-doc-2",
			Text:   "The toolkit emphasizes the Sandwich pattern: Mangle-Pre, Retrieval, Mangle-Post, then LLM reasoning.",
			Meta: map[string]any{
				"category": "architecture",
			},
		},
	}

	log.Printf("[Init] seeding %d sample documents into the in-memory retriever", len(docs))
	if err := updatable.Upsert(ctx, docs); err != nil {
		return fmt.Errorf("failed to upsert sample documents: %w", err)
	}
	return nil
}
