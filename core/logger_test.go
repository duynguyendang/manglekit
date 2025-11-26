package core

import (
	"context"
	"testing"
)

func TestNopLogger(t *testing.T) {
	// NopLogger should not panic when called
	logger := NopLogger{}

	// Test all methods don't panic
	logger.Debug("test debug")
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")

	// With should return the same NopLogger
	child := logger.With("key", "value")
	if _, ok := child.(NopLogger); !ok {
		t.Errorf("Expected NopLogger.With to return NopLogger, got %T", child)
	}
}

func TestLoggerFromContext_ReturnsNopLoggerWhenEmpty(t *testing.T) {
	ctx := context.Background()
	logger := LoggerFromContext(ctx)

	if _, ok := logger.(NopLogger); !ok {
		t.Errorf("Expected NopLogger when context has no logger, got %T", logger)
	}
}

func TestLoggerWithContext_StoresLogger(t *testing.T) {
	ctx := context.Background()
	testLogger := &testingLogger{}

	ctx = LoggerWithContext(ctx, testLogger)
	retrieved := LoggerFromContext(ctx)

	if retrieved != testLogger {
		t.Errorf("Expected to retrieve the same logger, got %T", retrieved)
	}
}

func TestLoggerWithContext_NilDefaultsToNop(t *testing.T) {
	ctx := context.Background()

	// Passing nil should store NopLogger
	ctx = LoggerWithContext(ctx, nil)
	retrieved := LoggerFromContext(ctx)

	if _, ok := retrieved.(NopLogger); !ok {
		t.Errorf("Expected NopLogger when nil is passed, got %T", retrieved)
	}
}

func TestLoggerRoundTrip(t *testing.T) {
	testLogger := &testingLogger{}
	ctx := context.Background()

	// Store logger
	ctx = LoggerWithContext(ctx, testLogger)

	// Retrieve and use logger
	logger := LoggerFromContext(ctx)
	logger.Info("test message", "key", "value")

	// Verify the message was recorded
	if len(testLogger.msgs) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(testLogger.msgs))
	}
	if testLogger.msgs[0] != "INFO: test message" {
		t.Errorf("Expected 'INFO: test message', got '%s'", testLogger.msgs[0])
	}
}

func TestSetDefaultLogger(t *testing.T) {
	// Save original default
	originalDefault := defaultLogger
	defer func() {
		defaultLogger = originalDefault
	}()

	testLogger := &testingLogger{}
	SetDefaultLogger(testLogger)

	// Now LoggerFromContext should return our test logger when context is empty
	ctx := context.Background()
	retrieved := LoggerFromContext(ctx)

	if retrieved != testLogger {
		t.Errorf("Expected test logger as default, got %T", retrieved)
	}
}

func TestSetDefaultLogger_NilResetsToNop(t *testing.T) {
	// Save original default
	originalDefault := defaultLogger
	defer func() {
		defaultLogger = originalDefault
	}()

	// Set to nil, should reset to NopLogger
	SetDefaultLogger(nil)

	ctx := context.Background()
	retrieved := LoggerFromContext(ctx)

	if _, ok := retrieved.(NopLogger); !ok {
		t.Errorf("Expected NopLogger after setting nil default, got %T", retrieved)
	}
}

// testingLogger is a simple logger implementation for testing
type testingLogger struct {
	msgs []string
}

func (l *testingLogger) Debug(msg string, fields ...any) {
	l.msgs = append(l.msgs, "DEBUG: "+msg)
}

func (l *testingLogger) Info(msg string, fields ...any) {
	l.msgs = append(l.msgs, "INFO: "+msg)
}

func (l *testingLogger) Warn(msg string, fields ...any) {
	l.msgs = append(l.msgs, "WARN: "+msg)
}

func (l *testingLogger) Error(msg string, fields ...any) {
	l.msgs = append(l.msgs, "ERROR: "+msg)
}

func (l *testingLogger) With(fields ...any) Logger {
	return l
}
