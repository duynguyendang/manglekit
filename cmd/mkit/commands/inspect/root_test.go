package inspect

import (
	"strings"
	"testing"
)

func TestInspectCommandRegistered(t *testing.T) {
	if InspectCmd == nil {
		t.Fatal("InspectCmd must be initialized")
	}
	if !strings.Contains(strings.ToLower(InspectCmd.Short), "inspect") {
		t.Errorf("expected inspect command short description, got %q", InspectCmd.Short)
	}
}
