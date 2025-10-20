package declarative

import (
	"context"
	"errors"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/internal/statehelper"
	"github.com/google/uuid"
)

const (
	// Standard keys for the execution context map.
	contextKeyQuery        = "query"
	contextKeyDocs         = "docs"
	contextKeyAnswer       = "answer"
	contextKeyMeta         = "meta"
	contextKeyDenialFlag   = "denied"
	contextKeyDenialReason = "denial_reason"
)

// DeclarativeOrchestrator implements the `core.Orchestrator` interface with a
// dynamic, rule-driven execution model. Unlike the linear `Sandwich` orchestrator,
// this implementation's workflow is not hardcoded in Go. Instead, it is defined
// declaratively as Mangle Datalog facts.
//
// The orchestrator queries a `FlowController` at runtime to determine which
// "tools" (configured components like retrievers or LLMs) to run and in what
// order. This separates the pipeline's execution logic (the "how" and "when")
// from the configuration of its components (the "what with"), allowing for
// highly flexible and dynamically configurable workflows without recompiling the code.
//
// A typical flow is defined with facts like:
//
//	flow_stage("my_flow", "1", "retrieval_stage").
//	stage_tool("retrieval_stage", "my_hybrid_retriever").
type DeclarativeOrchestrator struct {
	// flowController is the Mangle engine instance used to query for the workflow
	// definition and to evaluate pre/post rules.
	flowController core.FlowController
	// executionSteps is the configured sequence of tools to execute for this pipeline.
	executionSteps []core.Tool
	// stateProvider is the component responsible for persisting and retrieving session state.
	stateProvider core.StateProvider
	// obs holds the observability configuration (logger, meter, tracer).
	obs core.Observability
	// closers holds cleanup callbacks for external resources.
	closers []core.ResourceCloser
	// conversationManager is the helper for managing conversational state.
	conversationManager *statehelper.ConversationManager
}

// flowStage holds the information about a single stage in a declarative flow.
type flowStage struct {
	Name  string
	Order int
	Tool  string
}

// New creates a new DeclarativeOrchestrator.
func New(opts Options, fc core.FlowController, tools map[string]core.Tool, sp core.StateProvider, obs core.Observability, closers []core.ResourceCloser) (core.Orchestrator, error) {
	if obs.Logger == nil {
		obs.Logger = obslogger.NewStdLogger()
	}
	obs.Logger = obs.Logger.With("pipeline", "declarative")

	if fc == nil {
		err := fmt.Errorf("DeclarativeOrchestrator requires a non-nil FlowController")
		obs.Logger.Errorf(err.Error())
		return nil, err
	}
	if tools == nil {
		err := fmt.Errorf("DeclarativeOrchestrator requires a non-nil tools map")
		obs.Logger.Errorf(err.Error())
		return nil, err
	}
	if len(opts.Steps) == 0 {
		err := fmt.Errorf("DeclarativeOrchestrator requires at least one step")
		obs.Logger.Errorf(err.Error())
		return nil, err
	}

	executionSteps := make([]core.Tool, 0, len(opts.Steps))
	for _, step := range opts.Steps {
		tool, ok := tools[step.Name]
		if !ok {
			err := fmt.Errorf("tool '%s' not found in provided tools map", step.Name)
			obs.Logger.Errorf(err.Error())
			return nil, err
		}
		executionSteps = append(executionSteps, tool)
	}

	return &DeclarativeOrchestrator{
		flowController:      fc,
		executionSteps:      executionSteps,
		stateProvider:       sp,
		obs:                 obs,
		closers:             closers,
		conversationManager: statehelper.NewConversationManager(),
	}, nil
}

// Close releases any external resources held by the orchestrator.
func (o *DeclarativeOrchestrator) Close(ctx context.Context) error {
	var combined error
	for i := len(o.closers) - 1; i >= 0; i-- {
		if o.closers[i] == nil {
			continue
		}
		if err := o.closers[i](ctx); err != nil {
			combined = errors.Join(combined, err)
		}
	}
	return combined
}

// Execute runs the declarative workflow.
func (o *DeclarativeOrchestrator) Execute(ctx context.Context, sessionID string, q core.Query) (core.Answer, error) {
	requestID := uuid.NewString()
	logger := o.obs.Logger.With("request_id", requestID, "session_id", sessionID)
	logger.Infof("pipeline run started", "query", q.Text)

	// 1. RETRIEVE STATE: If a state provider and sessionID are available, get the history.
	history := o.conversationManager.LoadHistory(ctx, sessionID, o.stateProvider, logger)
	if q.Meta == nil {
		q.Meta = make(map[string]any)
	}
	q.Meta["history"] = history.Messages

	// 2. Create the execution context.
	execCtx := &core.ExecutionContext{
		Input: q.Text,
		Query: q,
		Answer: core.Answer{
			Meta: make(map[string]any),
		},
		Meta: make(map[string]any),
	}

	// 3. Loop through the configured execution steps.
	for _, tool := range o.executionSteps {
		if err := tool.Execute(ctx, execCtx); err != nil {
			// Immediately stop execution on error.
			logger.Errorf("tool execution failed", "error", err)
			return core.Answer{}, err
		}
	}

	// 4. UPDATE AND SAVE STATE: After a successful run, update and save the history.
	o.conversationManager.UpdateAndSaveHistory(ctx, sessionID, o.stateProvider, logger, history, q, execCtx.Answer)

	logger.Infof("pipeline run finished successfully")
	return execCtx.Answer, nil
}
