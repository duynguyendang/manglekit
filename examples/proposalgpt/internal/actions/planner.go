package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/examples/proposalgpt/internal/knowledge"
	"github.com/duynguyendang/manglekit/examples/proposalgpt/internal/types"
)

// PlanningInput is the input for the Planner Action.
type PlanningInput struct {
	Facts    types.ExtractedFacts
	Playbook knowledge.Playbook
}

// PlannerAction generates a proposal draft.
type PlannerAction struct {
	llm core.Action
}

// NewPlanner creates a new PlannerAction.
func NewPlanner(llm core.Action) core.Action {
	return &PlannerAction{llm: llm}
}

func (a *PlannerAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	var in PlanningInput
	// Use type assertion or casting logic depending on how it's passed
	// If it's passed as a struct directly:
	if v, ok := input.Payload.(PlanningInput); ok {
		in = v
	} else {
		return core.Envelope{}, fmt.Errorf("planner: expected payload of type PlanningInput, got %T", input.Payload)
	}

	// Build Prompt
	prompt := fmt.Sprintf(`SYSTEM: You are a Solution Architect.
Using the following RFP Facts and Playbook, generate a structured Proposal Draft.

PLAYBOOK:
Name: %s
Description: %s
Architecture: %s
Risks: %v
NFRs: %v

RFP FACTS:
Summary: %s
Keywords: %v
Budget: %f
Compliance: %v

TASK:
Write a proposal draft in JSON format matching this schema:
{
	"title": "string",
	"executive_summary": "string",
	"architecture": "string",
	"platform": "string",
	"risk_analysis": "string",
	"implementation": "string",
	"content": "string (full markdown)"
}
Ensure the content mentions the Architecture strategy explicitly.
Return ONLY valid JSON.
`,
		in.Playbook.Name, in.Playbook.Description, in.Playbook.ArchPattern, in.Playbook.Risks, in.Playbook.CriticalNFRs,
		in.Facts.Summary, in.Facts.Keywords, in.Facts.Budget, in.Facts.Compliance,
	)

	// Call LLM
	resp, err := a.llm.Execute(ctx, core.NewEnvelope(prompt))
	if err != nil {
		return core.Envelope{}, err
	}

	outText, ok := resp.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("planner: llm response not string")
	}

	// Clean JSON (basic)
	clean := strings.TrimSpace(outText)
	clean = strings.TrimPrefix(clean, "```json")
	clean = strings.TrimPrefix(clean, "```")
	clean = strings.TrimSuffix(clean, "```")
	clean = strings.TrimSpace(clean)

	var draft types.ProposalDraft
	if err := json.Unmarshal([]byte(clean), &draft); err != nil {
		return core.Envelope{}, fmt.Errorf("planner: failed to parse json: %w (Output: %s)", err, clean)
	}

	return core.NewEnvelope(draft), nil
}

func (a *PlannerAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "proposal_planner",
		Type: "generator",
	}
}
