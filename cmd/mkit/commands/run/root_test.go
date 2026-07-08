package run

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRunRequiresFlags(t *testing.T) {
	RunCmd.SetArgs([]string{})
	err := RunCmd.Execute()
	if err == nil {
		t.Fatal("expected error when required flags are missing")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected required-flag error, got: %v", err)
	}
}

func TestRunFlagsAreMarkedRequired(t *testing.T) {
	for _, name := range []string{"policy", "data", "target", "output"} {
		flag := RunCmd.Flags().Lookup(name)
		if flag == nil {
			t.Fatalf("flag %q not registered", name)
			continue
		}
		if flag.Annotations[cobra.BashCompOneRequiredFlag][0] != "true" {
			t.Errorf("flag %q should be marked required", name)
		}
	}
}
