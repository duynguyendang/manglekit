package main_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk"
)

// MockAction is a simple action that returns a success message
type MockAction struct{}

func (m *MockAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	return core.NewEnvelope("Success"), nil
}

func (m *MockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{Name: "mock_action", Type: "test"}
}

// BadTagStruct triggers a parsing error in toMangleFacts because "broken space"
// creates a malformed atom string like: broken space("Req", "test")
type BadTagStruct struct {
	Val string `mangle:"broken space"`
}

func createDummyPolicy(t *testing.T) string {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "dummy.dl")
	// Dummy rule to ensure runtime is active
	err := os.WriteFile(policyPath, []byte(`dummy(1).`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	return policyPath
}

func TestFailSafe_FailOpen_SystemError(t *testing.T) {
	// Setup: Fail-Open mode
	policyPath := createDummyPolicy(t)
	cfg := &config.Config{
		FailureMode: config.FailureModeOpen,
		Policy: config.PolicyConfig{
			Path: policyPath,
		},
	}

	client, err := sdk.NewClientWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	action := &MockAction{}
	protected := client.Protect(action)

	// Trigger: Send input that causes engine system error (parsing failure due to invalid tag)
	input := core.NewEnvelope(BadTagStruct{Val: "test"})

	// Execute
	result, err := protected.Execute(context.Background(), input)

	// Assert: Should succeed despite error because of Fail-Open
	if err != nil {
		t.Fatalf("expected success in fail-open mode, got error: %v", err)
	}
	if result.Payload.(string) != "Success" {
		t.Errorf("expected payload 'Success', got %v", result.Payload)
	}
}

func TestFailSafe_FailClosed_SystemError(t *testing.T) {
	// Setup: Fail-Closed mode (default)
	policyPath := createDummyPolicy(t)
	cfg := &config.Config{
		FailureMode: config.FailureModeClosed,
		Policy: config.PolicyConfig{
			Path: policyPath,
		},
	}

	client, err := sdk.NewClientWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	action := &MockAction{}
	protected := client.Protect(action)

	// Trigger: Send input that causes engine system error
	input := core.NewEnvelope(BadTagStruct{Val: "test"})

	// Execute
	_, err = protected.Execute(context.Background(), input)

	// Assert: Should fail
	if err == nil {
		t.Fatal("expected error in fail-closed mode, got nil")
	}
	// Verify it's not a policy violation but a system error propagation
	if errors.Is(err, core.ErrPolicyViolation) {
		t.Error("expected system error, got ErrPolicyViolation")
	}
	// The error message from solver.go should be "fact conversion error: ..."
	if !strings.Contains(err.Error(), "fact conversion error") {
		t.Errorf("expected fact conversion error, got: %v", err)
	}
}

func TestFailSafe_AlwaysBlockPolicyViolation(t *testing.T) {
	// Setup: Fail-Open mode
	cfg := &config.Config{
		FailureMode: config.FailureModeOpen,
		Policy: config.PolicyConfig{
			// We need a policy that denies.
		},
	}

	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "deny.dl")
	// Deny if foo matches bar.
	// We use strict Datalog syntax: deny(Req) :- foo(Req, "bar").
	// And we MUST declare 'foo' because Mangle's analysis expects it.
	err := os.WriteFile(policyPath, []byte(`
Decl foo(Req, Val).
deny(Req) :- foo(Req, "bar").
`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Policy.Path = policyPath

	client, err := sdk.NewClientWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	action := &MockAction{}
	protected := client.Protect(action)

	// Trigger: Send input that matches deny rule
	type Request struct {
		Foo string `mangle:"foo"`
	}
	input := core.NewEnvelope(Request{Foo: "bar"})

	// Execute
	_, err = protected.Execute(context.Background(), input)

	// Assert: Should fail even in Fail-Open mode because it's an explicit violation
	if err == nil {
		t.Fatal("expected error for policy violation, got nil")
	}
	if !errors.Is(err, core.ErrPolicyViolation) {
		t.Errorf("expected ErrPolicyViolation, got: %v", err)
	}
}
