package kernel

import "context"

type Tier int

const (
	TierSystem Tier = iota
	TierGovernance
	TierPlaybook
	TierUser
)

func (t Tier) String() string {
	switch t {
	case TierSystem:
		return "system"
	case TierGovernance:
		return "governance"
	case TierPlaybook:
		return "playbook"
	case TierUser:
		return "user"
	default:
		return "unknown"
	}
}

// TierConfig holds memory budget per tier
type TierConfig struct {
	SystemBudget     int // tokens for T0 (FP32 precision)
	GovernanceBudget int // tokens for T1 (FP32 precision)
	PlaybookBudget   int // tokens for T2 (INT8 precision)
	UserBudget       int // tokens for T3 (INT8 precision)
}

var DefaultTierConfig = TierConfig{
	SystemBudget:     2000, // Critical axioms
	GovernanceBudget: 4000, // Policy rules
	PlaybookBudget:   8000, // AI-induced rules
	UserBudget:       6000, // User input
}

// TotalBudget returns the total memory budget
func (c TierConfig) TotalBudget() int {
	return c.SystemBudget + c.GovernanceBudget + c.PlaybookBudget + c.UserBudget
}

// MemoryTier determines which tier an atom belongs to
func MemoryTier(predicate string) Tier {
	// System-level predicates (T0)
	systemPredicates := []string{
		"phase_order",
		"phase",
		"agent_role",
		"role_capability",
		"pattern",
	}

	for _, p := range systemPredicates {
		if predicate == p {
			return TierSystem
		}
	}

	// Governance predicates (T1)
	governancePredicates := []string{
		"validation_rule",
		"validation_severity",
		"security_gate",
	}

	for _, p := range governancePredicates {
		if predicate == p {
			return TierGovernance
		}
	}

	// Playbook predicates (T2)
	playbookPredicates := []string{
		"action",
		"tool",
		"workflow",
		"node_config",
	}

	for _, p := range playbookPredicates {
		if predicate == p {
			return TierPlaybook
		}
	}

	// Default to user tier (T3)
	return TierUser
}

// TierManager handles memory allocation per tier
type TierManager struct {
	config TierConfig
}

// NewTierManager creates a tier manager with default config
func NewTierManager() *TierManager {
	return &TierManager{config: DefaultTierConfig}
}

// WithConfig sets custom tier configuration
func (m *TierManager) WithConfig(cfg TierConfig) *TierManager {
	m.config = cfg
	return m
}

// GetBudget returns the budget for a given tier
func (m *TierManager) GetBudget(tier Tier) int {
	switch tier {
	case TierSystem:
		return m.config.SystemBudget
	case TierGovernance:
		return m.config.GovernanceBudget
	case TierPlaybook:
		return m.config.PlaybookBudget
	case TierUser:
		return m.config.UserBudget
	default:
		return 0
	}
}

// ClassifyAtoms distributes atoms into tiers based on predicates
func (m *TierManager) ClassifyAtoms(atoms []string) map[Tier][]string {
	result := map[Tier][]string{
		TierSystem:     {},
		TierGovernance: {},
		TierPlaybook:   {},
		TierUser:       {},
	}

	for _, atom := range atoms {
		tier := ClassifyAtom(atom)
		result[tier] = append(result[tier], atom)
	}

	return result
}

// ClassifyAtom determines the tier for a single atom
func ClassifyAtom(atom string) Tier {
	// Extract predicate from atom (simplified parsing)
	// Format: predicate(arg1, arg2, ...)
	predicate := extractPredicate(atom)
	if predicate == "" {
		return TierUser
	}
	return MemoryTier(predicate)
}

// extractPredicate pulls the predicate name from an atom string
func extractPredicate(atom string) string {
	// Find the opening parenthesis
	for i := 0; i < len(atom); i++ {
		if atom[i] == '(' {
			return atom[:i]
		}
	}
	return ""
}

// MemoryStats holds current memory usage per tier
type MemoryStats struct {
	SystemUsed     int
	GovernanceUsed int
	PlaybookUsed   int
	UserUsed       int
}

// Usage returns total tokens used
func (s MemoryStats) Usage() int {
	return s.SystemUsed + s.GovernanceUsed + s.PlaybookUsed + s.UserUsed
}

