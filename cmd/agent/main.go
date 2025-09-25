package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/localvec"
	"github.com/firebase/genkit/go/plugins/server"
	"gopkg.in/yaml.v3"
	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/orchestrator"
	"ndduy.dev/manglekit/internal/reranker"
	"ndduy.dev/manglekit/internal/retrieval"
	"ndduy.dev/manglekit/internal/types"
)

type EmbedderConfig struct {
	Model string `yaml:"model"`
}

type AppConfig struct {
	Orchestrator orchestrator.Config `yaml:"orchestrator"`
	LLM          types.LLMConfig     `yaml:"llm"`
	Embedder     EmbedderConfig      `yaml:"embedder"`
	Mangle       mangle.Config       `yaml:"mangle"`
}

func main() {
	ctx := context.Background()
	g := genkit.Init(ctx)

	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Manually wire the configs together
	orchConfig := cfg.Orchestrator
	orchConfig.LLM = cfg.LLM
	orchConfig.Mangle = cfg.Mangle

	// Initialize components
	embedder := googlegenai.GoogleAIEmbedder(g, cfg.Embedder.Model)
	if embedder == nil {
		log.Fatalf("failed to create embedder with model %s", cfg.Embedder.Model)
	}

	if err := localvec.Init(); err != nil {
		log.Fatalf("failed to initialize localvec: %v", err)
	}

	docs, err := loadDocuments(orchConfig.Retrieval.Path)
	if err != nil {
		log.Fatalf("failed to load documents: %v", err)
	}

	bm25Retriever, err := retrieval.NewBM25(ctx, orchConfig.Retrieval.Path)
	if err != nil {
		log.Fatalf("failed to create BM25 retriever: %v", err)
	}

	denseRetriever, err := retrieval.NewDense(ctx, g, embedder, docs)
	if err != nil {
		log.Fatalf("failed to create Dense retriever: %v", err)
	}

	hybridRetriever, err := retrieval.NewHybridRetriever(bm25Retriever, denseRetriever)
	if err != nil {
		log.Fatalf("failed to create Hybrid retriever: %v", err)
	}

	reranker, err := reranker.New(embedder)
	if err != nil {
		log.Fatalf("failed to create reranker: %v", err)
	}

	orch, err := orchestrator.New(ctx, g, orchConfig, hybridRetriever, reranker)
	if err != nil {
		log.Fatalf("failed to create orchestrator: %v", err)
	}

	answerFlow := genkit.DefineFlow(g, "answer", orch.RunFlow)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /"+answerFlow.Name(), genkit.Handler(answerFlow))
	fmt.Println("Server listening on 127.0.0.1:8082")
	log.Fatal(server.Start(ctx, "127.0.0.1:8082", mux))
}

func loadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	expandedData := []byte(os.ExpandEnv(string(data)))
	var cfg AppConfig
	err = yaml.Unmarshal(expandedData, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

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
			documents = append(documents, ai.DocumentFromText(string(content), nil))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return documents, nil
}
