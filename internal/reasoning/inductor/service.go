package inductor

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit-wip/internal/audit"
	"github.com/duynguyendang/manglekit-wip/internal/core/domain"
	"github.com/duynguyendang/manglekit-wip/internal/core/ports"
	"github.com/google/uuid"
)

// Inductor is the pipeline for Knowledge Induction (7.1, 7.2 LLD).
// It distills raw Markdown/policies into verified Tier 2 Datalog Genes.
type Inductor struct {
	llm           ports.GenerativePort
	auditor       *audit.Auditor
	perception    ports.PerceptionPort
	storage       ports.GenomeStoragePort
	evidenceStore ports.EvidenceStorePort
}

func New(
	llm ports.GenerativePort,
	auditor *audit.Auditor,
	perception ports.PerceptionPort,
	storage ports.GenomeStoragePort,
	evidenceStore ports.EvidenceStorePort,
) *Inductor {
	return &Inductor{
		llm:           llm,
		auditor:       auditor,
		perception:    perception,
		storage:       storage,
		evidenceStore: evidenceStore,
	}
}

// Process executes the sequential distillation workflow.
func (i *Inductor) Process(ctx context.Context, signal domain.Signal) (string, error) {

	// 1. Normalize raw content
	// payload, _ := i.perception.Normalize(ctx, signal)

	// 2. Extract Evidence & Deduplicate
	dedupedID, found := i.evidenceStore.FindSimilar(ctx, string(signal.Intent), signal.RawContent, 0.9)
	if !found {
		dedupedID = uuid.NewString()
		_ = i.evidenceStore.Save(ctx, ports.EvidenceItem{
			ID:      dedupedID,
			Content: signal.RawContent,
			Intent:  string(signal.Intent),
		})
	}

	// 3. GenerativePort LLM Distillation
	// Instruct LLM to generate Datalog rules from the text
	rawDatalog, err := i.llm.Induce(ctx, signal.RawContent)
	if err != nil {
		return "", fmt.Errorf("induction generation failed: %w", err)
	}

	// 4. Sanitize (strip markdown fences)
	cleanDatalog := i.sanitize(rawDatalog)

	// 5. Shadow Audit (Verify against Tier 0)
	frame := &domain.CognitiveFrame{
		TaskType:   domain.TaskTypeInduction,
		OutputType: domain.OutputTypeRule,
		Draft:      []byte(cleanDatalog),
		Intent:     signal.Intent,
		// ActiveGenes must contain Tier 0 at minimum
		ActiveGenes: []domain.DomainGene{
			{Name: "Kernel", Tier: domain.Tier0Kernel},
		},
	}

	res, err := i.auditor.Verify(ctx, frame)
	if err != nil || !res.Pass {
		return "", fmt.Errorf("induced rules failed shadow audit: %w", err)
	}

	// 6. Crystallization (PersistKnowledge)
	// Create the formal gene path based on intent UUID
	genePath := i.storage.ResolvePath("induced", string(signal.Intent)+".dl")

	err = i.storage.PersistKnowledge(ctx, string(signal.Intent), []byte(cleanDatalog))
	if err != nil {
		return "", fmt.Errorf("failed to persist induced gene %s: %w", genePath, err)
	}

	return cleanDatalog, nil
}

// sanitize strips markdown blocks and invalid tokens (LLD 7.3)
func (i *Inductor) sanitize(input string) string {
	lines := strings.Split(input, "\n")
	var out []string

	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if strings.HasPrefix(trim, "```") {
			continue // strip fences
		}
		if strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#") {
			continue // strip comments
		}
		if trim == "" {
			continue
		}
		out = append(out, trim)
	}

	return strings.Join(out, "\n")
}
