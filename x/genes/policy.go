package genes

import (
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
)

// CompileOptions controls the promotion discipline applied to genes.
type CompileOptions struct {
	// AllowHardTiers permits genes carrying blocking tiers (T0/T1) to be
	// compiled. Without it, compilation refuses them — the explicit opt-in
	// models human review promoting learned rules into governance.
	AllowHardTiers bool
}

// CompileOption mutates CompileOptions.
type CompileOption func(*CompileOptions)

// WithAllowHardTiers allows T0/T1 genes through the compiler.
func WithAllowHardTiers() CompileOption {
	return func(o *CompileOptions) { o.AllowHardTiers = true }
}

// Compile renders genes into a single self-contained Datalog program that
// can be loaded through the policy channel (sdk.Client.LoadPolicy /
// ReloadPolicySource). Each gene is signature-verified first; hard tiers
// require WithAllowHardTiers (see CompileOptions).
//
// Compile does NOT rewrite rule text — gene rules carry their own halt/3
// tiers and must be consistent with the declared gene tier. A mismatch
// (e.g. a T2 gene whose rules emit halt(..., "T1")) is rejected: the
// compiler refuses to be the place where tiers silently disagree.
func Compile(genes []Gene, opts ...CompileOption) (string, error) {
	cfg := CompileOptions{}
	for _, o := range opts {
		o(&cfg)
	}

	if len(genes) == 0 {
		return "", fmt.Errorf("x/genes: compile called with no genes")
	}

	var b strings.Builder
	b.WriteString("% ---- x/genes compiled policy ----\n")
	for _, g := range genes {
		if err := g.Verify(); err != nil {
			return "", err
		}
		if !validTier(g.Tier) {
			return "", fmt.Errorf("x/genes: gene %q has invalid tier %q", g.Name, g.Tier)
		}
		if isHardTier(g.Tier) && !cfg.AllowHardTiers {
			return "", fmt.Errorf("x/genes: gene %q carries hard tier %s (can block); "+
				"applying learned blocking rules requires WithAllowHardTiers() — promote through human review first",
				g.Name, g.Tier)
		}
		if err := checkTierConsistency(g); err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "\n%% gene: %s | tier: %s | source: %s\n", g.Name, g.Tier, g.Source)
		rules := strings.TrimSpace(g.Rules)
		b.WriteString(rules)
		b.WriteString("\n")
	}
	return b.String(), nil
}

// checkTierConsistency rejects rule bodies whose halt/3 tiers disagree with
// the declared gene tier, so signing a T2 gene guarantees advisory semantics.
func checkTierConsistency(g Gene) error {
	declared := string(g.Tier)
	lower := strings.ToLower(g.Rules)
	if !strings.Contains(lower, "halt(") && !strings.Contains(lower, "deny(") {
		return nil // pure facts/derived rules: no gate-blocking semantics
	}
	for _, line := range strings.Split(g.Rules, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "%") {
			continue
		}
		if tierInHead(line) != "" && tierInHead(line) != declared {
			return fmt.Errorf("x/genes: gene %q declares tier %s but rule emits tier %q: %s",
				g.Name, declared, tierInHead(line), line)
		}
	}
	return nil
}

// tierInHead extracts the third argument of halt(Entity, Reason, "Tier")
// from the head (before ':-') of a rule, or from a halt/deny fact.
// Returns "" when no tier-attributed halt/deny is present.
func tierInHead(ruleLine string) string {
	head := ruleLine
	if i := strings.Index(head, ":-"); i >= 0 {
		head = head[:i]
	}
	for _, pred := range []string{"halt", "deny"} {
		idx := strings.Index(strings.ToLower(head), pred+"(")
		if idx < 0 {
			continue
		}
		after := head[idx+len(pred)+1:]
		// Count to the third comma-separated argument, tolerate quoting.
		args := splitTopLevel(after)
		if len(args) >= 3 {
			t := strings.Trim(strings.TrimSpace(args[2]), "\"'")
			switch t {
			case string(core.TierT0_Axiom), string(core.TierT1_Governance),
				string(core.TierT2_Playbook), string(core.TierT3_User):
				return t
			}
		}
	}
	return ""
}

// splitTopLevel splits on commas outside parentheses, stopping at ')'.
func splitTopLevel(s string) []string {
	var args []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				args = append(args, s[start:i])
				return args
			}
			depth--
		case ',':
			if depth == 0 {
				args = append(args, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		args = append(args, s[start:])
	}
	return args
}
