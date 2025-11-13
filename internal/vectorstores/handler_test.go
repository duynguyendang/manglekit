package vectorstores

import (
	"testing"

	"github.com/duynguyendang/manglekit/core"
)

// MockLogger is a mock implementation of core.Logger for testing.
type MockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *MockLogger) Debugf(msg string, kv ...any) {
	m.debugMessages = append(m.debugMessages, msg)
}

func (m *MockLogger) Infof(msg string, kv ...any) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *MockLogger) Warnf(msg string, kv ...any) {
	m.warnMessages = append(m.warnMessages, msg)
}

func (m *MockLogger) Errorf(msg string, kv ...any) {
	m.errorMessages = append(m.errorMessages, msg)
}

func (m *MockLogger) With(kv ...any) core.Logger {
	return m
}

// TestNewHandler tests that NewHandler returns a valid Handler.
func TestNewHandler(t *testing.T) {
	handler := NewHandler()
	if handler == nil {
		t.Fatal("expected handler to be non-nil")
	}

	if handler.Kind() != core.KindVectorStore {
		t.Errorf("expected Kind to be %s, got %s", core.KindVectorStore, handler.Kind())
	}
}
