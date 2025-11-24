package sandwich

import (
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/pipeline"
)

// ActionStage is responsible for executing the primary action (e.g., retrieval)
// based on the user's query.
type ActionStage struct {
	Action core.Action
	Logger core.Logger
	Meter  core.Meter
}

// Name returns the identifier for the stage.
func (s *ActionStage) Name() string {
	return "action"
}

// Execute calls the configured action with the query from the PipelineContext.
// It populates the context with the results (e.g., retrieved documents) and records latency.
func (s *ActionStage) Execute(p *pipeline.PipelineContext) error {
	if s.Action == nil {
		s.Logger.Warnf("action is nil, skipping action stage")
		return core.ErrNoEvidence
	}

	start := time.Now()
	s.Logger.Debugf("calling action", "filters", p.Query.Meta["filters"], "expansions", p.Query.Meta["expansion_terms"])

	// Pass core.Query as input to the generic action.
	res, err := s.Action.Execute(p.Ctx, p.Query)
	if err != nil {
		s.Logger.Errorf("action execution failed", "error", err)
		return fmt.Errorf("action execution failed: %w", err)
	}

	p.RetrieveMS = float64(time.Since(start).Milliseconds())
	if s.Meter != nil {
		s.Meter.Record("manglekit.action_ms", p.RetrieveMS)
	}

	// Handle RetrieveResult specifically for backward compatibility (RAG support)
	if retrRes, ok := res.(core.RetrieveResult); ok {
		if len(retrRes.Docs) == 0 {
			s.Logger.Warnf("action returned no documents")
			return core.ErrNoEvidence
		}
		p.OriginalDocs = retrRes.Docs
		s.Logger.Infof("action returned documents", "count", len(p.OriginalDocs))
		return nil
	}

	// If result is generic, store it in Answer.Meta or handle appropriately
	p.Answer.Meta["action_result"] = res
	s.Logger.Infof("action executed successfully")
	return nil
}
