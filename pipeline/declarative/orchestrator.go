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

// DeclarativeOrchestrator implements the `core.Orchestrator` interface. Its
// workflow is defined as a statically configured sequence of steps, provided
// via `declarative.Options`. This replaces the prior rule-driven, Datalog-based
// execution model with a simpler, more direct approach.
//
// The orchestrator iterates through a list of `core.Tool` implementations,
// executing them in order and passing a shared `core.ExecutionContext` between them.
type DeclarativeOrchestrator struct {
	executionSteps      []core.Tool
	stateProvider       core.StateProvider // Optional state provider.
	obs                 core.Observability
	closers             []core.ResourceCloser
	conversationManager *statehelper.ConversationManager
}

// NewDeclarative is the factory function for creating a DeclarativeOrchestrator.
// It resolves the tool names from the configuration against the built components
// provided in the `deps` argument.
func NewDeclarative(ctx context.Context, deps core.Resolved, cfg Options) (core.Orchestrator, error) {
	logger := deps.Obs.Logger
	if logger == nil {
		logger = obslogger.NewStdLogger()
	}
	logger = logger.With("pipeline", "declarative")

	if len(cfg.Steps) == 0 {
		return nil, fmt.Errorf("declarative orchestrator requires at least one step")
	}

	executionSteps := make([]core.Tool, 0, len(cfg.Steps))
	for _, stepCfg := range cfg.Steps {
		tool, err := deps.GetToolByName(stepCfg.Name)
		if err != nil {
			logger.Errorf("failed to resolve tool", "name", stepCfg.Name, "error", err)
			return nil, err
		}
		logger.Debugf("resolved tool", "name", stepCfg.Name)
		executionSteps = append(executionSteps, tool)
	}

	// For now, we arbitrarily pick the first state provider, if any.
	// A more robust implementation might require an explicit state provider name.
	var sp core.StateProvider
	for _, provider := range deps.StateProviders {
		sp = provider
		break
	}

	return &DeclarativeOrchestrator{
		executionSteps:      executionSteps,
		stateProvider:       sp,
		obs:                 deps.Obs,
		closers:             deps.Closers,
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
