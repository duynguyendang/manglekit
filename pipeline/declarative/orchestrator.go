package declarative

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/duynguyendang/manglekit/core"
	obslogger "github.com/duynguyendang/manglekit/internal/logger"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
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
	// tools is a map of fully constructed component instances, where the key is
	// the tool's name as used in the Datalog facts (e.g., "my_hybrid_retriever").
	tools map[string]any
	// flowName is the name of the specific flow to execute, corresponding to the
	// first argument in the `flow_stage/3` Datalog facts.
	flowName string
	// stateProvider is the component responsible for persisting and retrieving session state.
	stateProvider core.StateProvider
	// obs holds the observability configuration (logger, meter, tracer).
	obs core.Observability
	// closers holds cleanup callbacks for external resources.
	closers []core.ResourceCloser
}

// flowStage holds the information about a single stage in a declarative flow.
type flowStage struct {
	Name  string
	Order int
	Tool  string
}

// New creates a new DeclarativeOrchestrator.
// This constructor is typically called by the main MangleKit builder.
//
// fc is the `core.FlowController` (typically a Mangle engine) that will be
// queried to determine the execution plan. It must not be nil.
// tools is a map of pre-configured and fully constructed tool instances (e.g.,
// retrievers, rerankers, LLMs). The map key is the name used to reference the
// tool in the Datalog rules. It must not be nil.
// flowName is the name of the specific flow to execute. It must not be empty.
//
// It returns a configured `core.Orchestrator` or an error if any of the
// required parameters are invalid.
func New(fc core.FlowController, tools map[string]any, flowName string, sp core.StateProvider, obs core.Observability, closers []core.ResourceCloser) (core.Orchestrator, error) {
	if obs.Logger == nil {
		obs.Logger = obslogger.NewStdLogger()
	}
	obs.Logger = obs.Logger.With("pipeline", "declarative", "flow", flowName)

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
	if flowName == "" {
		err := fmt.Errorf("DeclarativeOrchestrator requires a flow name")
		obs.Logger.Errorf(err.Error())
		return nil, err
	}

	return &DeclarativeOrchestrator{
		flowController: fc,
		tools:          tools,
		flowName:       flowName,
		stateProvider:  sp,
		obs:            obs,
		closers:        closers,
	}, nil
}

// Retriever satisfies the `core.Orchestrator` interface. It returns the first
// tool found in its configuration that implements the `retrieve.Retriever`
// interface. This provides a convenience method for runtime operations, such as
// updating documents in an updatable retriever.
//
// It returns the retriever instance as `any` (which can be type-asserted by the
// caller) or `nil` if no retriever tool is found in the orchestrator's configuration.
func (o *DeclarativeOrchestrator) Retriever() any {
	for _, tool := range o.tools {
		if r, ok := tool.(retrieve.Retriever); ok {
			return r
		}
	}
	return nil
}

