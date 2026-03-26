package ooda

import (
	"math"
	"strings"
)

// MeasureSaliency evaluates input importance based on keyword detection.
// Returns a list of saliency signals found in the input.
func MeasureSaliency(input string) []string {
	var signals []string
	lower := strings.ToLower(input)

	keywords := map[string]string{
		"production": "high_priority",
		"critical":   "high_priority",
		"security":   "high_priority",
		"urgent":     "high_priority",
		"migration":  "domain_specific",
		"deploy":     "domain_specific",
		"database":   "domain_specific",
		"api":        "domain_specific",
	}

	for keyword, category := range keywords {
		if strings.Contains(lower, keyword) {
			signals = append(signals, category)
		}
	}

	// Check for known project patterns
	if len(signals) > 0 {
		signals = append(signals, "known_pattern")
	}

	return signals
}

// CalculateEntropy computes entropy from conflict count and total rules.
// Returns 0.0 (no conflicts) to 1.0 (all rules conflicted).
func CalculateEntropy(conflictCount, totalRules int) float64 {
	if totalRules == 0 {
		return 0
	}
	return math.Min(float64(conflictCount)/float64(totalRules), 1.0)
}

// CalculateActivity builds a usage map from atoms accessed during Orient.
func CalculateActivity(atoms []Atom) map[string]int {
	activity := make(map[string]int)
	for _, atom := range atoms {
		key := atom.Predicate + ":" + atom.Subject
		activity[key]++
	}
	return activity
}

// DetectConflicts counts atoms that contradict AttentionSink axioms.
// A conflict occurs when an atom with Weight < 0.5 contradicts an axiom with Weight = 1.0.
func DetectConflicts(context []Atom, axioms []Atom) int {
	conflicts := 0
	axiomMap := make(map[string]string) // predicate:object → "axiom"
	for _, a := range axioms {
		key := a.Predicate + ":" + a.Object
		axiomMap[key] = "axiom"
	}

	for _, atom := range context {
		if atom.Weight < 0.5 {
			continue // Low confidence atoms don't count as conflicts
		}
		key := atom.Predicate + ":" + atom.Object
		// A conflict is when context has a different value for same predicate
		if _, exists := axiomMap[key]; !exists {
			// Check if same predicate exists in axioms with different object
			for aKey := range axiomMap {
				if strings.HasPrefix(aKey, atom.Predicate+":") {
					conflicts++
					break
				}
			}
		}
	}
	return conflicts
}

// ClassifyTrustTier determines the trust tier from the decision source.
func ClassifyTrustTier(source string) TrustTier {
	switch source {
	case "kernel_axiom":
		return Tier0Kernel
	case "admin_config":
		return Tier1Admin
	case "ai_induced":
		return Tier2AI
	default:
		return Tier3User
	}
}

// IsSaliencyHigh returns true if the saliency signals indicate high priority.
func IsSaliencyHigh(saliency []string) bool {
	for _, s := range saliency {
		if s == "high_priority" {
			return true
		}
	}
	return false
}

// IsSaliencyKnown returns true if the saliency signals include a known pattern.
func IsSaliencyKnown(saliency []string) bool {
	for _, s := range saliency {
		if s == "known_pattern" {
			return true
		}
	}
	return false
}

// UpdateEntropy adjusts entropy based on validation results.
// New entropy = old entropy + (violations / total rules).
func UpdateEntropy(oldEntropy float64, violations, totalRules int) float64 {
	if totalRules == 0 {
		return oldEntropy
	}
	delta := float64(violations) / float64(totalRules)
	return math.Min(oldEntropy+delta, 1.0)
}
