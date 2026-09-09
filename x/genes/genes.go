// Package genes provides an OPTIONAL, file-based "crystallized logic"
// extension for learned Datalog rules — genes — that are applied to the
// policy engine through its official source channel (sdk.Client.LoadPolicy).
//
// Design guardrails (ADR-003):
//   - This package is an extension: core, sdk, and adapters never import it.
//   - A gene NEVER enters the enforcement path via an interface hook
//     (the old genome parameter is gone). Applying a gene means compiling
//     it to Datalog source and loading it as policy — auditable, diffable,
//     reloadable, rollback-able like any other policy change.
//   - Integrity: gene rules are SHA-256 signed over their canonical form;
//     compilation refuses tampered or unsigned genes by default.
//   - Promotion: genes carrying hard tiers (T0/T1, which can BLOCK) are
//     refused unless the caller passes WithAllowHardTiers — modeling the
//     human-review step that promotes learned (advisory T2/T3) rules into
//     governance. Advisory rules follow the tier semantics of the gate
//     (see docs/context/governance/datalog-policies.md).
package genes

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
)

// Gene is a signed unit of Datalog logic with tiered governance intent.
type Gene struct {
	// Name uniquely identifies the gene within a manifest.
	Name string `yaml:"name"`
	// Tier is the governance tier the gene's rules are meant to carry
	// (core.TierT0_Axiom..TierT3_User). Hard tiers require explicit opt-in
	// at compile time (WithAllowHardTiers).
	Tier core.Tier `yaml:"tier"`
	// Source documents provenance, e.g. "mkit gen 2026-09-09, review #12".
	Source string `yaml:"source"`
	// Intents the gene applies to; empty means "all intents".
	Intents []string `yaml:"intents"`
	// Rules is self-contained Datalog source (facts and/or rules).
	Rules string `yaml:"rules"`
	// Signature is the SHA-256 over the canonical form (see Sign).
	Signature [32]byte `yaml:"-"`
	// Signed records whether a signature was produced/verified.
	Signed bool `yaml:"-"`
}

// canonical is the exact byte sequence covered by the signature.
func (g Gene) canonical() []byte {
	h := sha256.New()
	fmt.Fprintf(h, "name=%s\n", g.Name)
	fmt.Fprintf(h, "tier=%s\n", g.Tier)
	fmt.Fprintf(h, "source=%s\n", g.Source)
	fmt.Fprintf(h, "intents=%v\n", g.Intents)
	fmt.Fprintf(h, "rules=%s\n", g.Rules)
	return h.Sum(nil) // digest of the canonical preamble — see Sign
}

// Sign computes and attaches the integrity signature.
func (g *Gene) Sign() {
	// Double-hash: g.canonical() is already a digest of the fields.
	g.Signature = sha256.Sum256(g.canonical())
	g.Signed = true
}

// SignatureHex returns the hex-encoded signature (manifest representation).
func (g Gene) SignatureHex() string { return hex.EncodeToString(g.Signature[:]) }

// SetSignatureHex parses a hex signature into the gene.
func (g *Gene) SetSignatureHex(s string) error {
	raw, err := hex.DecodeString(s)
	if err != nil || len(raw) != len(g.Signature) {
		return fmt.Errorf("invalid signature hex: %q", s)
	}
	copy(g.Signature[:], raw)
	g.Signed = true
	return nil
}

// Verify recomputes the signature over the current field values.
func (g Gene) Verify() error {
	if !g.Signed {
		return fmt.Errorf("gene %q is unsigned", g.Name)
	}
	if g.Signature != sha256.Sum256(g.canonical()) {
		return fmt.Errorf("gene %q signature mismatch — rules or metadata were modified after signing", g.Name)
	}
	return nil
}

// validTier accepts only the four governance tiers (not Unknown/empty).
func validTier(t core.Tier) bool {
	switch t {
	case core.TierT0_Axiom, core.TierT1_Governance, core.TierT2_Playbook, core.TierT3_User:
		return true
	}
	return false
}

// isHardTier reports whether rules from this gene can block at the gate
// (supervisor blocks Tier0/Tier1; T2/T3 are advisory).
func isHardTier(t core.Tier) bool {
	return t == core.TierT0_Axiom || t == core.TierT1_Governance
}
