package sdk

import (
	"context"
	"os"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.opentelemetry.io/otel/trace"
)

// MockEvaluator to test WithEngine
type MockEvaluator struct {
	mock.Mock
}

func (m *MockEvaluator) Assess(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	args := m.Called(ctx, actionMeta, input)
	return args.Error(0)
}

func (m *MockEvaluator) Reflect(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	args := m.Called(ctx, actionMeta, output)
	return args.Get(0).(core.Envelope), args.Error(1)
}

func (m *MockEvaluator) EvaluateSteering(ctx context.Context, input core.Envelope) (string, map[string]string, error) {
	args := m.Called(ctx, input)
	return args.String(0), args.Get(1).(map[string]string), args.Error(2)
}

func (m *MockEvaluator) GetActionConfig(ctx context.Context, input core.Envelope) (map[string]string, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockEvaluator) CheckRequirement(ctx context.Context, input core.Envelope, reqName string) (bool, error) {
	args := m.Called(ctx, input, reqName)
	return args.Bool(0), args.Error(1)
}

func (m *MockEvaluator) LoadPolicy(ctx context.Context, policy string) error {
	args := m.Called(ctx, policy)
	return args.Error(0)
}

func (m *MockEvaluator) LoadGherkinPolicy(ctx context.Context, featureContent string) error {
	args := m.Called(ctx, featureContent)
	return args.Error(0)
}

func (m *MockEvaluator) LoadFacts(facts []string) error {
	args := m.Called(facts)
	return args.Error(0)
}

func (m *MockEvaluator) RegisterAction(meta core.ActionMetadata) error {
	args := m.Called(meta)
	return args.Error(0)
}

func (m *MockEvaluator) AssessPlan(ctx context.Context, input core.Envelope) (core.Decision, error) {
	args := m.Called(ctx, input)
	return args.Get(0).(core.Decision), args.Error(1)
}

func (m *MockEvaluator) Query(ctx context.Context, facts []string, queryStr string) ([]map[string]string, error) {
	args := m.Called(ctx, facts, queryStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]map[string]string), args.Error(1)
}

func (m *MockEvaluator) Logger() core.Logger {
	// Return nil or a mock logger if needed
	return nil
}

func TestWithEngine(t *testing.T) {
	mockEngine := new(MockEvaluator)
	client, err := NewClient(context.Background(), WithEngine(mockEngine))
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, mockEngine, client.Engine())
}

func TestJITInitialization(t *testing.T) {
	// Zero config
	client, err := NewClient(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.Engine())
	assert.NotNil(t, client.Tracer())
}

func TestWithBlueprintPath_LoadsPolicy(t *testing.T) {
	// Create a temporary policy file with a deny rule
	// We use "Req" as the entity ID because that's core.EntityInput
	content := `halt("Req").`
	tmpfile, err := os.CreateTemp("", "policy.dl")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.WriteString(content)
	assert.NoError(t, err)
	tmpfile.Close()

	// Initialize client with path
	client, err := NewClient(context.Background(), WithBlueprintPath(tmpfile.Name()))
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Verify policy is loaded by triggering a deny via Assess
	err = client.Engine().Assess(context.Background(), core.ActionMetadata{Name: "test"}, core.NewEnvelope("payload"))
	assert.Error(t, err, "Assess should return error due to deny policy")
}

func TestWithTracerProvider(t *testing.T) {
	// Use NoopTracerProvider to verify injection
	tp := trace.NewNoopTracerProvider()

	client, err := NewClient(context.Background(), WithTracerProvider(tp))
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.Tracer())
}
