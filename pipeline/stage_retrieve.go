package pipeline

import (
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/retrieve"
)

// RetrieveStage is responsible for fetching relevant documents from a retriever
// based on the user's query.
type RetrieveStage struct {
	Retriever retrieve.Retriever
	TopK      int
	Logger    core.Logger
	Meter     core.Meter
}

// Name returns the identifier for the stage.
func (s *RetrieveStage) Name() string {
	return "retrieve"
}

// Execute calls the configured retriever with the query from the PipelineContext.
// It populates the context with the retrieved documents and records the retrieval
// latency. If no documents are found, it returns `core.ErrNoEvidence`.
func (s *RetrieveStage) Execute(p *PipelineContext) error {
	if s.Retriever == nil {
		s.Logger.Warnf("retriever is nil, skipping retrieve stage")
		return core.ErrNoEvidence
	}

	start := time.Now()
	retrReq := retrieve.Request{Query: p.Query.Text, TopK: s.TopK, Meta: p.Query.Meta}
	s.Logger.Debugf("calling retriever", "filters", p.Query.Meta["filters"], "expansions", p.Query.Meta["expansion_terms"])

	retrRes, err := s.Retriever.Retrieve(p.Ctx, retrReq)
	if err != nil {
		s.Logger.Errorf("retrieve failed", "error", err)
		return fmt.Errorf("retrieve failed: %w", err)
	}

	p.RetrieveMS = float64(time.Since(start).Milliseconds())
	if s.Meter != nil {
		s.Meter.Record("manglekit.retrieve_ms", p.RetrieveMS)
	}

	if len(retrRes.Docs) == 0 {
		s.Logger.Warnf("retriever returned no documents")
		return core.ErrNoEvidence
	}

	p.OriginalDocs = retrRes.Docs
	s.Logger.Infof("retrieved documents", "count", len(p.OriginalDocs))
	return nil
}
