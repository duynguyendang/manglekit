package guard

import (
	"context"
	"sync"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/engine"
)

// MockAction is a simple action for testing that echoes the input.
type MockAction struct{}

func (m *MockAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	return input, nil
}

func (m *MockAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "mock-action",
		Type: "test",
	}
}

// LoggerCapturingAction is an action that verifies the logger is available in context.
type LoggerCapturingAction struct {
	mu            sync.Mutex
	capturedMsgs  []string
	loggerPresent bool
}

func (a *LoggerCapturingAction) Execute(ctx context.Context, input core.Envelope) (core.Envelope, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	logger := core.LoggerFromContext(ctx)
	// Check if the logger is not a NopLogger (i.e., a real logger was injected)
	_, isNop := logger.(core.NopLogger)
	a.loggerPresent = !isNop

	// Use the logger to verify it works
	logger.Info("test message from action", "key", "value")

	return input, nil
}

func (a *LoggerCapturingAction) Metadata() core.ActionMetadata {
	return core.ActionMetadata{
		Name: "logger-capturing-action",
		Type: "test",
	}
}

func (a *LoggerCapturingAction) WasLoggerPresent() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.loggerPresent
}

// TestLogger is a simple logger implementation for testing.
type TestLogger struct {
	mu   sync.Mutex
	logs []string
}

func (l *TestLogger) Debug(msg string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, "DEBUG: "+msg)
}

func (l *TestLogger) Info(msg string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, "INFO: "+msg)
}

func (l *TestLogger) Warn(msg string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, "WARN: "+msg)
}

func (l *TestLogger) Error(msg string, fields ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, "ERROR: "+msg)
}

func (l *TestLogger) With(fields ...any) core.Logger {
	return l
}

func (l *TestLogger) GetLogs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.logs...)
}

func TestGuardedAction_Execute(t *testing.T) {
	// 1. Setup
	eng := engine.New()
	mock := &MockAction{}
	guardedAction := New(mock, eng, "closed")

	// 2. Create input
	inputPayload := "hello world"
	input := core.NewEnvelope(inputPayload)

	// 3. Execute
	output, err := guardedAction.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	// 4. Verify
	if output.ID != input.ID {
		t.Errorf("Expected output ID to be %q, got %q", input.ID, output.ID)
	}

	outputPayload, ok := output.Payload.(string)
	if !ok {
		t.Fatalf("Expected output payload to be a string, but it was not")
	}

	if outputPayload != inputPayload {
		t.Errorf("Expected output payload to be %q, got %q", inputPayload, outputPayload)
	}
}

func TestGuardedAction_LoggerContextInjection(t *testing.T) {
	// 1. Setup with a custom logger
	testLogger := &TestLogger{}
	eng := engine.NewWithObservability(nil, testLogger)
	capturingAction := &LoggerCapturingAction{}
	guardedAction := New(capturingAction, eng, "closed")

	// 2. Create input
	input := core.NewEnvelope("test payload")

	// 3. Execute
	_, err := guardedAction.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute() returned an unexpected error: %v", err)
	}

	// 4. Verify the logger was present in the context
	if !capturingAction.WasLoggerPresent() {
		t.Error("Expected logger to be present in context, but it was not (got NopLogger)")
	}

	// 5. Verify the test logger captured messages
	logs := testLogger.GetLogs()
	if len(logs) == 0 {
		t.Error("Expected test logger to have captured log messages, but it was empty")
	}

	// 6. Verify expected log messages were captured
	foundActionDebug := false
	foundActionInfo := false
	for _, log := range logs {
		if log == "DEBUG: starting action execution" {
			foundActionDebug = true
		}
		if log == "INFO: test message from action" {
			foundActionInfo = true
		}
	}

	if !foundActionDebug {
		t.Error("Expected 'starting action execution' debug log, but it was not found")
	}
	if !foundActionInfo {
		t.Error("Expected 'test message from action' info log from inner action, but it was not found")
	}
}

func TestLoggerFromContext_ReturnsNopLoggerWhenNotSet(t *testing.T) {
	// When no logger is in context, LoggerFromContext should return NopLogger
	ctx := context.Background()
	logger := core.LoggerFromContext(ctx)

	// Verify it's a NopLogger (should not panic when used)
	logger.Debug("test debug")
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")

	// Verify the type is NopLogger
	if _, ok := logger.(core.NopLogger); !ok {
		t.Errorf("Expected NopLogger when context has no logger, got %T", logger)
	}
}

func TestLoggerWithContext_RoundTrip(t *testing.T) {
	// Test that a logger can be stored and retrieved from context
	testLogger := &TestLogger{}
	ctx := context.Background()

	// Store logger in context
	ctx = core.LoggerWithContext(ctx, testLogger)

	// Retrieve logger from context
	retrieved := core.LoggerFromContext(ctx)

	// Use the retrieved logger
	retrieved.Info("round trip test")

	// Verify the message was captured by the original logger
	logs := testLogger.GetLogs()
	if len(logs) != 1 || logs[0] != "INFO: round trip test" {
		t.Errorf("Expected log message 'INFO: round trip test', got %v", logs)
	}
}