// StateProvider returns the state provider component configured for the orchestrator.
func (o *DeclarativeOrchestrator) StateProvider() core.StateProvider {
	return o.stateProvider
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

// Run executes the declarative workflow. This method is the core interpreter for
// the rule-driven pipeline.
//
// The execution process involves several steps:
//  1. It queries the Mangle `FlowController` to fetch the stages of the specified
//     flow, defined by `flow_stage/3` and `stage_tool/2` facts.
//  2. It evaluates `pre-retrieval` rules, which can deny the request or provide
//     runtime information, such as a list of stages to skip dynamically.
//  3. It creates an execution context map to pass state (like the query, retrieved
//     documents, and the evolving answer) between stages.
//  4. It iterates through the flow's stages in their specified order, skipping any
//     stages flagged by the pre-rules.
//  5. For each stage, it dispatches to the configured tool, which modifies the
//     execution context.
//  6. After all stages are complete, it returns the final `core.Answer` from the
//     execution context.
//
// ctx is the context for the entire operation.
// q is the user's incoming query.
// It returns the final `core.Answer` or an error if the flow is undefined, a
// tool is missing, or a tool fails during execution.
func (o *DeclarativeOrchestrator) Execute(ctx context.Context, sessionID string, q core.Query) (core.Answer, error) {
	requestID := uuid.NewString()
	logger := o.obs.Logger.With("request_id", requestID, "session_id", sessionID)
	logger.Infof("pipeline run started", "query", q.Text)
	// 1. Query the static execution plan.
	stages, err := o.getFlowStages(ctx)
	if err != nil {
		err = fmt.Errorf("could not get flow stages for flow '%s': %w", o.flowName, err)
		logger.Errorf(err.Error())
		return core.Answer{}, err
	}
	if len(stages) == 0 {
		err = fmt.Errorf("no stages found for flow '%s'", o.flowName)
		logger.Errorf(err.Error())
		return core.Answer{}, err
	}

	// 2. Evaluate pre-rules to get runtime information like which stages to skip.
	preResult, err := o.flowController.Evaluate(core.Pre, q, nil)
	if err != nil {
		err = fmt.Errorf("pre-rules evaluation failed: %w", err)
		logger.Errorf(err.Error())
		return core.Answer{}, err
	}
	if !preResult.Allowed {
		return core.Answer{Meta: map[string]any{"denial_reason": preResult.Reason}}, core.ErrDenied
	}

	// 3. Create the execution context.
	execContext := map[string]any{
		contextKeyQuery:  q,
		contextKeyAnswer: core.Answer{Meta: map[string]any{}},
		contextKeyMeta:   map[string]any{},
	}

	// 3b. Apply mutations from pre-rules so filters/expansions are present in Query.Meta.
	if preResult.Mutate != nil {
		cq := execContext[contextKeyQuery].(core.Query)
		ca := execContext[contextKeyAnswer].(core.Answer)
		preResult.Mutate(&cq, &ca)
		execContext[contextKeyQuery] = cq
		execContext[contextKeyAnswer] = ca

		// Optional debug logs
		if cq.Meta != nil {
			if f, ok := cq.Meta["filters"]; ok {
				logger.Debugf("pre-rules filters applied", "filters", f)
			}
			if ex, ok := cq.Meta["expansion_terms"]; ok {
				logger.Debugf("pre-rules expansions applied", "expansions", ex)
			}
		}
	}

	// 4. Loop and dispatch.
	for _, stage := range stages {
		// a. Check for conditional skip from the pre-rule evaluation.
		if preResult.SkippedStages[stage.Name] {
			continue
		}

		// b. Look up the tool instance.
		tool, ok := o.tools[stage.Tool]
		if !ok {
			return core.Answer{}, fmt.Errorf("tool '%s' configured for stage '%s' not found in the tool map", stage.Tool, stage.Name)
		}

		// c. Execute the tool.
		if err := o.dispatchToTool(ctx, logger, stage, tool, execContext); err != nil {
			finalAnswer, _ := execContext[contextKeyAnswer].(core.Answer)
			if errors.Is(err, core.ErrDenied) {
				if reason, ok := execContext[contextKeyDenialReason].(string); ok && reason != "" {
					if finalAnswer.Meta == nil {
						finalAnswer.Meta = map[string]any{}
					}
					finalAnswer.Meta["denial_reason"] = reason
				}
				return finalAnswer, err
			}
			return core.Answer{}, fmt.Errorf("execution of tool '%s' for stage '%s' failed: %w", stage.Tool, stage.Name, err)
		}
	}

	// 5. Assemble the final answer.
	finalAnswer, ok := execContext[contextKeyAnswer].(core.Answer)
	if !ok {
		return core.Answer{}, fmt.Errorf("execution context ended without a valid answer object")
	}
	logger.Infof("pipeline run finished successfully")
	return finalAnswer, nil
}

// getFlowStages queries the Mangle engine for `flow_stage` and `stage_tool` facts
// and returns a single sorted list of stages.
func (o *DeclarativeOrchestrator) getFlowStages(ctx context.Context) ([]flowStage, error) {
	// Query for flow stages
	stageQuery := fmt.Sprintf(`flow_stage("%s", Order, StageName).`, o.flowName)
	stagesByName := make(map[string]flowStage)
	err := o.flowController.Query(ctx, stageQuery, func(sol map[string]any) error {
		orderStr, _ := sol["Order"].(string)
		stageName, _ := sol["StageName"].(string)
		order, err := strconv.Atoi(orderStr)
		if err != nil {
			return fmt.Errorf("could not parse order '%s' for stage '%s'", orderStr, stageName)
		}
		stagesByName[stageName] = flowStage{Name: stageName, Order: order}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query for flow stages: %w", err)
	}

	// Query for tool assignments
	toolQuery := `stage_tool(StageName, ToolName).`
	err = o.flowController.Query(ctx, toolQuery, func(sol map[string]any) error {
		stageName, _ := sol["StageName"].(string)
		toolName, _ := sol["ToolName"].(string)
		if stage, ok := stagesByName[stageName]; ok {
			stage.Tool = toolName
			stagesByName[stageName] = stage
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query for stage tools: %w", err)
	}

	// Convert map to slice and sort
	var stages []flowStage
	for _, stage := range stagesByName {
		if stage.Tool == "" {
			return nil, fmt.Errorf("stage '%s' is defined but has no tool assigned via stage_tool/2", stage.Name)
		}
		stages = append(stages, stage)
	}
	sort.Slice(stages, func(i, j int) bool {
		return stages[i].Order < stages[j].Order
	})
	return stages, nil
}

// dispatchToTool executes the appropriate method on a tool based on its type.
func (o *DeclarativeOrchestrator) dispatchToTool(ctx context.Context, logger core.Logger, stage flowStage, tool any, execContext map[string]any) error {
	query, ok := execContext[contextKeyQuery].(core.Query)
	if !ok {
		return fmt.Errorf("query not found in execution context")
	}
	answer, ok := execContext[contextKeyAnswer].(core.Answer)
	if !ok {
		return fmt.Errorf("answer not found in execution context")
	}

	meta, _ := execContext[contextKeyMeta].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		execContext[contextKeyMeta] = meta
	}

	switch t := tool.(type) {
	case retrieve.Retriever:
		// Ensure Meta is non-nil
		if query.Meta == nil {
			query.Meta = map[string]any{}
		}

		logger.Debugf("calling retriever", "stage", stage.Name, "filters", query.Meta["filters"], "expansions", query.Meta["expansion_terms"])

		req := retrieve.Request{Query: query.Text, Meta: query.Meta}
		res, err := t.Retrieve(ctx, req)
		if err != nil {
			return err
		}
		logger.Infof("retriever returned documents", "stage", stage.Name, "count", len(res.Docs))

		// Store docs for next stages
		execContext[contextKeyDocs] = res.Docs
		meta["retrieved_count"] = len(res.Docs)
		if res.Meta != nil {
			meta["retriever_meta"] = res.Meta
		}

		// Save original docs into the answer meta for post-processing rules
		ans := answer
		if ans.Meta == nil {
			ans.Meta = make(map[string]any)
		}
		ans.Meta["original_docs"] = res.Docs
		execContext[contextKeyAnswer] = ans

	case rerank.Reranker:
		docs, ok := execContext[contextKeyDocs].([]core.Doc)
		if !ok {
			return fmt.Errorf("no documents in context for reranker to process")
		}
		req := rerank.Request{Query: query.Text, Docs: docs}
		rerankedDocs, err := t.Rerank(ctx, req)
		if err != nil {
			return err
		}
		// Overwrite docs with the new, reranked order.
		docs = make([]core.Doc, len(rerankedDocs))
		citations := make([]core.Citation, len(rerankedDocs))
		for i, rd := range rerankedDocs {
			docs[i] = rd.Doc
			citations[i] = core.Citation{
				ID:      rd.Doc.ID,
				Source:  rd.Doc.Source,
				URI:     rd.Doc.URI,
				Snippet: rd.Doc.Text,
				Score:   rd.Score,
			}
		}
		execContext[contextKeyDocs] = docs
		answer.Citations = citations
		execContext[contextKeyAnswer] = answer
		if len(rerankedDocs) > 0 {
			meta["best_score"] = rerankedDocs[0].Score
		}

	case llm.Client:
		if denied, _ := execContext[contextKeyDenialFlag].(bool); denied {
			return nil
		}
		docs, _ := execContext[contextKeyDocs].([]core.Doc) // It's ok if there are no docs.
		passages := make([]string, len(docs))
		for i, d := range docs {
			passages[i] = d.Text
		}
		req := llm.Request{Prompt: query.Text, Context: passages, Data: query.Meta}
		res, err := t.Complete(ctx, req)
		if err != nil {
			return err
		}
		answer.Text = res.Text
		if answer.Meta == nil {
			answer.Meta = make(map[string]any)
		}
		answer.Meta["token_usage"] = res.Usage
		execContext[contextKeyAnswer] = answer

	case core.PostRuleEvaluator:
		docs, _ := execContext[contextKeyDocs].([]core.Doc)
		start := time.Now()
		result, err := t.Post(ctx, query, docs, meta)
		duration := time.Since(start)
		if meter := o.obs.Meter; meter != nil {
			meter.Record("manglekit.rules_post_ms", float64(duration.Milliseconds()))
		}

		if err != nil {
			return err
		}

		fired := 0
		if v, ok := result.Meta["fired_rules"].(int); ok {
			fired = v
		}
		logger.Infof("post-rules executed", "stage", stage.Name, "duration_ms", duration.Milliseconds(), "fired_rules", fired, "denied", result.Denied, "reason", result.Reason)

		execContext[contextKeyDocs] = result.Filtered

		ans := answer
		if ans.Meta == nil {
			ans.Meta = make(map[string]any)
		}
		ans.Meta["rules_post_ms"] = duration.Milliseconds()
		meta["rules_post_ms"] = duration.Milliseconds()
		for k, v := range result.Meta {
			ans.Meta[k] = v
			meta[k] = v
		}

		if len(ans.Citations) > 0 {
			allowed := make(map[string]struct{}, len(result.Filtered))
			for _, doc := range result.Filtered {
				allowed[doc.ID] = struct{}{}
			}
			filteredCitations := make([]core.Citation, 0, len(ans.Citations))
			for _, citation := range ans.Citations {
				if _, ok := allowed[citation.ID]; ok {
					filteredCitations = append(filteredCitations, citation)
				}
			}
			ans.Citations = filteredCitations
		}

		if result.Denied {
			ans.Text = result.Reason
			ans.Meta["denied"] = true
			ans.Meta["denial_reason"] = result.Reason
			execContext[contextKeyDenialFlag] = true
			execContext[contextKeyDenialReason] = result.Reason
			execContext[contextKeyAnswer] = ans
			return fmt.Errorf("%w: %s", core.ErrDenied, result.Reason)
		}

		execContext[contextKeyAnswer] = ans

	case core.RuleSet:
		// Rule sets are invoked explicitly via pre/post stages outside the dispatch loop.
		// Treat declarative invocations that reference the rules engine directly as no-ops.
		return nil

	default:
		return fmt.Errorf("unsupported tool type: %T", tool)
	}
	return nil
}
