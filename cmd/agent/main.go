package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/server"
	"gopkg.in/yaml.v3"
	"ndduy.dev/manglekit/internal/mangle"
	"ndduy.dev/manglekit/internal/orchestrator"
	"ndduy.dev/manglekit/internal/types"

	"github.com/joho/godotenv"
)

type EmbedderConfig struct {
	Model    string `yaml:"model"`
	Provider string `yaml:"provider"`
	APIKey   string `yaml:"apiKey"`
}

type AppConfig struct {
	Orchestrator orchestrator.Config `yaml:"orchestrator"`
	LLM          types.LLMConfig     `yaml:"llm"`
	Embedder     EmbedderConfig      `yaml:"embedder"`
	Mangle       mangle.Config       `yaml:"mangle"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file, using environment variables: %v", err)
	}
	ctx := context.Background()
	g := genkit.Init(ctx)

	builder, err := NewBuilder(ctx, g, "config/config.yaml")
	if err != nil {
		log.Fatalf("failed to create builder: %v", err)
	}

	orch, err := builder.Build()
	if err != nil {
		log.Fatalf("failed to build application: %v", err)
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
		fmt.Fprintf(os.Stderr, "could not parse front matter: %v, file content will be used as is", err)
		return nil, string(fileContent)
	}
	return metadata, string(parts[2])
}