package genes

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

func testGene(name string, tier core.Tier, rules string) Gene {
	g := Gene{
		Name:    name,
		Tier:    tier,
		Source:  "unit-test",
		Intents: []string{"doc_gen"},
		Rules:   rules,
	}
	g.Sign()
	return g
}

const advisoryRule = `halt(Req, "outline missing", "T2") :- action_operation(Req, "write_doc"), ! meta("has_outline", "true").`

func TestSignVerifyAndTamperDetection(t *testing.T) {
	g := testGene("g1", core.TierT2_Playbook, advisoryRule)
	if err := g.Verify(); err != nil {
		t.Fatalf("fresh signature invalid: %v", err)
	}
	g.Rules = strings.Replace(g.Rules, "T2", "T1", 1)
	if err := g.Verify(); err == nil {
		t.Fatal("tampered rules must fail verification")
	}
}

func TestPoolManifestRoundTrip(t *testing.T) {
	g1 := testGene("outline-guard", core.TierT2_Playbook, advisoryRule)
	g2 := testGene("tone-hint", core.TierT3_User, `halt(Req, "tone", "T3") :- action_operation(Req, "write_doc"), ! meta("tone", "formal").`)

	var buf bytes.Buffer
	if err := WritePool(&buf, []Gene{g1, g2}); err != nil {
		t.Fatal(err)
	}
	pool, err := LoadPool(&buf)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(pool.Genes()) != 2 {
		t.Fatalf("want 2 genes, got %d", len(pool.Genes()))
	}
	active := pool.Active("doc_gen")
	if len(active) != 2 {
		t.Fatalf("Active(doc_gen) = %d, want 2", len(active))
	}
	if len(pool.Active("other_intent")) != 0 {
		t.Error("intent filter leaked genes")
	}
}

func TestLoadPoolRejectsCorruption(t *testing.T) {
	g := testGene("g", core.TierT2_Playbook, advisoryRule)
	var buf bytes.Buffer
	if err := WritePool(&buf, []Gene{g}); err != nil {
		t.Fatal(err)
	}
	// Flip one byte inside the rules text (signature must now mismatch).
	broken := strings.Replace(buf.String(), "outline missing", "outline MISSING", 1)
	if _, err := LoadPool(strings.NewReader(broken)); err == nil {
		t.Fatal("expected signature mismatch on corrupted manifest")
	}
	// Unsigned gene.
	bare := "version: 1\ngenes:\n  - name: x\n    tier: T2\n    rules: 'a.'\n"
	if _, err := LoadPool(strings.NewReader(bare)); err == nil {
		t.Fatal("expected rejection of unsigned gene")
	}
	// Bad tier.
	badTier := "version: 1\ngenes:\n  - name: x\n    tier: T7\n    rules: 'a.'\n    signature: 00\n"
	if _, err := LoadPool(strings.NewReader(badTier)); err == nil {
		t.Fatal("expected rejection of invalid tier")
	}
}

func TestCompilePromotionDiscipline(t *testing.T) {
	soft := testGene("soft", core.TierT2_Playbook, advisoryRule)
	hard := testGene("hard", core.TierT1_Governance,
		strings.Replace(advisoryRule, `"T2"`, `"T1"`, 1))

	// Advisory genes compile by default.
	src, err := Compile([]Gene{soft})
	if err != nil {
		t.Fatalf("advisory compile: %v", err)
	}
	if !strings.Contains(src, "outline missing") {
		t.Errorf("compiled source missing rule text:\n%s", src)
	}

	// Hard tiers require the explicit opt-in.
	if _, err := Compile([]Gene{hard}); err == nil {
		t.Fatal("hard-tier gene must not compile without WithAllowHardTiers")
	}
	if _, err := Compile([]Gene{hard}, WithAllowHardTiers()); err != nil {
		t.Fatalf("hard-tier compile with opt-in: %v", err)
	}

	// Declared tier must match the tiers inside the rules.
	mismatched := Gene{Name: "liar", Tier: core.TierT2_Playbook,
		Rules: strings.Replace(advisoryRule, `"T2"`, `"T1"`, 1)}
	mismatched.Sign()
	if _, err := Compile([]Gene{mismatched}, WithAllowHardTiers()); err == nil {
		t.Fatal("tier mismatch between gene and rules must be rejected")
	}

	// Tampered gene cannot compile even with opt-in.
	soft.Rules += "\n" // mutation after signing
	if _, err := Compile([]Gene{soft}, WithAllowHardTiers()); err == nil {
		t.Fatal("tampered gene must fail compile-time verification")
	}
}

func TestApplyToEndToEnd(t *testing.T) {
	ctx := context.Background()
	client, err := newClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Shutdown(ctx) }()

	// A supervised action that simply echoes.
	act := &staticAction{}
	client.RegisterSupervised("write_doc", act)

	g := testGene("outline-guard", core.TierT2_Playbook, advisoryRule)

	// 1. Without the gene: action runs, no violation facts exist.
	if _, err := client.ExecuteByName(ctx, "write_doc", "draft"); err != nil {
		t.Fatalf("baseline execute: %v", err)
	}

	// 2. Apply the learned advisory gene (no payload outline meta → halt/2
	//    rule matches with tier T2) — advisory, so execution still succeeds.
	if err := ApplyTo(ctx, client, []Gene{g}); err != nil {
		t.Fatalf("ApplyTo advisory: %v", err)
	}
	if _, err := client.ExecuteByName(ctx, "write_doc", "draft"); err != nil {
		t.Fatalf("advisory gene must not block, got: %v", err)
	}

	// 3. Promoted gene (same lesson, T1 tier, explicit human opt-in) blocks.
	hardGene := testGene("outline-guard-promoted", core.TierT1_Governance,
		strings.Replace(advisoryRule, `"T2"`, `"T1"`, 1))
	promoClient, err := newClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = promoClient.Shutdown(ctx) }()
	promoClient.RegisterSupervised("write_doc", &staticAction{})
	if err := ApplyTo(ctx, promoClient, []Gene{hardGene}, WithAllowHardTiers()); err != nil {
		t.Fatalf("ApplyTo promoted: %v", err)
	}
	_, err = promoClient.ExecuteByName(ctx, "write_doc", "draft")
	if !core.IsPolicyViolationError(err) {
		t.Fatalf("promoted T1 gene must block, got err=%v", err)
	}
	// ...and the same call with the learned meta present is allowed:
	// the lesson is honored, not just punished.
	_, err = promoClient.ExecuteByName(ctx, "write_doc", "draft",
		sdk.WithMetadata("has_outline", "true"))
	if err != nil {
		t.Fatalf("compliant payload must proceed: %v", err)
	}
}

func TestApplyToRejectsInvalidProgramAtomically(t *testing.T) {
	ctx := context.Background()
	client, err := newClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Shutdown(ctx) }()

	bad := testGene("broken", core.TierT2_Playbook, "this is not datalog (((.")
	err = ApplyTo(ctx, client, []Gene{bad})
	if err == nil {
		t.Fatal("expected compile+load of invalid rules to fail")
	}
	if !strings.Contains(err.Error(), "x/genes") {
		t.Errorf("error should be wrapped by x/genes, got %v", err)
	}
}