// IsWithinBudget checks if current usage is within budget
func (m *TierManager) IsWithinBudget(stats MemoryStats) bool {
	return stats.SystemUsed <= m.config.SystemBudget &&
		stats.GovernanceUsed <= m.config.GovernanceBudget &&
		stats.PlaybookUsed <= m.config.PlaybookBudget &&
		stats.UserUsed <= m.config.UserBudget
}

// PruneRequired returns how many tokens need to be pruned per tier
func (m *TierManager) PruneRequired(stats MemoryStats) map[Tier]int {
	result := map[Tier]int{}

	if stats.SystemUsed > m.config.SystemBudget {
		result[TierSystem] = stats.SystemUsed - m.config.SystemBudget
	}
	if stats.GovernanceUsed > m.config.GovernanceBudget {
		result[TierGovernance] = stats.GovernanceUsed - m.config.GovernanceBudget
	}
	if stats.PlaybookUsed > m.config.PlaybookBudget {
		result[TierPlaybook] = stats.PlaybookUsed - m.config.PlaybookBudget
	}
	if stats.UserUsed > m.config.UserBudget {
		result[TierUser] = stats.UserUsed - m.config.UserBudget
	}

	return result
}

// ShaveStrategy determines which atoms to prune first
type ShaveStrategy int

const (
	ShaveUserFirst      ShaveStrategy = iota // Prune T3 before T2 before T1 before T0
	ShaveLowWeightFirst                      // Prune by atom weight regardless of tier
	ShaveOldestFirst                         // Prune by timestamp
)

// AutoShaveConfig holds configuration for automatic context shaving
type AutoShaveConfig struct {
	Enabled       bool
	Strategy      ShaveStrategy
	MinFreeBudget int // Minimum free space to maintain
}

// DefaultAutoShaveConfig returns the default auto-shave configuration
func DefaultAutoShaveConfig() AutoShaveConfig {
	return AutoShaveConfig{
		Enabled:       true,
		Strategy:      ShaveUserFirst,
		MinFreeBudget: 500,
	}
}

// MemoryManager handles automatic context shaving
type MemoryManager struct {
	tierManager *TierManager
	autoShave   AutoShaveConfig
}

// NewMemoryManager creates a memory manager with defaults
func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		tierManager: NewTierManager(),
		autoShave:   DefaultAutoShaveConfig(),
	}
}

// WithTierConfig sets custom tier configuration
func (m *MemoryManager) WithTierConfig(cfg TierConfig) *MemoryManager {
	m.tierManager.WithConfig(cfg)
	return m
}

// WithAutoShave configures automatic shaving behavior
func (m *MemoryManager) WithAutoShave(cfg AutoShaveConfig) *MemoryManager {
	m.autoShave = cfg
	return m
}

// EstimateTokens estimates token count for atoms (simplified)
func EstimateTokens(atoms []string) int {
	// Rough estimate: average 4 tokens per atom
	return len(atoms) * 4
}

// ShouldShave determines if context needs pruning
func (m *MemoryManager) ShouldShave(ctx context.Context, stats MemoryStats) bool {
	if !m.autoShave.Enabled {
		return false
	}

	totalBudget := m.tierManager.config.TotalBudget()
	currentUsage := stats.Usage()

	// Need to shave if we don't have MinFreeBudget left
	return (totalBudget - currentUsage) < m.autoShave.MinFreeBudget
}

// SelectAtomsForPruning selects atoms to prune based on strategy
func (m *MemoryManager) SelectAtomsForPruning(atoms map[Tier][]string, strategy ShaveStrategy) []string {
	var selected []string

	switch strategy {
	case ShaveUserFirst:
		// Prune from lowest tier first
		selected = append(selected, atoms[TierUser]...)
		selected = append(selected, atoms[TierPlaybook]...)
		selected = append(selected, atoms[TierGovernance]...)
		// System tier is never pruned (attention sink)
	case ShaveLowWeightFirst:
		// This would require atom weights - simplified here
		selected = append(selected, atoms[TierUser]...)
		selected = append(selected, atoms[TierPlaybook]...)
	case ShaveOldestFirst:
		// This would require timestamps - simplified here
		selected = append(selected, atoms[TierUser]...)
	}

	return selected
}
