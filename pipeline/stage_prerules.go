package pipeline

import (
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/core"
)

// PreRulesStage evaluates the set of rules configured to run at the "pre" stage.
// It is responsible for query validation, normalization, and mutation (e.g., adding
// metadata filters) before the retrieval stage is executed.
type PreRulesStage struct {
	RuleSet core.RuleSet
	Logger  core.Logger
	Meter   core.Meter
}

// Name returns the identifier for the stage.
func (s *PreRulesStage) Name() string {
	return "prerules"
}

// Execute runs the pre-retrieval rule evaluation.
// It reads the initial query from the context, evaluates the rules, and then
// applies any mutations back to the context's query object. If the rules
// deny the request, it returns a `core.ErrDenied` error to halt the pipeline.
func (s *PreRulesStage) Execute(p *PipelineContext) error {
	if s.RuleSet == nil {
		return nil
	}

	tPreRulesStart := time.Now()
	res, err := s.RuleSet.Evaluate(core.Pre, p.Query, &p.Answer)
	if s.Meter != nil {
		s.Meter.Record("manglekit.rules_pre_ms", float64(time.Since(tPreRulesStart).Milliseconds()))
	}
	if err != nil {
		s.Logger.Errorf("pre-rules failed", "error", err)
		return fmt.Errorf("pre-rules failed: %w", err)
	}

	if !res.Allowed {
		if res.Mutate != nil {
			res.Mutate(&p.Query, &p.Answer)
		}
		s.Logger.Warnf("request denied by pre-rule", "reason", res.Reason)
		return fmt.Errorf("%w: %s", core.ErrDenied, res.Reason)
	}

	if res.Mutate != nil {
		res.Mutate(&p.Query, &p.Answer)
		s.Logger.Debugf("query mutated by pre-rule")
	}

	return nil
}
