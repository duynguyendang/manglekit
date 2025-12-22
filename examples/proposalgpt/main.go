package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/examples/proposalgpt/internal/actions"
	"github.com/duynguyendang/manglekit/examples/proposalgpt/internal/knowledge"
	"github.com/duynguyendang/manglekit/examples/proposalgpt/internal/types"
	_ "github.com/duynguyendang/manglekit/providers/google" // Register Google Provider
	"github.com/duynguyendang/manglekit/sdk"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	ctx := context.Background()

	// 1. Initialize Client
	// This loads "examples/proposalgpt/mangle.yaml" which configures "google_llm".
	client, err := sdk.NewClientFromFile(ctx, "examples/proposalgpt/mangle.yaml")
	if err != nil {
		log.Fatalf("Failed to init client: %v", err)
	}
	defer client.Shutdown(ctx)

	// 2. Load Logic (Gap, Selection, Validation)
	policyContent := loadPolicies()
	if err := client.Engine().LoadPolicy(ctx, policyContent); err != nil {
		log.Fatalf("Failed to load policies: %v", err)
	}

	// 3. Setup Actions
	// Create a proxy to the "google_llm" defined in YAML
	llmProxy := client.Action("google_llm")

	// Extractor
	extractor, err := actions.NewExtractor(llmProxy)
	if err != nil {
		log.Fatalf("Failed to create extractor: %v", err)
	}

	// Planner
	planner := actions.NewPlanner(llmProxy)

	// 4. Execution Flow
	rfpPath := "examples/proposalgpt/test/rfps/project_brief.md"
	if len(os.Args) > 1 {
		rfpPath = os.Args[1]
	}

	fmt.Printf("--- ProposalGPT: Processing %s ---\n", rfpPath)

	// Step 4.1: Ingest
	rfpText, err := actions.IngestRFP(rfpPath)
	if err != nil {
		log.Fatalf("Ingest failed: %v", err)
	}

	// Step 4.2: Extract
	resEnv, err := extractor.Execute(ctx, core.NewEnvelope(rfpText))
	if err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}
	facts, ok := resEnv.Payload.(types.ExtractedFacts)
	if !ok {
		log.Fatalf("Extraction payload mismatch: %T", resEnv.Payload)
	}
	fmt.Printf("✅ Facts Extracted: Budget=$%.2f, Keywords=%v\n", facts.Budget, facts.Keywords)

	// Step 4.3: Reflect (Load Usage Facts to Engine)
	datalogFacts := convertToDatalog(facts)
	if err := client.LoadFacts(datalogFacts); err != nil {
		log.Fatalf("Failed to load facts: %v", err)
	}

	// Step 4.4: Reason (Steering & Selection)
	// Check for Steering (e.g. Missing Info)
	steer, _, err := client.Engine().EvaluateSteering(ctx, core.NewEnvelope(facts))
	if err != nil {
		log.Printf("Steering check warning: %v", err)
	}
	if steer == "ask_clarification" {
		fmt.Println("⚠️ HALT: Missing critical information in RFP. Please clarify budget/constraints.")
		return
	}

	// Check for Selection (Playbook)
	configs, err := client.Engine().GetActionConfig(ctx, core.NewEnvelope(facts))
	if err != nil {
		log.Fatalf("Config retrieval failed: %v", err)
	}

	playbookID := configs["playbook_id"]
	playbookFile := configs["playbook_file"]

	if playbookID == "" {
		fmt.Println("❌ No suitable playbook found for this RFP request.")
		return
	}
	fmt.Printf("✅ Selected Playbook: %s (File: %s)\n", playbookID, playbookFile)

	// Step 4.5: Execute (Plan)
	playbookPath := "examples/proposalgpt/internal/knowledge/library/" + playbookFile
	pb, err := knowledge.LoadPlaybookFromFile(playbookID, playbookPath)
	if err != nil {
		log.Fatalf("Failed to load playbook '%s' from '%s': %v", playbookID, playbookPath, err)
	}

	fmt.Printf("Generating Proposal using strategy: %s...\n", pb.Name)

	planInput := actions.PlanningInput{
		Facts:    facts,
		Playbook: *pb,
	}

	// Wrapper for Planner (Supervised if we want validation, but PlannerAction calls LLM directly)
	// If we want validation.dl to run, we should supervise the Planner.
	// But Planner output is ProposalDraft struct. Validation rules check "proposal_content".
	// We need to ensure Planner wrapper puts generic map or struct that engine can read.
	// For simplicity, we run Planner then manually validate if needed, or let Supervise handle it.
	// Let's use Supervise for the Planner!
	supervisedPlanner := client.Supervise(planner)

	planRes, err := supervisedPlanner.Execute(ctx, core.NewEnvelope(planInput))
	if err != nil {
		fmt.Printf("❌ Planning Rejected by Governance: %v\n", err)
		// Try to show retry hint if available
		return
	}

	proposal := planRes.Payload.(types.ProposalDraft)
	fmt.Printf("\n✅ Proposal Generated: %s\n", proposal.Title)
	fmt.Printf("Architecture: %s\n", proposal.Architecture)
	fmt.Printf("Summary: %s\n", proposal.ExecutiveSummary)
	fmt.Printf("\n--- Full Application Proposal ---\n\n%s\n", proposal.Content)

	// Optional warning if validation passed but had non-critical issues (not implemented)
}

// Helpers

func loadPolicies() string {
	files := []string{
		"examples/proposalgpt/rules/gap.dl",
		"examples/proposalgpt/rules/selection.dl",
		"examples/proposalgpt/rules/validation.dl",
	}
	var sb strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			log.Printf("Warning: failed to read policy %s: %v", f, err)
			continue
		}
		sb.Write(b)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func convertToDatalog(f types.ExtractedFacts) []string {
	var facts []string

	// Value predicates
	if f.Budget > 0 {
		facts = append(facts, fmt.Sprintf(`value("rfp", "budget", %f)`, f.Budget))
	}
	if f.CloudPref != "" {
		facts = append(facts, fmt.Sprintf(`value("rfp", "cloud_pref", "%s")`, f.CloudPref))
	}

	// Keywords
	for _, k := range f.Keywords {
		facts = append(facts, fmt.Sprintf(`has_keyword("%s")`, strings.ToLower(k)))
	}

	return facts
}
