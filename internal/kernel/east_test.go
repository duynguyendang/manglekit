package kernel

import (
	"math"
	"testing"
)

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name     string
		conflicts int
		want     float64
	}{
		{"zero conflicts", 0, 0.0},
		{"one conflict", 1, 0.0},
		{"five conflicts", 5, 0.5},
		{"ten conflicts", 10, 1.0},
		{"beyond ten clamped", 15, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateEntropy(tt.conflicts)
			if got != tt.want {
				t.Errorf("CalculateEntropy(%d) = %v, want %v", tt.conflicts, got, tt.want)
			}
		})
	}
}

func TestCalculateActivity(t *testing.T) {
	tests := []struct {
		name         string
		accessCounts map[string]int
		want         float64
	}{
		{"empty map", map[string]int{}, 0.0},
		{"all hot atoms", map[string]int{"a": 10, "b": 8, "c": 6}, 1.0},
		{"all cold atoms", map[string]int{"a": 1, "b": 2, "c": 3}, 0.0},
		{"mixed atoms", map[string]int{"hot": 10, "cold": 1, "warm": 5}, 0.333},
		{"one hot one cold", map[string]int{"hot": 10, "cold": 1}, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateActivity(tt.accessCounts)
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("CalculateActivity(%v) = %v, want %v", tt.accessCounts, got, tt.want)
			}
		})
	}
}

func TestCalculateSaliency(t *testing.T) {
	cfg := DefaultEASTConfig

	tests := []struct {
		name     string
		input    string
		keywords []string
		want     float64
	}{
		{"empty input", "", []string{"a", "b"}, 0.0},
		{"empty keywords", "production critical", []string{}, 0.0},
		{"full match", "production critical security", []string{"production", "critical", "security"}, 1.0},
		{"no match", "hello world", []string{"production", "critical"}, 0.0},
		{"partial match", "production hello", []string{"production", "critical"}, 0.5},
		{"case insensitive", "PRODUCTION critical", []string{"production", "critical"}, 1.0},
		{"single keyword full match", "critical", []string{"critical"}, 1.0},
		{"single keyword no match", "hello", []string{"critical"}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSaliency(tt.input, tt.keywords)
			if got != tt.want {
				t.Errorf("CalculateSaliency(%q, %v) = %v, want %v", tt.input, tt.keywords, got, tt.want)
			}
		})
	}

	_ = cfg
}

func TestDetermineTrust(t *testing.T) {
	tests := []struct {
		name          string
		decisionSource string
		want          TierLevel
	}{
		{"kernel_axiom", "kernel_axiom", T0_Axiom},
		{"admin_config", "admin_config", T1_Governance},
		{"ai_induced", "ai_induced", T2_Playbook},
		{"user_input", "user_input", T3_User},
		{"unknown", "unknown_source", TierUnknown},
		{"empty", "", TierUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineTrust(tt.decisionSource)
			if got != tt.want {
				t.Errorf("DetermineTrust(%q) = %v, want %v", tt.decisionSource, got, tt.want)
			}
		})
	}
}

func TestDeterminePath(t *testing.T) {
	cfg := DefaultEASTConfig

	tests := []struct {
		name    string
		metrics EASTMetrics
		want    PathType
	}{
		{
			"fast path: low entropy + high trust",
			EASTMetrics{Entropy: 0.3, Trust: T2_Playbook, Conflicts: 0},
			FastPath,
		},
		{
			"slow path: unknown trust",
			EASTMetrics{Entropy: 0.3, Trust: TierUnknown, Conflicts: 1},
			SlowPath,
		},
		{
			"fast path: moderate entropy + high trust",
			EASTMetrics{Entropy: 0.5, Trust: T2_Playbook, Conflicts: 4},
			FastPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeterminePath(tt.metrics, cfg)
			if got != tt.want {
				t.Errorf("DeterminePath(%+v, cfg) = %v, want %v", tt.metrics, got, tt.want)
			}
		})
	}
}

func TestCalculateEAST(t *testing.T) {
	cfg := EASTConfig{
		SaliencyKeywords: []string{"production", "critical"},
	}

	frame := &EASTFrame{
		ConflictCount:  5,
		AccessCounts:  map[string]int{"a": 10, "b": 1},
		Input:          "production critical",
		DecisionSource: "kernel_axiom",
	}

	metrics := CalculateEAST(frame, cfg)

	if metrics.Entropy != 0.5 {
		t.Errorf("Entropy = %v, want 0.5", metrics.Entropy)
	}
	if metrics.Activity != 0.5 {
		t.Errorf("Activity = %v, want 0.5", metrics.Activity)
	}
	if metrics.Saliency != 1.0 {
		t.Errorf("Saliency = %v, want 1.0", metrics.Saliency)
	}
	if metrics.Trust != T0_Axiom {
		t.Errorf("Trust = %v, want T0_Axiom", metrics.Trust)
	}
	if metrics.Conflicts != 5 {
		t.Errorf("Conflicts = %v, want 5", metrics.Conflicts)
	}
}

