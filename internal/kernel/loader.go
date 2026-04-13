package kernel

import (
	"embed"
	"fmt"
	"strings"
)

var (
	//go:embed embedded/*.dl
	embeddedFS embed.FS
)

// KernelRules contains the embedded framework rules
type KernelRules struct {
	OODA       string
	Agents     string
	Patterns   string
	Validation string
}

// LoadKernel returns all embedded kernel rules concatenated
func LoadKernel() (string, error) {
	rules := KernelRules{}

	// Load OODA phases
	ooda, err := embeddedFS.ReadFile("embedded/ooda_phases.dl")
	if err != nil {
		return "", fmt.Errorf("failed to load ooda_phases.dl: %w", err)
	}
	rules.OODA = string(ooda)

	// Load agents
	agents, err := embeddedFS.ReadFile("embedded/agents.dl")
	if err != nil {
		return "", fmt.Errorf("failed to load agents.dl: %w", err)
	}
	rules.Agents = string(agents)

	// Load patterns
	patterns, err := embeddedFS.ReadFile("embedded/patterns.dl")
	if err != nil {
		return "", fmt.Errorf("failed to load patterns.dl: %w", err)
	}
	rules.Patterns = string(patterns)

	// Load validation
	validation, err := embeddedFS.ReadFile("embedded/validation.dl")
	if err != nil {
		return "", fmt.Errorf("failed to load validation.dl: %w", err)
	}
	rules.Validation = string(validation)

	return rules.Combine(), nil
}

// Combine concatenates all kernel rules
func (k *KernelRules) Combine() string {
	var parts []string
	parts = append(parts, "% ==========================================")
	parts = append(parts, "% MANGLEKIT KERNEL RULES (Embedded)")
	parts = append(parts, "% ==========================================")
	parts = append(parts, "")
	parts = append(parts, "% --- OODA Phases ---")
	parts = append(parts, k.OODA)
	parts = append(parts, "")
	parts = append(parts, "% --- Agent Registry ---")
	parts = append(parts, k.Agents)
	parts = append(parts, "")
	parts = append(parts, "% --- Patterns ---")
	parts = append(parts, k.Patterns)
	parts = append(parts, "")
	parts = append(parts, "% --- Validation Rules ---")
	parts = append(parts, k.Validation)

	return strings.Join(parts, "\n")
}

// Profile represents a security/business rules profile from Nexus
type Profile struct {
	Name     string
	Rules    string
	Metadata map[string]string
}

// Loader handles kernel + profile rule loading
type Loader struct {
	kernel string
}

// NewLoader creates a new kernel loader
func NewLoader() (*Loader, error) {
	kernel, err := LoadKernel()
	if err != nil {
		return nil, err
	}
	return &Loader{kernel: kernel}, nil
}

// GetKernel returns the embedded kernel rules
func (l *Loader) GetKernel() string {
	return l.kernel
}

// Merge combines kernel rules with profile rules
// Profile rules override kernel rules (last predicate wins)
func (l *Loader) Merge(profile Profile) (string, error) {
	if profile.Rules == "" {
		return l.kernel, nil
	}

	// Simple merge: kernel + profile
	// In practice, the engine will evaluate both and later facts override earlier
	return l.kernel + "\n\n% --- Profile: " + profile.Name + " ---\n" + profile.Rules, nil
}

// DefaultProfile returns a permissive default profile
func DefaultProfile() Profile {
	return Profile{
		Name: "default",
		Rules: `
% Default permissive rules - no restrictions
allow(_, _, _) :- true.
`,
		Metadata: map[string]string{
			"description": "Permissive default - allows all actions",
		},
	}
}
