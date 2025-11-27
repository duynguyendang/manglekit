package sandwich

import (
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/v1/core"
	"github.com/duynguyendang/manglekit/v1/pipeline"
)

// ActionStage is responsible for executing the primary action (e.g., retrieval)
// based on the user's query.
type ActionStage struct {
	DefaultAction core.Action
	SubActions    map[string]core.Action
	Logger        core.Logger
	Meter         core.Meter
}

// Name returns the identifier for the stage.
func (s *ActionStage) Name() string {
	return "action"
}

// selectAction dynamically chooses the action to execute based on the `target_action`
// predicate in the pipeline's answer metadata.
func (s *ActionStage) selectAction(p *pipeline.PipelineContext) (core.Action, string) {
	if targetAction, ok := p.Answer.Meta["target_action"].(string); ok {
		if action, exists := s.SubActions[targetAction]; exists {
			s.Logger.Infof("routing to sub-action", "action_name", targetAction)
			return action, targetAction
		}
		s.Logger.Warnf("target_action specified but not found in SubActions", "action_name", targetAction)
	}

	s.Logger.Infof("executing default action")
	return s.DefaultAction, "default"
}

// Execute calls the configured action with the query from the PipelineContext.
// It populates the context with the results (e.g., retrieved documents) and records latency.
func (s *ActionStage) Execute(p *pipeline.PipelineContext) error {
	action, actionName := s.selectAction(p)
	if action == nil {
		s.Logger.Warnf("action is nil, skipping action stage")
		return core.ErrNoEvidence
	}

	p.Answer.Meta["executed_action"] = actionName
	start := time.Now()
	s.Logger.Debugf("calling action", "action_name", actionName, "filters", p.Query.Meta["filters"], "expansions", p.Query.Meta["expansion_terms"])

	// Pass core.Query as input to the generic action.
	res, err := action.Execute(p.Ctx, p.Query)
	if err != nil {
		s.Logger.Errorf("action execution failed", "action_name", actionName, "error", err)
		return fmt.Errorf("action execution failed for '%s': %w", actionName, err)
	}

	p.RetrieveMS = float64(time.Since(start).Milliseconds())
	if s.Meter != nil {
		s.Meter.Record("manglekit.action_ms", p.RetrieveMS)
	}

	// Always store the result for inspection.
	p.Answer.Meta["action_result"] = res

	// Handle RetrieveResult specifically for backward compatibility (RAG support)
	if retrRes, ok := res.(core.RetrieveResult); ok {
		if len(retrRes.Docs) == 0 {
			s.Logger.Warnf("action returned no documents")
			return core.ErrNoEvidence
		}
		p.OriginalDocs = retrRes.Docs
		s.Logger.Infof("action returned documents", "count", len(p.OriginalDocs))
	} else {
		s.Logger.Infof("action executed successfully with generic result")
	}

	return nil
}
