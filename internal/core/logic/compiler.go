package logic

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit-wip/internal/core/domain"
)

// RefinementContext encapsulates audit failures used to correct the LLM.
type RefinementContext struct {
	AuditResult   *domain.AuditResult
	PreviousDraft interface{}
}

// PromptConfig holds all possible inputs for the functional options compiler.
type PromptConfig struct {
	Intent             domain.IntentStr
	TaskType           domain.TaskType
	Context            []domain.Atom // Soft Logic (INT8)
	Axioms             []domain.Atom // Hard Logic (FP32) - Attention Sink
	Genes              []domain.DomainGene
	Facts              []byte // N-Quads
	SteeringMagnitude  float64
	InjectParadox      bool
	RefinementFeedback *RefinementContext
	Pass               int // Multi-pass iteration
	PreviousOutput     string
	SessionHistory     []domain.AuditResult
}

// PromptOption defines the functional option pattern.
type PromptOption func(*PromptConfig)

// Functional Options Setters
func WithIntent(i domain.IntentStr) PromptOption   { return func(c *PromptConfig) { c.Intent = i } }
func WithTaskType(t domain.TaskType) PromptOption  { return func(c *PromptConfig) { c.TaskType = t } }
func WithContext(a []domain.Atom) PromptOption     { return func(c *PromptConfig) { c.Context = a } }
func WithAxioms(a []domain.Atom) PromptOption      { return func(c *PromptConfig) { c.Axioms = a } }
func WithGenes(g []domain.DomainGene) PromptOption { return func(c *PromptConfig) { c.Genes = g } }
func WithFacts(f []byte) PromptOption              { return func(c *PromptConfig) { c.Facts = f } }
func WithSteering(m float64, p bool) PromptOption {
	return func(c *PromptConfig) { c.SteeringMagnitude = m; c.InjectParadox = p }
}
func WithRefinement(r *RefinementContext) PromptOption {
	return func(c *PromptConfig) { c.RefinementFeedback = r }
}
func WithPass(p int) PromptOption        { return func(c *PromptConfig) { c.Pass = p } }
func WithPrevious(o string) PromptOption { return func(c *PromptConfig) { c.PreviousOutput = o } }
func WithHistory(h []domain.AuditResult) PromptOption {
	return func(c *PromptConfig) { c.SessionHistory = h }
}

// Compiler orchestrates prompt assembly based on LLD 5.1/5.2.
type Compiler struct {
	// dependencies like template engine can go here
}

// NewCompiler initializes the prompt compiler.
func NewCompiler() *Compiler {
	return &Compiler{}
}

// Compile builds the final string to send to the GenerativePort.
func (c *Compiler) Compile(ctx context.Context, intent domain.IntentStr, options ...interface{}) (string, error) {
	config := &PromptConfig{
		Pass:   1, // default
		Intent: intent,
	}

	for _, opt := range options {
		if po, ok := opt.(PromptOption); ok {
			po(config)
		}
	}

	var sb strings.Builder

	// 1. Attention Sink Strategy (LLD 5.2): Pin Tier 0 Axioms to the very top.
	sb.WriteString("[SYSTEM PROMPT]\n## ABSOLUTE TRUTHS (Never Override)\n")
	for _, axiom := range config.Axioms {
		sb.WriteString(fmt.Sprintf("- %s(%s, %s)\n", axiom.Predicate, axiom.Subject, axiom.Object))
	}
	sb.WriteString("\n")

	// 2. Active Datalog Genes translated to Natural Language
	if len(config.Genes) > 0 {
		sb.WriteString("## ACTIVE SYSTEM POLICIES\n")
		for _, gene := range config.Genes {
			sb.WriteString(c.mangleToNaturalLanguage(gene.Rules))
		}
		sb.WriteString("\n")
	}

	// 3. Teacher-Student Refinement Correction (If audit failed)
	if config.RefinementFeedback != nil && !config.RefinementFeedback.AuditResult.Pass {
		sb.WriteString("## CRITICAL EXECUTION FAILURE\n")
		sb.WriteString("Your previous proposal violated system invariants. You MUST correct this.\n")
		sb.WriteString(fmt.Sprintf("Violation Path: %s\n", config.RefinementFeedback.AuditResult.ConflictPath))
		if config.RefinementFeedback.AuditResult.ProofTree != nil {
			sb.WriteString(fmt.Sprintf("Rule Trace: %s\n", config.RefinementFeedback.AuditResult.ProofTree.Rule))
		}
		sb.WriteString("\n")
	}

	// 4. Soft Context (Pruned by ContextManager)
	if len(config.Context) > 0 {
		sb.WriteString("## OBSERVED CONTEXT\n")
		for _, atom := range config.Context {
			sb.WriteString(fmt.Sprintf("- %s(%s, %s) [w=%.2f]\n", atom.Predicate, atom.Subject, atom.Object, atom.Weight))
		}
		sb.WriteString("\n")
	}

	// 5. Hard Structural Facts (N-Quads from tri-stream recall)
	if len(config.Facts) > 0 {
		sb.WriteString("## STRUCTURAL TOPOLOGY (Required Sub-graph)\n")
		sb.WriteString(string(config.Facts))
		sb.WriteString("\n")
	}

	// 6. Objective / Task
	sb.WriteString("## TASK\n")
	sb.WriteString(fmt.Sprintf("Intent: %s\n", config.Intent))
	sb.WriteString(fmt.Sprintf("Task Type: %s\n", config.TaskType))

	if config.InjectParadox {
		sb.WriteString("WARNING: Cognitive pressure is EXTREME. Limit creative assumptions. Act deterministically and enumerate steps carefully.\n")
	}

	return sb.String(), nil
}

// mangleToNaturalLanguage translates raw Datalog constraints into LLM instructions.
func (c *Compiler) mangleToNaturalLanguage(rules []byte) string {
	// A naive string-based translation approach for the MVP.
	// In production, this uses an AST walker over the parsed Mangle program.
	str := string(rules)
	lines := strings.Split(str, "\n")
	var out []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "halt(") {
			out = append(out, "- NEVER allow action where: "+line)
		} else if strings.HasPrefix(line, "warn(") {
			out = append(out, "- Avoid action if possible: "+line)
		} else if line != "" {
			out = append(out, "- Infer: "+line)
		}
	}
	return strings.Join(out, "\n") + "\n"
}
