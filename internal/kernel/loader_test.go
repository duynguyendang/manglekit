package kernel

import (
	"strings"
	"testing"
)

func TestLoadKernel(t *testing.T) {
	rules, err := LoadKernel()
	if err != nil {
		t.Fatalf("LoadKernel() returned error: %v", err)
	}

	if rules == "" {
		t.Fatal("LoadKernel() returned empty string")
	}

	if !strings.Contains(rules, "% --- OODA Phases ---") {
		t.Error("LoadKernel() missing OODA section")
	}
	if !strings.Contains(rules, "% --- Agent Registry ---") {
		t.Error("LoadKernel() missing Agents section")
	}
	if !strings.Contains(rules, "% --- Patterns ---") {
		t.Error("LoadKernel() missing Patterns section")
	}
	if !strings.Contains(rules, "% --- Validation Rules ---") {
		t.Error("LoadKernel() missing Validation section")
	}

	if !strings.Contains(rules, "phase_order") {
		t.Error("LoadKernel() missing phase_order predicate")
	}
}

func TestKernelRulesCombine(t *testing.T) {
	r := KernelRules{
		OODA:       "ooda_content",
		Agents:     "agents_content",
		Patterns:   "patterns_content",
		Validation: "validation_content",
	}

	combined := r.Combine()

	if !strings.Contains(combined, "ooda_content") {
		t.Error("Combine() missing OODA content")
	}
	if !strings.Contains(combined, "agents_content") {
		t.Error("Combine() missing Agents content")
	}
	if !strings.Contains(combined, "patterns_content") {
		t.Error("Combine() missing Patterns content")
	}
	if !strings.Contains(combined, "validation_content") {
		t.Error("Combine() missing Validation content")
	}
}

func TestLoaderNewLoader(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() returned error: %v", err)
	}

	kernel := loader.GetKernel()
	if kernel == "" {
		t.Error("GetKernel() returned empty string")
	}

	if !strings.Contains(kernel, "MANGLEKIT KERNEL RULES") {
		t.Error("GetKernel() missing header comment")
	}
}

func TestLoaderMerge(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() returned error: %v", err)
	}

	kernelOnly, err := loader.Merge(Profile{})
	if err != nil {
		t.Fatalf("Merge(Profile{}) returned error: %v", err)
	}
	if kernelOnly != loader.GetKernel() {
		t.Error("Merge(empty profile) should return kernel unchanged")
	}

	profile := Profile{
		Name: "test-profile",
		Rules: `
% Test profile rules
allow_test(_, _) :- true.
`,
	}

	merged, err := loader.Merge(profile)
	if err != nil {
		t.Fatalf("Merge(profile) returned error: %v", err)
	}

	if !strings.Contains(merged, "allow_test") {
		t.Error("Merge() missing profile rules")
	}
	if !strings.Contains(merged, "Profile: test-profile") {
		t.Error("Merge() missing profile name comment")
	}
}

func TestDefaultProfile(t *testing.T) {
	profile := DefaultProfile()

	if profile.Name != "default" {
		t.Errorf("DefaultProfile().Name = %q, want default", profile.Name)
	}

	if profile.Rules == "" {
		t.Error("DefaultProfile().Rules is empty")
	}

	if !strings.Contains(profile.Rules, "allow(_, _, _)") {
		t.Error("DefaultProfile().Rules missing allow rule")
	}

	if profile.Metadata == nil {
		t.Error("DefaultProfile().Metadata is nil")
	}
}