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
Using the following RFP Facts and Playbook, generate a comprehensive, detailed, and professional Proposal Draft.

PLAYBOOK GUIDELINE:
"""%s"""

RFP FACTS:
Summary: %s
Original Content: """%s"""
Keywords: %v
Budget: %f
Compliance: %v

TASK:
Write a LONG-FORM proposal draft in JSON format matching this schema:
{
	"title": "Professional Title",
	"executive_summary": "High-level summary",
	"architecture": "Selected architecture strategy name",
	"platform": "Proposed technology stack",
	"risk_analysis": "Summary of risks and mitigations",
	"implementation": "High-level roadmap",
	"content": "FULL MARKDOWN CONTENT (This is the most important part)"
}

Instructions for "content" field:
- Use the "PLAYBOOK GUIDELINE" above as the structural and content guide for the proposal.
- The proposal must be comprehensive and professional (multi-page).
- Use the original RFP content to tailor the proposal specifically to the client's needs.
- Do not use placeholders. Write actual, persuasive content.

Return ONLY valid JSON.
`,
		in.Playbook.RawContent,
		in.Facts.Summary, in.Facts.OriginalContent, in.Facts.Keywords, in.Facts.Budget, in.Facts.Compliance,
	)

	fmt.Println("Prompt:", prompt)

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
