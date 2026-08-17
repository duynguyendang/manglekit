package domain

import (
	"testing"
)

func TestVerifyStatusConstants(t *testing.T) {
	cases := map[VerifyStatus]bool{
		VerifyStatusPending: true,
		VerifyStatusPassed:  true,
		VerifyStatusFailed:  true,
		VerifyStatusWarning: true,
	}
	for status := range cases {
		if status == "" {
			t.Errorf("VerifyStatus constant must not be empty")
		}
	}
	// Ensure the values are the canonical ones.
	if VerifyStatusPassed != "FP32_PASSED" || VerifyStatusFailed != "LOGIC_VIOLATION" {
		t.Errorf("VerifyStatus constants drifted: %q / %q", VerifyStatusPassed, VerifyStatusFailed)
	}
}

func TestTrustTierConstants(t *testing.T) {
	tiers := []TrustTier{Tier0Kernel, Tier1Admin, Tier2AI, Tier3User}
	for _, tier := range tiers {
		if tier == "" {
			t.Errorf("TrustTier constant must not be empty")
		}
	}
}

func TestTaskAndOutputTypeConstants(t *testing.T) {
	if TaskTypeInduction != "INDUCTION" || TaskTypeGeneration != "GENERATION" ||
		TaskTypeAudit != "AUDIT" || TaskTypeRecovery != "RECOVERY" {
		t.Errorf("TaskType constants drifted")
	}
	if OutputTypePlan != "PLAN" || OutputTypeRule != "RULE" {
		t.Errorf("OutputType constants drifted")
	}
}

func TestPayloadIteration(t *testing.T) {
	p := Payload(func(yield func(Atom) bool) {
		yield(Atom{Subject: "s1", Predicate: "p", Object: "o1"})
		yield(Atom{Subject: "s2", Predicate: "p", Object: "o2"})
	})

	got := 0
	var first Atom
	for atom := range p {
		if got == 0 {
			first = atom
		}
		got++
	}
	if got != 2 {
		t.Fatalf("expected 2 atoms from Payload, got %d", got)
	}
	if first.Subject != "s1" {
		t.Errorf("expected first atom subject s1, got %q", first.Subject)
	}
}

func TestSignalConstruction(t *testing.T) {
	sig := Signal{
		ID:         "sig-1",
		Source:     PortType("sensor"),
		RawContent: "event",
		Intent:     IntentStr("observe"),
	}
	if sig.ID != "sig-1" || sig.Source != "sensor" || sig.RawContent != "event" {
		t.Errorf("Signal fields not preserved: %+v", sig)
	}
}

func TestCognitiveFrameDefaults(t *testing.T) {
	frame := CognitiveFrame{Intent: IntentStr("generate")}
	if frame.Status != "" {
		t.Errorf("expected zero-value status, got %q", frame.Status)
	}
	if frame.Intent != "generate" {
		t.Errorf("expected intent generate, got %q", frame.Intent)
	}
}

func TestDomainGeneSignature(t *testing.T) {
	var sig [32]byte
	sig[0] = 0xab
	gene := DomainGene{Name: "g1", Tier: Tier0Kernel, Rules: []byte("halt(Req)."), Signature: sig}
	if gene.Signature[0] != 0xab {
		t.Errorf("gene signature not preserved")
	}
	if string(gene.Rules) != "halt(Req)." {
		t.Errorf("gene rules not preserved: %q", string(gene.Rules))
	}
}

func TestAuditResultDefaults(t *testing.T) {
	res := AuditResult{Pass: true, ViolationTier: Tier3User}
	if !res.Pass {
		t.Errorf("expected Pass true")
	}
	if res.ViolationTier != Tier3User {
		t.Errorf("expected Tier3User, got %q", res.ViolationTier)
	}
}
