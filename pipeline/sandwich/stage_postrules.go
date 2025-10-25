package sandwich

import (
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/pipeline"
)

// PostRulesStage evaluates the set of rules configured to run at the "post" stage.
// It is responsible for final filtering of the answer and citations for compliance,
// safety, or other policy enforcement after the LLM has generated a response.
type PostRulesStage struct {
	RuleSet core.RuleSet
	Logger  core.Logger
	Meter   core.Meter
}

// Name returns the identifier for the stage.
func (s *PostRulesStage) Name() string {
	return "postrules"
}

// Execute runs the post-generation rule evaluation.
// It reads the query and the generated answer from the context, evaluates the rules,
// and then applies any mutations back to the context's answer object. If the rules
// deny the request, it returns a `core.ErrDenied` error to halt the pipeline.
func (s *PostRulesStage) Execute(p *pipeline.PipelineContext) error {
	if s.RuleSet == nil {
		return nil
	}

	tPostRulesStart := time.Now()
	res, err := s.RuleSet.Evaluate(core.Post, p.Query, &p.Answer)
	if s.Meter != nil {
		s.Meter.Record("manglekit.rules_post_ms", float64(time.Since(tPostRulesStart).Milliseconds()))
	}
	if err != nil {
		s.Logger.Errorf("post-rules failed", "error", err)
		return fmt.Errorf("post-rules failed: %w", err)
	}

	if !res.Allowed {
		s.Logger.Warnf("request denied by post-rule", "reason", res.Reason)
		return fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
	}

	if res.Mutate != nil {
		s.Logger.Debugf("answer mutated by post-rule")
		res.Mutate(&p.Query, &p.Answer)
	}
	return nil
}
