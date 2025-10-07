package declarative

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/llm"
	"github.com/duynguyendang/manglekit/rerank"
	"github.com/duynguyendang/manglekit/retrieve"
)

const (
	// Standard keys for the execution context map.
	contextKeyQuery  = "query"
	contextKeyDocs   = "docs"
	contextKeyAnswer = "answer"
)

// DeclarativeOrchestrator implements the core.Orchestrator interface. It executes
// workflows defined declaratively as Mangle Datalog facts. This separates the
// pipeline's logic (the "how" and "when") from the configuration of its
// components (the "what with").
type DeclarativeOrchestrator struct {
	flowController core.FlowController
	tools          map[string]any // Maps ToolName (e.g., "hybrid_search") to its provider instance.
	flowName       string
}

// flowStage holds the information about a single stage in a declarative flow.
type flowStage struct {
	Name  string
	Order int
	Tool  string
}

// New creates a new DeclarativeOrchestrator.
// It requires a Mangle FlowController to query for the workflow definition and
// evaluate runtime rules, a map of fully constructed "tool" instances, and the
// name of the flow to execute.
func New(fc core.FlowController, tools map[string]any, flowName string) (core.Orchestrator, error) {
	if fc == nil {
		return nil, fmt.Errorf("DeclarativeOrchestrator requires a non-nil FlowController")
	}
	if tools == nil {
		return nil, fmt.Errorf("DeclarativeOrchestrator requires a non-nil tools map")
	}
	if flowName == "" {
		return nil, fmt.Errorf("DeclarativeOrchestrator requires a flow name")
	}
	return &DeclarativeOrchestrator{
		flowController: fc,
		tools:          tools,
		flowName:       flowName,
	}, nil
}

// Retriever returns the first tool that implements the retrieve.Retriever interface.
// This is useful for runtime operations like updating an in-memory vector store.
// It returns nil if no retriever tool is found in the orchestrator's configuration.
func (o *DeclarativeOrchestrator) Retriever() any {
	for _, tool := range o.tools {
		if r, ok := tool.(retrieve.Retriever); ok {
			return r
		}
	}
	return nil
}

// Run executes the declarative workflow.
// This method is the core interpreter of the declarative pipeline. It queries the
// Mangle engine for the stages of the specified flow, sorts them, and executes
// them sequentially. It passes a context map between stages to share state.
func (o *DeclarativeOrchestrator) Run(ctx context.Context, q core.Query) (core.Answer, error) {
	// 1. Query the static execution plan.
	stages, err := o.getFlowStages(ctx)
	if err != nil {
		return core.Answer{}, fmt.Errorf("could not get flow stages for flow '%s': %w", o.flowName, err)
	}
	if len(stages) == 0 {
		return core.Answer{}, fmt.Errorf("no stages found for flow '%s'", o.flowName)
	}

	// 2. Evaluate pre-rules to get runtime information like which stages to skip.
	preResult, err := o.flowController.Evaluate(core.Pre, q, nil)
	if err != nil {
		return core.Answer{}, fmt.Errorf("pre-rules evaluation failed: %w", err)
	}
	if !preResult.Allowed {
		return core.Answer{Meta: map[string]any{"denial_reason": preResult.Reason}}, core.ErrDenied
	}

	// 3. Create the execution context.
	execContext := map[string]any{
		contextKeyQuery:  q,
		contextKeyAnswer: core.Answer{Meta: map[string]any{}},
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
		if err := o.dispatchToTool(ctx, tool, execContext); err != nil {
			return core.Answer{}, fmt.Errorf("execution of tool '%s' for stage '%s' failed: %w", stage.Tool, stage.Name, err)
		}
	}

	// 5. Assemble the final answer.
	finalAnswer, ok := execContext[contextKeyAnswer].(core.Answer)
	if !ok {
		return core.Answer{}, fmt.Errorf("execution context ended without a valid answer object")
	}
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
func (o *DeclarativeOrchestrator) dispatchToTool(ctx context.Context, tool any, execContext map[string]any) error {
	query, ok := execContext[contextKeyQuery].(core.Query)
	if !ok {
		return fmt.Errorf("query not found in execution context")
	}
	answer, ok := execContext[contextKeyAnswer].(core.Answer)
	if !ok {
		return fmt.Errorf("answer not found in execution context")
	}

	switch t := tool.(type) {
	case retrieve.Retriever:
		req := retrieve.Request{Query: query.Text, Meta: query.Meta}
		res, err := t.Retrieve(req)
		if err != nil {
			return err
		}
		execContext[contextKeyDocs] = res.Docs

	case rerank.Reranker:
		docs, ok := execContext[contextKeyDocs].([]core.Doc)
		if !ok {
			return fmt.Errorf("no documents in context for reranker to process")
		}
		req := rerank.Request{Query: query.Text, Docs: docs}
		rerankedDocs, err := t.Rerank(req)
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

	case llm.Client:
		docs, _ := execContext[contextKeyDocs].([]core.Doc) // It's ok if there are no docs.
		passages := make([]string, len(docs))
		for i, d := range docs {
			passages[i] = d.Text
		}
		req := llm.Request{Prompt: query.Text, Context: passages, Data: query.Meta}
		res, err := t.Complete(req)
		if err != nil {
			return err
		}
		answer.Text = res.Text
		if answer.Meta == nil {
			answer.Meta = make(map[string]any)
		}
		answer.Meta["token_usage"] = res.Usage
		execContext[contextKeyAnswer] = answer

	default:
		return fmt.Errorf("unsupported tool type: %T", tool)
	}
	return nil
}