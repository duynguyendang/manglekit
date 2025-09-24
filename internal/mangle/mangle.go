// Package mangle implements the rule and fact engine for pre- and post-processing.
// It provides a basic rule processor that loads rules and facts, and applies them for query expansion and content filtering.

package mangle

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"ndduy.dev/manglekit/internal/types"
)

// Rule represents a simple rule structure for processing.
type Rule struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"` // "preprocess" or "postprocess"
	Pattern     string            `json:"pattern"`
	Action      string            `json:"action"`
	Conditions  map[string]string `json:"conditions"`
	Expansions  []string          `json:"expansions,omitempty"`
	Filters     map[string]string `json:"filters,omitempty"`
	Priority    int               `json:"priority"`
	Description string            `json:"description"`
}

// Fact represents a fact from the facts.json.
type Fact struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Value    string                 `json:"value"`
	Metadata map[string]interface{} `json:"metadata"`
}

// MangleImpl is the concrete implementation of the Processor interface.
type MangleImpl struct {
	rules map[string][]Rule
	facts map[string]Fact
}

// NewMangle creates a new Mangle engine by loading rules and facts from files.
func NewMangle(rulesFile, factsFile string) types.Processor {
	m := &MangleImpl{
		rules: make(map[string][]Rule),
		facts: make(map[string]Fact),
	}

	// Load rules
	if err := m.loadRules(rulesFile); err != nil {
		fmt.Printf("Warning: Failed to load rules from %s: %v\n", rulesFile, err)
	}

	// Load facts
	if err := m.loadFacts(factsFile); err != nil {
		fmt.Printf("Warning: Failed to load facts from %s: %v\n", factsFile, err)
	}

	fmt.Printf("Mangle engine initialized: %d rules, %d facts\n", len(m.rules["preprocess"])+len(m.rules["postprocess"]), len(m.facts))
	return m
}

// loadRules loads rules from the rules file (assuming JSON format for simplicity).
func (m *MangleImpl) loadRules(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read rules file: %w", err)
	}

	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		// If not JSON, try simple text format
		return m.parseTextRules(string(data))
	}

	for _, rule := range rules {
		m.rules[rule.Type] = append(m.rules[rule.Type], rule)
	}

	return nil
}

// parseTextRules parses simple text-based rules (one per line: type:pattern->action).
func (m *MangleImpl) parseTextRules(content string) error {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 2 {
			continue
		}

		ruleType := strings.TrimSpace(parts[0])
		rest := strings.TrimSpace(strings.Join(parts[1:], ":"))
		patternEnd := strings.Index(rest, "->")
		if patternEnd == -1 {
			continue
		}

		pattern := strings.TrimSpace(rest[:patternEnd])
		action := strings.TrimSpace(rest[patternEnd+2:])

		rule := Rule{
			ID:       fmt.Sprintf("rule_%d", len(m.rules[ruleType])),
			Type:     ruleType,
			Pattern:  pattern,
			Action:   action,
			Priority: 1,
		}

		m.rules[ruleType] = append(m.rules[ruleType], rule)
	}

	return nil
}

// loadFacts loads facts from JSON file.
func (m *MangleImpl) loadFacts(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read facts file: %w", err)
	}

	var facts []Fact
	if err := json.Unmarshal(data, &facts); err != nil {
		return fmt.Errorf("failed to parse facts JSON: %w", err)
	}

	for _, fact := range facts {
		m.facts[fact.ID] = fact
	}

	return nil
}

// PreProcess normalizes, applies constraints, and expands the query using rules.
func (m *MangleImpl) PreProcess(input *types.QueryInput) (*types.ExpandedQuery, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}

	normalized := strings.ToLower(input.Query)
	expansion := []string{}
	filters := map[string]string{"access_level": "internal"}

	// Apply preprocess rules
	preRules := m.rules["preprocess"]
	for _, rule := range preRules {
		if strings.Contains(normalized, rule.Pattern) {
			// Check conditions
			if m.matchesConditions(rule.Conditions, input.UserContext) {
				expansion = append(expansion, rule.Expansions...)
				for k, v := range rule.Filters {
					filters[k] = v
				}
				normalized = strings.ReplaceAll(normalized, rule.Pattern, rule.Action)
			}
		}
	}

	// Incorporate relevant facts
	for _, fact := range m.facts {
		if strings.Contains(normalized, fact.Value) {
			if meta, ok := fact.Metadata["filter"]; ok {
				for k, v := range meta.(map[string]interface{}) {
					filters[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	return &types.ExpandedQuery{
		NormalizedQuery: normalized,
		ExpansionTerms:  expansion,
		Filters:         filters,
		Explanation:     fmt.Sprintf("Applied %d preprocess rules and %d facts", len(preRules), len(m.facts)),
	}, nil
}

// PostProcess validates, redacts, and annotates chunks using rules.
func (m *MangleImpl) PostProcess(chunks []*types.Chunk, ctx *types.Context) ([]*types.Chunk, *[]types.Explanation) {
	var vetted []*types.Chunk
	var expls []types.Explanation

	postRules := m.rules["postprocess"]
	for _, chunk := range chunks {
		kept := true
		currentExpls := []types.Explanation{}

		// Apply postprocess rules
		for _, rule := range postRules {
			if strings.Contains(strings.ToLower(chunk.Text), rule.Pattern) {
				if !m.matchesConditions(rule.Conditions, ctx.UserContext) {
					kept = false
					currentExpls = append(currentExpls, types.Explanation{
						Type:      "filter",
						Rule:      rule.ID,
						Action:    "discarded",
						Reason:    rule.Description,
						Timestamp: time.Now(),
					})
					break
				} else {
					// Apply action (e.g., redact)
					if rule.Action == "redact" {
						chunk.Text = strings.ReplaceAll(chunk.Text, rule.Pattern, "[REDACTED]")
					}
					currentExpls = append(currentExpls, types.Explanation{
						Type:      "modification",
						Rule:      rule.ID,
						Action:    rule.Action,
						Reason:    rule.Description,
						Timestamp: time.Now(),
					})
				}
			}
		}

		// Check against facts for additional filtering
		for _, fact := range m.facts {
			if strings.Contains(strings.ToLower(chunk.Text), fact.Value) {
				if access, ok := fact.Metadata["access"].(string); ok && access == "restricted" {
					if _, userOk := ctx.UserContext["role"]; !userOk || ctx.UserContext["role"] != "admin" {
						kept = false
						currentExpls = append(currentExpls, types.Explanation{
							Type:      "fact_filter",
							Rule:      fact.ID,
							Action:    "discarded",
							Reason:    "Restricted content access",
							Timestamp: time.Now(),
						})
						break
					}
				}
			}
		}

		if kept {
			vetted = append(vetted, chunk)
		}

		expls = append(expls, currentExpls...)
	}

	return vetted, &expls
}

// matchesConditions checks if the conditions map matches the context.
func (m *MangleImpl) matchesConditions(conditions map[string]string, context map[string]interface{}) bool {
	for key, requiredValue := range conditions {
		if actualValue, ok := context[key]; ok {
			if fmt.Sprintf("%v", actualValue) != requiredValue {
				return false
			}
		} else {
			return false
		}
	}
	return true
}
