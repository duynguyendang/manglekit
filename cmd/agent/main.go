package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/server"
	"gopkg.in/yaml.v3"
	"ndduy.dev/manglekit/internal/orchestrator"
	"ndduy.dev/manglekit/internal/rag"
	"ndduy.dev/manglekit/internal/retrieval"
	"ndduy.dev/manglekit/internal/types"
)

type AppConfig struct {
	Orchestrator orchestrator.Config `yaml:"orchestrator"`
	LLM          types.LLMConfig     `yaml:"llm"`
	RAG          rag.Config          `yaml:"rag"`
}

func main() {
	ctx := context.Background()
	g := genkit.Init(ctx)

	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Manually wire the LLM config into the orchestrator config
	orchConfig := cfg.Orchestrator
	orchConfig.LLM = cfg.LLM

	rag, err := rag.New(ctx, g, &cfg.RAG)
	if err != nil {
		log.Fatalf("failed to create rag: %v", err)
	}

	retriever := retrieval.NewMock()

	orch, err := orchestrator.New(ctx, orchConfig, retriever, rag)
	if err != nil {
		log.Fatalf("failed to create orchestrator: %v", err)
	}

	answerFlow := genkit.DefineFlow(g, "answer", orch.RunFlow)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /"+answerFlow.Name(), genkit.Handler(answerFlow))
	log.Fatal(server.Start(ctx, "127.0.0.1:8082", mux))
}

func loadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Expand environment variables in the config file
	expandedData := []byte(os.ExpandEnv(string(data)))
	var cfg AppConfig
	err = yaml.Unmarshal(expandedData, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}