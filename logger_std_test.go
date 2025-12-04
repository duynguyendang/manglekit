package manglekit

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

func TestDefaultLoggerSingleton(t *testing.T) {
	l1 := getDefaultLogger()
	l2 := getDefaultLogger()

	if l1 != l2 {
		t.Errorf("expected singleton logger instance, got different instances")
	}

	if l1 == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestSlogAdapter_Interface(t *testing.T) {
	var _ core.Logger = &slogAdapter{}
}

func TestNewClient_InjectsDefaultLogger(t *testing.T) {
	// This test mainly ensures NewClient doesn't panic with the new default logger logic
	ctx := context.Background()
	c, err := NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	if c == nil {
		t.Fatal("NewClient returned nil client")
	}
}

func TestSlogAdapter_Output(t *testing.T) {
	// We can't easily capture os.Stdout in parallel tests safely without races or impacting other tests,
	// but we can test the adapter logic with a custom writer if we exposed the handler creation.
	// Since getDefaultLogger uses os.Stdout, we'll trust slog works.
	// However, we can construct a manual slogAdapter for testing purposes.

	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, nil)
	l := &slogAdapter{l: slog.New(h)}

	l.Info("test message", "key", "value")

	output := buf.String()
	if output == "" {
		t.Error("expected output, got empty string")
	}
	// Basic check for content
	if !strings.Contains(output, "msg=\"test message\"") {
		t.Errorf("expected output to contain msg=\"test message\", got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("expected output to contain key=value, got: %s", output)
	}
}
