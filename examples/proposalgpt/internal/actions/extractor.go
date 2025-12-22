package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/examples/proposalgpt/internal/types"
)

// RfpExtractor extracts structured data from text using an LLM with playbook awareness.
type RfpExtractor struct {
	llm core.Action
}

// NewExtractor creates an action that extracts RFP facts.
func NewExtractor(llm core.Action) (core.Action, error) {
	return &RfpExtractor{llm: llm}, nil
}

// Execute extracts facts from the RFP text.
func (e *RfpExtractor) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	inputText, ok := input.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("input payload must be string, got %T", input.Payload)
	}

	prompt := fmt.Sprintf(`SYSTEM: You are a specialized RFP Analysis Agent.
Your goal is to extract key information from the Request for Proposal (RFP) to help select the correct Solution Design Playbook.

The available Playbooks are:
1. Greenfield Application (Keywords: "build from scratch", "new app", "greenfield")
2. System Re-architecting/Modernization (Keywords: "monolith", "legacy", "strangler fig", "re-architect")
3. Lift & Shift Migration (Keywords: "lift and shift", "datacenter exit", "aws mgn", "vmware")
4. Feature Enhancement (Keywords: "new feature", "addon", "extension", "functionality")
5. Data Migration (Keywords: "data migration", "transfer data", "etl")
6. Report Migration (Keywords: "bi migration", "tableau", "power bi", "reports")
7. Platform Setup (Keywords: "platform engineering", "developer experience", "kubernetes")
8. Data Platform Modernization (Keywords: "data warehouse", "data lake", "performance issues", "modernization")
9. AI/ML Platform (Keywords: "mlops", "machine learning", "model training")
10. CRM Implementation (Keywords: "crm", "salesforce", "dynamics", "customer relationship")

Extract the following JSON structure:
{
  "summary": "Brief summary of the RFP",
  "keywords": ["List", "of", "strong", "keywords", "from", "the", "Playbook", "list", "that", "match"],
  "budget": 0.00,
  "deadline": "YYYY-MM-DD",
  "pain_points": ["List of problems"],
  "cloud_pref": "AWS/GCP/Azure or Empty",
  "compliance": ["List of compliance requirements"]
}

Return ONLY the raw JSON object. No markdown.

USER: %s`, inputText)

	// Call LLM
	respEnv, err := e.llm.Execute(ctx, core.NewEnvelope(prompt))
	if err != nil {
		return core.Envelope{}, fmt.Errorf("llm extraction failed: %w", err)
	}

	respText, ok := respEnv.Payload.(string)
	if !ok {
		return core.Envelope{}, fmt.Errorf("llm response payload is not string, got %T", respEnv.Payload)
	}

	// Clean up potential markdown code blocks
	respText = strings.TrimPrefix(respText, "```json")
	respText = strings.TrimPrefix(respText, "```")
	respText = strings.TrimSuffix(respText, "```")
	respText = strings.TrimSpace(respText)

	var facts types.ExtractedFacts
	if err := json.Unmarshal([]byte(respText), &facts); err != nil {
		return core.Envelope{}, fmt.Errorf("failed to unmarshal extracted facts: %v (response: %s)", err, respText)
	}

	return core.NewEnvelope(facts), nil
}

// Metadata returns the metadata for this extractor action.
func (e *RfpExtractor) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "rfp_extractor",
		Type: "extractor",
	}
}
