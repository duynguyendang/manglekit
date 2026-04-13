package kernel

import (
	"math"
)

type TierLevel int

const (
	TierUnknown TierLevel = iota
	T0_Axiom
	T1_Governance
	T2_Playbook
	T3_User
)

func (t TierLevel) String() string {
	switch t {
	case T0_Axiom:
		return "TIER_0"
	case T1_Governance:
		return "TIER_1"
	case T2_Playbook:
		return "TIER_2"
	case T3_User:
		return "TIER_3"
	default:
		return "TIER_UNKNOWN"
	}
}

// EASTMetrics contains the four cognitive steering metrics
type EASTMetrics struct {
	Entropy   float64
	Activity  float64
	Saliency  float64
	Trust     TierLevel
	Conflicts int
}

// PathType determines whether to use fast or slow cognitive path
type PathType int

const (
	PathUnknown PathType = iota
	FastPath
	SlowPath
)

func (p PathType) String() string {
	switch p {
	case FastPath:
		return "fast"
	case SlowPath:
		return "slow"
	default:
		return "unknown"
	}
}

// Config holds EAST calculation configuration
type EASTConfig struct {
	EntropyThreshold float64
	ChaosThreshold   float64
	SaliencyKeywords []string
}

var DefaultEASTConfig = EASTConfig{
	EntropyThreshold: 0.7,
	ChaosThreshold:   0.9,
	SaliencyKeywords: []string{"production", "critical", "security"},
}

// CalculateEntropy computes entropy based on conflict count
// High entropy = more contradictions in reasoning frame
func CalculateEntropy(conflicts int) float64 {
	if conflicts <= 1 {
		return 0.0
	}
	// Normalize to [0, 1] range
	entropy := float64(conflicts) / 10.0
	if entropy > 1.0 {
		entropy = 1.0
	}
	return entropy
}

// CalculateActivity computes activity based on atom access patterns
// Hot atoms are frequently accessed, cold atoms are rarely used
func CalculateActivity(accessCounts map[string]int) float64 {
	if len(accessCounts) == 0 {
		return 0.0
	}

	var totalAccess, hotCount int
	for _, count := range accessCounts {
		totalAccess += count
		if count > 5 {
			hotCount++
		}
	}

	if totalAccess == 0 {
		return 0.0
	}

	// Activity ratio: hot atoms / total atoms
	return float64(hotCount) / float64(len(accessCounts))
}

// CalculateSaliency determines if input contains high-priority keywords
func CalculateSaliency(input string, keywords []string) float64 {
	if input == "" || len(keywords) == 0 {
		return 0.0
	}

	lowerInput := toLower(input)
	matchCount := 0

	for _, kw := range keywords {
		if contains(lowerInput, toLower(kw)) {
			matchCount++
		}
	}

	return float64(matchCount) / float64(len(keywords))
}

// DetermineTrust assigns a tier level based on decision source
func DetermineTrust(decisionSource string) TierLevel {
	switch decisionSource {
	case "kernel_axiom":
		return T0_Axiom
	case "admin_config":
		return T1_Governance
	case "ai_induced":
		return T2_Playbook
	case "user_input":
		return T3_User
	default:
		return TierUnknown
	}
}

// DeterminePath decides fast vs slow cognitive path based on EAST metrics
func DeterminePath(metrics EASTMetrics, cfg EASTConfig) PathType {
	// Fast path: low entropy + high trust + known saliency
	if metrics.Entropy < cfg.EntropyThreshold && metrics.Trust >= T2_Playbook {
		return FastPath
	}

	// Slow path: high entropy or low trust
	if metrics.Entropy >= cfg.ChaosThreshold {
		return SlowPath
	}

	if metrics.Trust <= T3_User {
		return SlowPath
	}

	return FastPath
}

