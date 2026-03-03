package logic

import (
	"sort"

	"github.com/duynguyendang/manglekit/internal/core/domain"
)

// ContextManager handles token allocation and rule-based pruning for LLM prompts.
// It ensures the LLM never exceeds its context window while maximizing relevant knowledge.
type ContextManager struct {
	BaseBuffer     int // Default safety margin (e.g., 200 tokens)
	ShaveThreshold int // Estimated token count at which Intensive Shaving occurs
}

// NewContextManager creates a default context manager.
func NewContextManager() *ContextManager {
	return &ContextManager{
		BaseBuffer:     200,
		ShaveThreshold: 24000,
	}
}

// CalculateContextBudget determines how many tokens remain for the soft context (INT8 facts).
// LLD 5.3: It employs "Logic Success Sharpening". As the agent fails logic audits
// (LogicSuccess drops), the safety buffer increases, restricting the context to
// force the LLM to focus only on the most critical facts.
func (cm *ContextManager) CalculateContextBudget(
	totalBudget, fp32Usage int, logicSuccess float64, previousOutputTokens int,
) int {
	// 1.0 + (1.0 - 0.8) = 1.2x buffer
	sharpeningMultiplier := 1.0 + (1.0 - logicSuccess)
	adjustedBuffer := int(float64(cm.BaseBuffer) * sharpeningMultiplier)

	remaining := totalBudget - fp32Usage - adjustedBuffer - previousOutputTokens
	if remaining < 0 {
		return 0
	}
	return remaining
}

// FilterStrategic performs the first pass of context pruning according to LLD 5.3.
// It retains atoms that match the Current Intent, OR are mathematically certain (Weight >= 0.9), OR are global context (no origin intent).
func (cm *ContextManager) FilterStrategic(atoms []domain.Atom, currentIntent domain.IntentStr) []domain.Atom {
	var kept []domain.Atom
	for _, a := range atoms {
		if a.OriginIntent == "" || a.OriginIntent == currentIntent || a.Weight >= 0.9 {
			kept = append(kept, a)
		}
	}
	return kept
}

// Prune restricts the context to the maxTokens budget.
// It estimates token counts using a fast heuristic, applies Intelligent Shaving if bloated,
// and sorts by Weight (Confidence) descending.
func (cm *ContextManager) Prune(atoms []domain.Atom, maxTokens int) []domain.Atom {
	if maxTokens <= 0 {
		return []domain.Atom{}
	}

	type atomWithSize struct {
		atom domain.Atom
		size int
	}

	totalEstimated := 0
	paramList := make([]atomWithSize, 0, len(atoms))

	// Fast heuristic: ~4 chars per token for English text
	for _, a := range atoms {
		estLen := len(a.Subject) + len(a.Predicate) + len(a.Object) + 15 // +15 formatting chars
		tokenCount := estLen / 4
		paramList = append(paramList, atomWithSize{atom: a, size: tokenCount})
		totalEstimated += tokenCount
	}

	// Intelligent Shaving: If context exceeds the threshold (e.g., 24k tokens),
	// drop all low-confidence atoms (Weight < 0.5) to clear noise.
	shouldShave := totalEstimated > cm.ShaveThreshold

	var qualified []atomWithSize
	for _, item := range paramList {
		if shouldShave && item.atom.Weight < 0.5 {
			continue
		}
		qualified = append(qualified, item)
	}
	paramList = qualified

	// Sort by Weight descending (highest confidence first). Stable sort preserves insertion order.
	sort.SliceStable(paramList, func(i, j int) bool {
		return paramList[i].atom.Weight > paramList[j].atom.Weight
	})

	var finalAtoms []domain.Atom
	currentTokens := 0

	for _, item := range paramList {
		if currentTokens+item.size > maxTokens {
			break // Budget exhausted
		}
		finalAtoms = append(finalAtoms, item.atom)
		currentTokens += item.size
	}

	return finalAtoms
}
