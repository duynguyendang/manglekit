package pipeline

// Runner composes and executes a sequence of pipeline stages. It is responsible
// for iterating through the added stages, calling their Execute method, and
// handling error propagation.
type Runner struct {
	stages []Stage
}

// Add appends a new stage to the runner's execution sequence. Stages will be
// run in the order they are added.
func (r *Runner) Add(s Stage) {
	if s == nil {
		return // Do not add nil stages
	}
	r.stages = append(r.stages, s)
}

// Run executes the pipeline stages in the order they were added. Execution
// halts immediately if any stage returns an error. The error from the failing
// stage is propagated up and also stored in the PipelineContext.
func (r *Runner) Run(p *PipelineContext) error {
	for _, s := range r.stages {
		// If a prior stage set the error, stop.
		if p.Err != nil {
			return p.Err
		}

		if err := s.Execute(p); err != nil {
			p.Err = err
			return err
		}
	}
	return nil
}
