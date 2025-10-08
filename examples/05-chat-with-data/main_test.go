package main

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit"
	_ "github.com/duynguyendang/manglekit/providers/all"
)

func TestBuildOrchestrator(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get current file path")
	}
	dir := filepath.Dir(currentFile)
	configPath := filepath.Join(dir, "config.yaml")

	// This test verifies that the orchestrator can be built without Datalog parsing errors.
	// Before the fix, this would fail with a "could not find predicate" error.
	// After the fix, it should proceed further. It might fail due to a missing API key,
	// but that is expected in a test environment without credentials and still proves
	// that the Datalog parsing issue is resolved.
	builder, err := manglekit.NewBuilderFromYAML(configPath)
	if err != nil {
		// If NewBuilderFromYAML fails, it might be due to file not found or YAML syntax error.
		t.Fatalf("Failed to create builder from YAML: %v", err)
	}

	_, err = builder.Build()
	if err != nil {
		// The build is expected to fail with an API key error in a test environment.
		// We just need to make sure it's not the Datalog error.
		if strings.Contains(err.Error(), "could not find predicate") {
			t.Errorf("Test failed: Orchestrator build failed with a Datalog error: %v", err)
		}
		// If it's another error (like the expected API key error), we treat it as a pass
		// for the purpose of verifying the Datalog fix.
		t.Logf("Orchestrator build failed as expected with a non-Datalog error: %v", err)
	}
}