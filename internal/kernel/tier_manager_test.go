package kernel

import (
	"context"
	"testing"
)

func TestMemoryTier(t *testing.T) {
	tests := []struct {
		name     string
		predicate string
		want     Tier
	}{
		{"phase_order is system", "phase_order", TierSystem},
		{"phase is system", "phase", TierSystem},
		{"agent_role is system", "agent_role", TierSystem},
		{"role_capability is system", "role_capability", TierSystem},
		{"pattern is system", "pattern", TierSystem},
		{"validation_rule is governance", "validation_rule", TierGovernance},
		{"validation_severity is governance", "validation_severity", TierGovernance},
		{"security_gate is governance", "security_gate", TierGovernance},
		{"action is playbook", "action", TierPlaybook},
		{"tool is playbook", "tool", TierPlaybook},
		{"workflow is playbook", "workflow", TierPlaybook},
		{"node_config is playbook", "node_config", TierPlaybook},
		{"unknown predicate is user", "unknown_predicate", TierUser},
		{"random is user", "random", TierUser},
		{"empty is user", "", TierUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MemoryTier(tt.predicate)
			if got != tt.want {
				t.Errorf("MemoryTier(%q) = %v, want %v", tt.predicate, got, tt.want)
			}
		})
	}
}

func TestTierConfigTotalBudget(t *testing.T) {
	cfg := TierConfig{
		SystemBudget:     2000,
		GovernanceBudget: 4000,
		PlaybookBudget:   8000,
		UserBudget:       6000,
	}

	got := cfg.TotalBudget()
	want := 20000
	if got != want {
		t.Errorf("TierConfig.TotalBudget() = %v, want %v", got, want)
	}
}

func TestDefaultTierConfigTotalBudget(t *testing.T) {
	got := DefaultTierConfig.TotalBudget()
	want := 20000
	if got != want {
		t.Errorf("DefaultTierConfig.TotalBudget() = %v, want %v", got, want)
	}
}

func TestClassifyAtom(t *testing.T) {
	tests := []struct {
		name  string
		atom  string
		want  Tier
	}{
		{"phase_order atom", "phase_order(1, observe)", TierSystem},
		{"action atom", "action(p1, exec)", TierPlaybook},
		{"validation_rule atom", "validation_rule(v1, high)", TierGovernance},
		{"unknown predicate", "foo(a, b)", TierUser},
		{"empty atom", "", TierUser},
		{"no parenthesis", "just_a_predicate", TierUser},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyAtom(tt.atom)
			if got != tt.want {
				t.Errorf("ClassifyAtom(%q) = %v, want %v", tt.atom, got, tt.want)
			}
		})
	}
}

func TestExtractPredicate(t *testing.T) {
	tests := []struct {
		name  string
		atom  string
		want  string
	}{
		{"simple predicate", "foo(a, b)", "foo"},
		{"predicate no args", "bar", ""},
		{"empty string", "", ""},
		{"nested parens", "action(foo(bar), baz)", "action"},
		{"spaces before paren", "my_predicate (a)", "my_predicate "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPredicate(tt.atom)
			if got != tt.want {
				t.Errorf("extractPredicate(%q) = %q, want %q", tt.atom, got, tt.want)
			}
		})
	}
}

func TestTierManagerGetBudget(t *testing.T) {
	mgr := NewTierManager()

	if got := mgr.GetBudget(TierSystem); got != 2000 {
		t.Errorf("GetBudget(TierSystem) = %v, want 2000", got)
	}
	if got := mgr.GetBudget(TierGovernance); got != 4000 {
		t.Errorf("GetBudget(TierGovernance) = %v, want 4000", got)
	}
	if got := mgr.GetBudget(TierPlaybook); got != 8000 {
		t.Errorf("GetBudget(TierPlaybook) = %v, want 8000", got)
	}
	if got := mgr.GetBudget(TierUser); got != 6000 {
		t.Errorf("GetBudget(TierUser) = %v, want 6000", got)
	}
	if got := mgr.GetBudget(Tier(100)); got != 0 {
		t.Errorf("GetBudget(Tier(100)) = %v, want 0", got)
	}
}

func TestTierManagerClassifyAtoms(t *testing.T) {
	mgr := NewTierManager()

	atoms := []string{
		"phase_order(1,2)",
		"action(a,b)",
		"validation_rule(x)",
		"tool(t)",
		"unknown(u)",
	}

	result := mgr.ClassifyAtoms(atoms)

	if len(result[TierSystem]) != 1 {
		t.Errorf("TierSystem count = %d, want 1", len(result[TierSystem]))
	}
	if len(result[TierGovernance]) != 1 {
		t.Errorf("TierGovernance count = %d, want 1", len(result[TierGovernance]))
	}
	if len(result[TierPlaybook]) != 2 {
		t.Errorf("TierPlaybook count = %d, want 2", len(result[TierPlaybook]))
	}
	if len(result[TierUser]) != 1 {
		t.Errorf("TierUser count = %d, want 1", len(result[TierUser]))
	}
}