func TestCalculator(t *testing.T) {
	calc := NewEASTCalculator()

	frame := &EASTFrame{
		ConflictCount:  3,
		AccessCounts:   map[string]int{"x": 10, "y": 2, "z": 1},
		Input:          "critical security",
		DecisionSource: "ai_induced",
	}

	metrics := calc.Calculate(frame)
	if math.Abs(metrics.Entropy-0.3) > 0.001 {
		t.Errorf("Calculate() Entropy = %v, want 0.3", metrics.Entropy)
	}

	path := calc.GetPath(metrics)
	if path != FastPath {
		t.Errorf("GetPath() = %v, want FastPath", path)
	}

	fullMetrics, fullPath := calc.FullAnalysis(frame)
	if fullMetrics.Entropy != 0.3 {
		t.Errorf("FullAnalysis() Entropy = %v, want 0.3", fullMetrics.Entropy)
	}
	if fullPath != FastPath {
		t.Errorf("FullAnalysis() Path = %v, want FastPath", fullPath)
	}
}

func TestCalculatorWithConfig(t *testing.T) {
	cfg := EASTConfig{
		EntropyThreshold: 0.3,
		ChaosThreshold:   0.5,
		SaliencyKeywords: []string{"critical"},
	}

	calc := NewEASTCalculator().WithConfig(cfg)
	if calc.config.EntropyThreshold != 0.3 {
		t.Errorf("EntropyThreshold = %v, want 0.3", calc.config.EntropyThreshold)
	}

	frame := &EASTFrame{
		ConflictCount:  5,
		AccessCounts:  map[string]int{"a": 1},
		Input:          "critical",
		DecisionSource: "user_input",
	}

	metrics, _ := calc.FullAnalysis(frame)
	if metrics.Trust != T3_User {
		t.Errorf("Trust = %v, want T3_User", metrics.Trust)
	}
}

func TestTierLevelString(t *testing.T) {
	tests := []struct {
		tier  TierLevel
		wants string
	}{
		{T0_Axiom, "TIER_0"},
		{T1_Governance, "TIER_1"},
		{T2_Playbook, "TIER_2"},
		{T3_User, "TIER_3"},
		{TierUnknown, "TIER_UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.wants, func(t *testing.T) {
			if got := tt.tier.String(); got != tt.wants {
				t.Errorf("TierLevel.String() = %v, want %v", got, tt.wants)
			}
		})
	}
}

func TestPathTypeString(t *testing.T) {
	tests := []struct {
		path  PathType
		wants string
	}{
		{FastPath, "fast"},
		{SlowPath, "slow"},
		{PathUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.wants, func(t *testing.T) {
			if got := tt.path.String(); got != tt.wants {
				t.Errorf("PathType.String() = %v, want %v", got, tt.wants)
			}
		})
	}
}

func TestPruneCandidates(t *testing.T) {
	atoms := []AtomScore{
		{Subject: "a", Predicate: "b", Object: "c", Weight: 0.3},
		{Subject: "d", Predicate: "e", Object: "f", Weight: 0.8},
		{Subject: "g", Predicate: "h", Object: "i", Weight: 0.1},
		{Subject: "j", Predicate: "k", Object: "l", Weight: 0.5},
	}

	got := PruneCandidates(atoms, 0.4)
	if len(got) != 2 {
		t.Errorf("PruneCandidates(len=4, threshold=0.4) returned %d, want 2", len(got))
	}

	for _, a := range got {
		if a.Weight >= 0.4 {
			t.Errorf("PruneCandidates included atom with weight %v >= 0.4", a.Weight)
		}
	}
}

func TestFloatHelpers(t *testing.T) {
	if !floatEquals(1.0, 1.0, 0.001) {
		t.Errorf("floatEquals(1.0, 1.0, 0.001) = false, want true")
	}
	if floatEquals(1.0, 1.1, 0.001) {
		t.Errorf("floatEquals(1.0, 1.1, 0.001) = true, want false")
	}
	if !floatGreater(2.0, 1.0) {
		t.Errorf("floatGreater(2.0, 1.0) = false, want true")
	}
	if floatGreater(1.0, 2.0) {
		t.Errorf("floatGreater(1.0, 2.0) = true, want false")
	}
	if !floatLess(1.0, 2.0) {
		t.Errorf("floatLess(1.0, 2.0) = false, want true")
	}
	if floatLess(2.0, 1.0) {
		t.Errorf("floatLess(2.0, 1.0) = true, want false")
	}
}