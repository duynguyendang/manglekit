package rag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/core"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/localvec"
)

const (
	collectionName = "manglekit-collection"
)

// Config holds the configuration for the RAG system.
type Config struct {
	Embedder struct {
		Model string `yaml:"model"`
	} `yaml:"embedder"`
	Retriever struct {
		Path string `yaml:"path"`
	} `yaml:"retriever"`
}

// RAG encapsulates the retriever and other components.
type RAG struct {
	retriever ai.Retriever
	generator *genkit.Genkit
}

// New creates and initializes a new RAG system.
func New(ctx context.Context, g *genkit.Genkit, cfg *Config) (*RAG, error) {
	embedder := googlegenai.GoogleAIEmbedder(g, cfg.Embedder.Model)
	if embedder == nil {
		return nil, fmt.Errorf("failed to create embedder")
	}

	if err := localvec.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize localvec: %w", err)
	}

	retOpts := &ai.RetrieverOptions{
		ConfigSchema: core.InferSchemaMap(localvec.RetrieverOptions{}),
		Label:        collectionName,
	}
	docStore, retriever, err := localvec.DefineRetriever(g, collectionName, localvec.Config{Embedder: embedder}, retOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to define retriever: %w", err)
	}

	docs, err := loadDocuments(cfg.Retriever.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load documents: %w", err)
	}

	if err := localvec.Index(ctx, docs, docStore); err != nil {
		return nil, fmt.Errorf("failed to index documents: %w", err)
	}

	return &RAG{
		retriever: retriever,
		generator: g,
	}, nil
}

// Retrieve finds relevant documents for a given query.
func (r *RAG) Retrieve(ctx context.Context, query string) ([]string, error) {
	dRequest := ai.DocumentFromText(query, nil)
	response, err := genkit.Retrieve(ctx, r.generator,
		ai.WithRetriever(r.retriever),
		ai.WithDocs(dRequest),
		ai.WithConfig(&localvec.RetrieverOptions{K: 2}))
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	var results []string
	for _, d := range response.Documents {
		results = append(results, d.Content[0].Text)
	}
	return results, nil
}

// loadDocuments reads all markdown files from a directory and returns them as Documents.
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