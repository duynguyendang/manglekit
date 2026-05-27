package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/logic"
	"github.com/duynguyendang/manglekit/internal/core/ports"
	"github.com/google/uuid"
)

// Orchestrator manages the OODA loop lifecycle for a given Signal.
type Orchestrator struct {
	perception ports.PerceptionPort
	proposer   ports.GenerativePort // Typically the multi-stream Proposer service
	compiler   ports.CompilerPort
	auditor    ports.AuditorPort
	genePool   ports.GenePoolPort
	storage    ports.GenomeStoragePort
}

func New(
	perception ports.PerceptionPort,
	proposer ports.GenerativePort,
	compiler ports.CompilerPort,
	auditor ports.AuditorPort,
	genePool ports.GenePoolPort,
	storage ports.GenomeStoragePort,
) *Orchestrator {
	return &Orchestrator{
		perception: perception,
		proposer:   proposer,
		compiler:   compiler,
		auditor:    auditor,
		genePool:   genePool,
		storage:    storage,
	}
}

// Execute loops through Observe, Orient, Decide, Verify until success or deadlock.
func (o *Orchestrator) Execute(ctx context.Context, signal domain.Signal) (*domain.DecisionOutput, error) {
	// Initialize Epoch Passport (CognitiveFrame)
	frame := &domain.CognitiveFrame{
		ID:        uuid.New(),
		Timestamp: time.Now().UTC(),
		Intent:    signal.Intent,
		TaskType:  domain.TaskTypeGeneration, // Default
		Status:    domain.VerifyStatusPending,
		EAST: domain.EASTState{
			LogicSuccess:       1.0,
			EntropyCoefficient: 1.0, // Assuming baseline novelty
		},
	}

	frame.EAST.SteeringMagnitude = logic.CalculateEAST(frame.EAST.LogicSuccess, frame.EAST.EntropyCoefficient)

	// 1. OBSERVE (Signal -> Atoms)
	payload, err := o.perception.Normalize(ctx, signal)
	if err != nil {
		return nil, fmt.Errorf("observe phase failed: %w", err)
	}

	// Convert iterator to slice for context
	for atom := range payload {
		frame.Context = append(frame.Context, atom)
	}

	// 2. ORIENT (Load Genes & Context)
	for gene := range o.genePool.ActiveGenes(ctx, frame.Intent) {
		frame.ActiveGenes = append(frame.ActiveGenes, *gene)
		if gene.Tier == domain.Tier0Kernel {
			// Extract FP32 axioms from Tier 0 rules for Attention Sink
			// Mocking atom extraction for now
			frame.AttentionSink = append(frame.AttentionSink, domain.Atom{Predicate: "kernel_rule", Subject: gene.Name})
		}
	}

	// 3. DECIDE & VERIFY (Teacher-Student Loop)
	maxRefinements := 3
	for pass := 1; pass <= maxRefinements; pass++ {
		// DECIDE Phase
		var refinement *logic.RefinementContext
		if frame.Status == domain.VerifyStatusFailed {
			refinement = &logic.RefinementContext{
				AuditResult:   frame.Proof,
				PreviousDraft: frame.Draft,
			}
		}

		prompt, err := o.compiler.Compile(ctx, frame.Intent,
			logic.WithTaskType(frame.TaskType),
			logic.WithContext(frame.Context),
			logic.WithAxioms(frame.AttentionSink),
			logic.WithGenes(frame.ActiveGenes),
			logic.WithPass(pass),
			logic.WithSteering(frame.EAST.SteeringMagnitude, logic.ShouldInjectParadox(frame.EAST.SteeringMagnitude)),
			logic.WithRefinement(refinement),
			logic.WithHistory(frame.SessionHistory),
		)
		if err != nil {
			return nil, fmt.Errorf("decide phase (compile) failed: %w", err)
		}

		plan, err := o.proposer.Generate(ctx, frame.Intent, prompt, frame.Context, frame.ActiveGenes)
		if err != nil {
			return nil, fmt.Errorf("decide phase (generate) failed: %w", err)
		}

		frame.Draft = plan
		frame.OutputType = domain.OutputTypePlan

		// VERIFY Phase (Shadow Audit)
		auditResult, err := o.auditor.Verify(ctx, frame)
		if err != nil {
			// Halt immediately on Auditor system error (e.g. missing Tier 0)
			return nil, err
		}

		frame.Proof = auditResult
		frame.SessionHistory = append(frame.SessionHistory, *auditResult)

		if auditResult.Pass {
			// ACT Phase Preparation
			frame.Status = domain.VerifyStatusPassed

			// Save Trace
			trace := o.auditor.GenerateTrace(frame)
			_ = o.storage.PersistTrace(ctx, frame, trace)

			return o.finalizeOutput(plan), nil
		}

		// Teacher-Student Loop Refinement update
		// Recalculate LogicSuccess based on EntropyDelta feedback
		frame.EAST.LogicSuccess = frame.EAST.LogicSuccess - 0.2 // simplistic penalty
		frame.EAST.SteeringMagnitude = logic.CalculateEAST(frame.EAST.LogicSuccess, frame.EAST.EntropyCoefficient)
		frame.Status = domain.VerifyStatusFailed
	}

	return nil, fmt.Errorf("DEADLOCK: failed to produce logic-compliant plan after %d refinements", maxRefinements)
}

func (o *Orchestrator) finalizeOutput(plan *ports.Plan) *domain.DecisionOutput {
	// Convert LLM Plan to final structured execution output
	if plan == nil || len(plan.Steps) == 0 {
		return &domain.DecisionOutput{Action: "NO_OP"}
	}

	// Assuming first step is the desired action for now
	return &domain.DecisionOutput{
		Action: "EXECUTE_PLAN",
		Params: map[string]any{"steps": plan.Steps},
	}
}
