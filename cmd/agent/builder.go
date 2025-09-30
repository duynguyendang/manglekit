package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/firebase/genkit/go/plugins/googlegenai"
	"github.com/firebase/genkit/go/plugins/localvec"
	"go.uber.org/zap"
	"ndduy.dev/manglekit/internal/embedder"
	"ndduy.dev/manglekit/internal/logger"
	"ndduy.dev/manglekit/internal/orchestrator"
	"ndduy.dev/manglekit/internal/reranker"
	"ndduy.dev/manglekit/internal/retrieval"
	"ndduy.dev/manglekit/internal/types"
)

// Builder is responsible for wiring up the application.
type Builder struct {
	cfg *AppConfig
	g   *genkit.Genkit
	ctx context.Context
	log *zap.Logger
}

// NewBuilder creates a new application builder.
func NewBuilder(ctx context.Context, g *genkit.Genkit, configPath string) (*Builder, error) {
	log, err := logger.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &Builder{
		cfg: cfg,
		g:   g,
		ctx: ctx,
		log: log,
	}, nil
}

// Build constructs and returns the orchestrator.
func (b *Builder) Build() (types.Orchestrator, error) {
	// Manually wire the configs together
	orchConfig := b.cfg.Orchestrator
	orchConfig.LLM = b.cfg.LLM
	orchConfig.Mangle = b.cfg.Mangle

	embedder, err := b.createEmbedder()
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	if err := localvec.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize localvec: %w", err)
	}

	docs, err := b.loadDocuments(orchConfig.Retrieval.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to load documents: %w", err)
	}

	bm25Retriever, err := retrieval.NewBM25(b.ctx, orchConfig.Retrieval.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to create BM25 retriever: %w", err)
	}

	denseRetriever, err := retrieval.NewDense(b.ctx, b.g, embedder, docs)
	if err != nil {
		return nil, fmt.Errorf("failed to create Dense retriever: %w", err)
	}

	hybridRetriever, err := retrieval.NewHybridRetriever(bm25Retriever, denseRetriever)
	if err != nil {
		return nil, fmt.Errorf("failed to create Hybrid retriever: %w", err)
	}

	docReranker, err := reranker.New(embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to create reranker: %w", err)
	}

	orch, err := orchestrator.New(b.ctx, b.g, orchConfig, hybridRetriever, docReranker, b.log)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	return orch, nil
}

func (b *Builder) createEmbedder() (ai.Embedder, error) {
	var e ai.Embedder
	switch b.cfg.Embedder.Provider {
	case "google":
		plugin := &googlegenai.GoogleAI{APIKey: b.cfg.Embedder.APIKey}
		g := genkit.Init(b.ctx, genkit.WithPlugins(plugin))
		e = googlegenai.GoogleAIEmbedder(g, b.cfg.Embedder.Model)
	case "openai":
		e = embedder.NewOpenAIEmbedder(os.Getenv("OPENAI_API_KEY"), b.cfg.Embedder.Model)
	default:
		return nil, fmt.Errorf("unknown embedder provider: %s", b.cfg.Embedder.Provider)
	}

	if e == nil {
		return nil, fmt.Errorf("failed to create embedder with model %s", b.cfg.Embedder.Model)
	}
	return e, nil
}

func (b *Builder) loadDocuments(path string) ([]*ai.Document, error) {
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
			documents = append(documents, ai.DocumentFromText(contentStr, metadata))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return documents, nil
}