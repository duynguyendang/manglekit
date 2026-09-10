package ooda

import (
	"sort"
)

const (
	// DefaultMaxContextTokens is the default budget for INT8 context atoms.
	DefaultMaxContextTokens = 24000
	// ShaveThreshold is the minimum weight to keep during context shaving.
	ShaveThreshold = 0.5
)

// PinAxiom adds an immutable axiom to the AttentionSink (FP32).
// Axioms are never pruned and are pinned at the top of LLM prompts.
func PinAxiom(frame *CognitiveFrame, atom Atom) {
	atom.Weight = 1.0 // FP32: always max confidence
	frame.AttentionSink = append(frame.AttentionSink, atom)
}

// AddContext adds a soft fact to the Context (INT8).
// Weight can be 0.1-0.9. Low-weight atoms are prunable.
func AddContext(frame *CognitiveFrame, atom Atom) {
	if atom.Weight == 0 {
		atom.Weight = 0.5 // Default weight
	}
	frame.Context = append(frame.Context, atom)
}

// ShaveContext removes low-weight atoms when context exceeds the token budget.
// Atoms with Weight < ShaveThreshold are dropped first, then sorted by weight descending.
func ShaveContext(frame *CognitiveFrame, maxTokens int) {
	if maxTokens <= 0 {
		maxTokens = DefaultMaxContextTokens
	}

	currentTokens := estimateTokens(frame.Context)
	if currentTokens <= maxTokens {
		return
	}

	// Drop atoms with Weight < threshold
	shaved := make([]Atom, 0, len(frame.Context))
	for _, atom := range frame.Context {
		if atom.Weight >= ShaveThreshold {
			shaved = append(shaved, atom)
		}
	}

	// Sort by weight descending
	sort.Slice(shaved, func(i, j int) bool {
		return shaved[i].Weight > shaved[j].Weight
	})

	// Trim to budget
	frame.Context = trimmedToBudget(shaved, maxTokens)
}

// GetAttentionSink returns all AttentionSink atoms (FP32, immutable).
func GetAttentionSink(frame *CognitiveFrame) []Atom {
	return frame.AttentionSink
}

// GetContext returns all Context atoms (INT8, pruneable).
func GetContext(frame *CognitiveFrame) []Atom {
	return frame.Context
}

// EstimateTotalTokens estimates total token count for the frame.
func EstimateTotalTokens(frame *CognitiveFrame) int {
	return estimateTokens(frame.AttentionSink) + estimateTokens(frame.Context)
}

// PruneColdAtoms removes atoms with access count <= 1 and weight < 0.5.
func PruneColdAtoms(frame *CognitiveFrame, activity map[string]int) {
	pruned := make([]Atom, 0, len(frame.Context))
	for _, atom := range frame.Context {
		key := atom.Predicate + ":" + atom.Subject
		count := activity[key]
		if count <= 1 && atom.Weight < ShaveThreshold {
			continue // Prune cold, low-weight atoms
		}
		pruned = append(pruned, atom)
	}
	frame.Context = pruned
}

// estimateTokens approximates token count from atoms.
// Rough estimate: 4 tokens per atom (predicate + subject + object).
func estimateTokens(atoms []Atom) int {
	total := 0
	for _, atom := range atoms {
		total += len(atom.Predicate) + len(atom.Subject) + len(atom.Object)
		total += 3 // separators
	}
	// Rough: 4 chars ≈ 1 token
	if total > 0 {
		return total / 4
	}
	return len(atoms) * 4 // Fallback: 4 tokens per atom
}

// trimmedToBudget trims atoms to fit within the token budget.
func trimmedToBudget(atoms []Atom, maxTokens int) []Atom {
	used := 0
	var result []Atom
	for _, atom := range atoms {
		tokens := estimateTokens([]Atom{atom})
		if used+tokens > maxTokens {
			break
		}
		result = append(result, atom)
		used += tokens
	}
	return result
}