// CalculateEAST computes all four metrics and returns EASTMetrics
func CalculateEAST(frame *EASTFrame, cfg EASTConfig) EASTMetrics {
	metrics := EASTMetrics{
		Conflicts: frame.ConflictCount,
	}

	// Calculate entropy
	metrics.Entropy = CalculateEntropy(frame.ConflictCount)

	// Calculate activity
	metrics.Activity = CalculateActivity(frame.AccessCounts)

	// Calculate saliency
	metrics.Saliency = CalculateSaliency(frame.Input, cfg.SaliencyKeywords)

	// Determine trust
	metrics.Trust = DetermineTrust(frame.DecisionSource)

	return metrics
}

// EASTFrame contains input data for EAST calculation
type EASTFrame struct {
	ConflictCount  int
	AccessCounts   map[string]int
	Input          string
	DecisionSource string
}

// Calculator provides EAST calculation as a service
type Calculator struct {
	config EASTConfig
}

// NewEASTCalculator creates a new EAST calculator with default config
func NewEASTCalculator() *Calculator {
	return &Calculator{config: DefaultEASTConfig}
}

// WithConfig sets custom configuration
func (c *Calculator) WithConfig(cfg EASTConfig) *Calculator {
	c.config = cfg
	return c
}

// Calculate computes EAST metrics for a frame
func (c *Calculator) Calculate(frame *EASTFrame) EASTMetrics {
	return CalculateEAST(frame, c.config)
}

// GetPath returns the recommended cognitive path
func (c *Calculator) GetPath(metrics EASTMetrics) PathType {
	return DeterminePath(metrics, c.config)
}

// FullAnalysis returns both metrics and recommended path
func (c *Calculator) FullAnalysis(frame *EASTFrame) (EASTMetrics, PathType) {
	metrics := c.Calculate(frame)
	path := c.GetPath(metrics)
	return metrics, path
}

// Built-in predicates for Datalog engine

// calculate_entropy_predicate is used to register calculate_entropy in the engine
func CalculateEntropyPredicate(ctx interface{}, inputs []interface{}) ([][]interface{}, error) {
	if len(inputs) < 1 {
		return nil, nil
	}

	conflicts, ok := toInt(inputs[0])
	if !ok {
		return nil, nil
	}

	entropy := CalculateEntropy(conflicts)
	return [][]interface{}{{entropy}}, nil
}

// calculate_activity_predicate computes activity from access counts
func CalculateActivityPredicate(ctx interface{}, inputs []interface{}) ([][]interface{}, error) {
	if len(inputs) < 1 {
		return nil, nil
	}

	accessMap, ok := inputs[0].(map[string]int)
	if !ok {
		return nil, nil
	}

	activity := CalculateActivity(accessMap)
	return [][]interface{}{{activity}}, nil
}

// calculate_saliency_predicate checks for keyword matches
func CalculateSaliencyPredicate(ctx interface{}, inputs []interface{}) ([][]interface{}, error) {
	if len(inputs) < 2 {
		return nil, nil
	}

	input, ok := toString(inputs[0])
	if !ok {
		return nil, nil
	}

	keywordsRaw, ok := inputs[1].([]string)
	if !ok {
		return nil, nil
	}

	saliency := CalculateSaliency(input, keywordsRaw)
	return [][]interface{}{{saliency}}, nil
}

// Helper functions

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func contains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

func toString(v interface{}) (string, bool) {
	if s, ok := v.(string); ok {
		return s, true
	}
	return "", false
}

// PruneCandidates returns atoms that should be pruned based on coldness and low weight
func PruneCandidates(atoms []AtomScore, threshold float64) []AtomScore {
	var candidates []AtomScore
	for _, atom := range atoms {
		if atom.Weight < threshold {
			candidates = append(candidates, atom)
		}
	}
	return candidates
}

// AtomScore holds an atom with its computed weight
type AtomScore struct {
	Subject   string
	Predicate string
	Object    string
	Weight    float64
}

// Math helpers for float comparison
func floatEquals(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func floatGreater(a, b float64) bool {
	return a > b
}

func floatLess(a, b float64) bool {
	return a < b
}