func TestTierManagerIsWithinBudget(t *testing.T) {
	mgr := NewTierManager()

	tests := []struct {
		name   string
		stats  MemoryStats
		expect bool
	}{
		{
			"within all budgets",
			MemoryStats{SystemUsed: 1000, GovernanceUsed: 2000, PlaybookUsed: 4000, UserUsed: 3000},
			true,
		},
		{
			"exactly at system budget",
			MemoryStats{SystemUsed: 2000, GovernanceUsed: 0, PlaybookUsed: 0, UserUsed: 0},
			true,
		},
		{
			"exceeds system budget",
			MemoryStats{SystemUsed: 2001, GovernanceUsed: 0, PlaybookUsed: 0, UserUsed: 0},
			false,
		},
		{
			"exceeds playbook budget",
			MemoryStats{SystemUsed: 0, GovernanceUsed: 0, PlaybookUsed: 8001, UserUsed: 0},
			false,
		},
		{
			"all zero budgets",
			MemoryStats{SystemUsed: 0, GovernanceUsed: 0, PlaybookUsed: 0, UserUsed: 0},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.IsWithinBudget(tt.stats)
			if got != tt.expect {
				t.Errorf("IsWithinBudget(%+v) = %v, want %v", tt.stats, got, tt.expect)
			}
		})
	}
}

func TestTierManagerPruneRequired(t *testing.T) {
	mgr := NewTierManager()

	tests := []struct {
		name   string
		stats  MemoryStats
		expect map[Tier]int
	}{
		{
			"all within budget",
			MemoryStats{SystemUsed: 1000, GovernanceUsed: 2000, PlaybookUsed: 4000, UserUsed: 3000},
			map[Tier]int{},
		},
		{
			"exceeds system only",
			MemoryStats{SystemUsed: 2500, GovernanceUsed: 0, PlaybookUsed: 0, UserUsed: 0},
			map[Tier]int{TierSystem: 500},
		},
		{
			"exceeds multiple tiers",
			MemoryStats{SystemUsed: 2500, GovernanceUsed: 5000, PlaybookUsed: 9000, UserUsed: 7000},
			map[Tier]int{TierSystem: 500, TierGovernance: 1000, TierPlaybook: 1000, TierUser: 1000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.PruneRequired(tt.stats)
			if len(got) != len(tt.expect) {
				t.Errorf("PruneRequired(%+v) len = %d, want %d", tt.stats, len(got), len(tt.expect))
				return
			}
			for tier, wantTokens := range tt.expect {
				if got[tier] != wantTokens {
					t.Errorf("PruneRequired(%+v)[%v] = %d, want %d", tt.stats, tier, got[tier], wantTokens)
				}
			}
		})
	}
}

func TestMemoryStatsUsage(t *testing.T) {
	stats := MemoryStats{SystemUsed: 1000, GovernanceUsed: 2000, PlaybookUsed: 3000, UserUsed: 4000}
	if got := stats.Usage(); got != 10000 {
		t.Errorf("MemoryStats.Usage() = %v, want 10000", got)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name   string
		atoms  []string
		expect int
	}{
		{"empty", []string{}, 0},
		{"one atom", []string{"a"}, 4},
		{"three atoms", []string{"a", "b", "c"}, 12},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.atoms)
			if got != tt.expect {
				t.Errorf("EstimateTokens(%v) = %d, want %d", tt.atoms, got, tt.expect)
			}
		})
	}
}

func TestMemoryManagerShouldShave(t *testing.T) {
	tests := []struct {
		name   string
		stats  MemoryStats
		expect bool
	}{
		{
			"plenty of space",
			MemoryStats{SystemUsed: 1000, GovernanceUsed: 2000, PlaybookUsed: 3000, UserUsed: 4000},
			false,
		},
		{
			"at total budget",
			MemoryStats{SystemUsed: 2000, GovernanceUsed: 4000, PlaybookUsed: 8000, UserUsed: 6000},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := NewMemoryManager()
			mgr.WithAutoShave(AutoShaveConfig{
				Enabled:       true,
				Strategy:      ShaveUserFirst,
				MinFreeBudget: 500,
			})

			got := mgr.ShouldShave(context.Background(), tt.stats)
			if got != tt.expect {
				t.Errorf("ShouldShave(%+v) = %v, want %v", tt.stats, got, tt.expect)
			}
		})
	}
}

func TestMemoryManagerShouldShave_Disabled(t *testing.T) {
	mgr := NewMemoryManager()
	mgr.WithAutoShave(AutoShaveConfig{
		Enabled:       false,
		Strategy:      ShaveUserFirst,
		MinFreeBudget: 500,
	})

	if mgr.ShouldShave(context.Background(), MemoryStats{SystemUsed: 20000}) {
		t.Errorf("ShouldShave with Enabled=false should always return false")
	}
}

func TestMemoryManagerSelectAtomsForPruning(t *testing.T) {
	mgr := NewMemoryManager()

	atoms := map[Tier][]string{
		TierSystem:     {"sys1", "sys2"},
		TierGovernance: {"gov1", "gov2"},
		TierPlaybook:   {"play1", "play2"},
		TierUser:       {"user1", "user2"},
	}

	selected := mgr.SelectAtomsForPruning(atoms, ShaveUserFirst)
	if len(selected) != 6 {
		t.Errorf("ShaveUserFirst selected %d atoms, want 6", len(selected))
	}

	selected = mgr.SelectAtomsForPruning(atoms, ShaveLowWeightFirst)
	if len(selected) != 4 {
		t.Errorf("ShaveLowWeightFirst selected %d atoms, want 4", len(selected))
	}

	selected = mgr.SelectAtomsForPruning(atoms, ShaveOldestFirst)
	if len(selected) != 2 {
		t.Errorf("ShaveOldestFirst selected %d atoms, want 2", len(selected))
	}
}

func TestTierString(t *testing.T) {
	tests := []struct {
		tier  Tier
		wants string
	}{
		{TierSystem, "system"},
		{TierGovernance, "governance"},
		{TierPlaybook, "playbook"},
		{TierUser, "user"},
		{Tier(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.wants, func(t *testing.T) {
			if got := tt.tier.String(); got != tt.wants {
				t.Errorf("Tier.String() = %v, want %v", got, tt.wants)
			}
		})
	}
}