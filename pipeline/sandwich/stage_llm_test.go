package sandwich_test

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/internal/providers/mock"
	"github.com/duynguyendang/manglekit/pipeline"
	"github.com/duynguyendang/manglekit/pipeline/sandwich"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger is a no-op logger for testing.
type mockLogger struct{}

func (l *mockLogger) With(kv ...any) core.Logger              { return &mockLogger{} }
func (l *mockLogger) Warnf(template string, args ...any)  {}
func (l *mockLogger) Errorf(template string, args ...any) {}
func (l *mockLogger) Infof(template string, args ...any)   {}
func (l *mockLogger) Debugf(template string, args ...any)  {}

// mockMeter is a no-op meter for testing.
type mockMeter struct{}

func (m *mockMeter) Record(name string, value float64, attrs ...any) {}

func TestLLMStage_Execute(t *testing.T) {
	t.Parallel()

	t.Run("should execute successfully", func(t *testing.T) {
		t.Parallel()

		llmClient := mock.NewLLM("test-model")
		stage := &sandwich.LLMStage{
			LLM:    llmClient,
			Logger: &mockLogger{},
			Meter:  &mockMeter{},
		}
		pctx := &pipeline.PipelineContext{
			Ctx:   context.Background(),
			Query: core.Query{Text: "test query"},
		}
		expectedAnswer := core.Answer{
			Text:      "model: test-model prompt: test query context: ",
			Citations: []core.Citation{},
			Meta: map[string]any{
				"token_usage": map[string]int{},
			},
		}

		err := stage.Execute(pctx)
		require.NoError(t, err)
		assert.Equal(t, expectedAnswer, pctx.Answer)
	})

	t.Run("should return an error if LLM fails", func(t *testing.T) {
		t.Parallel()

		llmClient := &mock.LLM{
			CompleteFunc: func(ctx context.Context, req core.LLMRequest) (core.LLMResponse, error) {
				return core.LLMResponse{}, errors.New("llm failed")
			},
		}
		stage := &sandwich.LLMStage{
			LLM:    llmClient,
			Logger: &mockLogger{},
			Meter:  &mockMeter{},
		}
		pctx := &pipeline.PipelineContext{
			Ctx:   context.Background(),
			Query: core.Query{Text: "test query"},
		}

		err := stage.Execute(pctx)
		require.Error(t, err)
	})
}
