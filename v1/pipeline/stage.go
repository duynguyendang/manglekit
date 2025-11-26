package pipeline

// Stage is the core interface for a single, well-defined step within the
// orchestration pipeline (e.g., retrieve, rerank, llm). Each stage is
// responsible for a specific task, reading its inputs from the PipelineContext
// and writing its outputs back to it.
type Stage interface {
	// Name returns the identifier for the stage (e.g., "retrieve", "rerank").
	// This is useful for logging, metrics, and debugging.
	Name() string

	// Execute performs the primary logic of the stage. It receives a mutable
	// PipelineContext, which it reads from and writes to. If an error is
	// returned, the pipeline runner will halt execution.
	Execute(p *PipelineContext) error
}
