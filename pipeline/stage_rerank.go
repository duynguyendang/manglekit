package pipeline

import (
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

// RerankStage is responsible for re-scoring and re-ordering the documents
// retrieved in the previous stage to improve their relevance.
type RerankStage struct {
	Reranker          core.Reranker
	TopK              int
	FallbackThreshold float64
	Logger            core.Logger
	Meter             core.Meter
}

// Name returns the identifier for the stage.
func (s *RerankStage) Name() string {
	return "rerank"
}

// Execute calls the configured reranker with the documents from the context.
// It populates the context with the reranked documents and the best score.
// If the best score is below the fallback threshold, it returns `core.ErrNoEvidence`.
// If no reranker is configured, this stage is a no-op.
func (s *RerankStage) Execute(p *PipelineContext) error {
	// If no reranker is configured, pass original documents to the next stage.
	if s.Reranker == nil {
		p.FinalDocs = p.OriginalDocs
		return nil
	}

	start := time.Now()
	rerankedDocs, err := s.Reranker.Rerank(p.Ctx, core.RerankRequest{Query: p.Query.Text, Docs: p.OriginalDocs, TopK: s.TopK})
	p.RerankMS = float64(time.Since(start).Milliseconds())
	if s.Meter != nil {
		s.Meter.Record("manglekit.rerank_ms", p.RerankMS)
	}

	if err != nil {
		s.Logger.Errorf("rerank failed", "error", err)
		return fmt.Errorf("rerank failed: %w", err)
	}

	s.Logger.Infof("reranked documents", "count", len(rerankedDocs))
	p.RerankedDocs = rerankedDocs

	// Extract the best score for fallback logic.
	if len(rerankedDocs) > 0 {
		p.BestScore = rerankedDocs[0].Score
	}

	// Check against the fallback threshold.
	if s.FallbackThreshold > 0 && p.BestScore < s.FallbackThreshold {
		s.Logger.Warnf("fallback threshold not met", "best_score", p.BestScore, "threshold", s.FallbackThreshold)
		return core.ErrNoEvidence
	}

	// Populate FinalDocs for the next stage.
	p.FinalDocs = make([]core.Doc, len(rerankedDocs))
	for i, rd := range rerankedDocs {
		p.FinalDocs[i] = rd.Doc
	}

	return nil
}
