package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStreamGenerator is a TextGenerator that streams fixed chunks.
type mockStreamGenerator struct {
	chunks     []string
	streams    int // number of times Stream was opened
	completeFn func(ctx context.Context, prompt string) (string, error)
}

func (m *mockStreamGenerator) Complete(ctx context.Context, prompt string) (string, error) {
	if m.completeFn != nil {
		return m.completeFn(ctx, prompt)
	}
	return "", nil
}

func (m *mockStreamGenerator) Generate(ctx context.Context, prompt string, opts ...core.GenerateOption) (*core.LLMResponse, error) {
	return &core.LLMResponse{}, nil
}

func (m *mockStreamGenerator) Stream(ctx context.Context, prompt string) (<-chan core.StreamChunk, error) {
	m.streams++
	ch := make(chan core.StreamChunk, len(m.chunks))
	go func() {
		defer close(ch)
		for _, c := range m.chunks {
			ch <- core.StreamChunk{Text: c}
		}
	}()
	return ch, nil
}

// mockGate is a governance gate that records calls and can deny either phase.
type mockGate struct {
	assessErr  error
	reflectErr error

	assessInput   *core.Envelope
	reflectOutput *core.Envelope
}

func (g *mockGate) Assess(_ context.Context, _ core.ActionMetadata, input core.Envelope) error {
	g.assessInput = &input
	return g.assessErr
}

func (g *mockGate) Reflect(_ context.Context, _ core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	g.reflectOutput = &output
	return output, g.reflectErr
}

func collectChunks(t *testing.T, ch <-chan core.StreamChunk) []core.StreamChunk {
	t.Helper()
	var got []core.StreamChunk
	for c := range ch {
		got = append(got, c)
	}
	return got
}

func TestStreamingSupervisedAction_ChunksForwardedAndPostChecked(t *testing.T) {
	gen := &mockStreamGenerator{chunks: []string{"Hello", " ", "world"}}
	gate := &mockGate{}
	action, err := NewStreamingSupervisedAction("stream-llm", gen, gate)
	require.NoError(t, err)

	ch, err := action.Stream(context.Background(), core.NewEnvelope("hi"))
	require.NoError(t, err)

	got := collectChunks(t, ch)
	require.Len(t, got, 3)
	assert.Equal(t, "Hello", got[0].Text)
	assert.Equal(t, " ", got[1].Text)
	assert.Equal(t, "world", got[2].Text)
	assert.NoError(t, got[2].Err)

	// Post-check ran on the assembled full response.
	require.NotNil(t, gate.reflectOutput)
	assert.Equal(t, "Hello world", gate.reflectOutput.Payload)
	assert.Equal(t, "stream-llm", gate.reflectOutput.GetMeta("action_name"))

	// Final envelope is exposed after a passing post-check.
	final, ok := action.FinalEnvelope()
	require.True(t, ok)
	assert.Equal(t, "Hello world", final.Payload)
}

func TestStreamingSupervisedAction_PreCheckDenyHaltsBeforeFirstChunk(t *testing.T) {
	gen := &mockStreamGenerator{chunks: []string{"secret"}}
	gate := &mockGate{assessErr: core.ErrAlignment}
	action, err := NewStreamingSupervisedAction("stream-llm", gen, gate)
	require.NoError(t, err)

	ch, err := action.Stream(context.Background(), core.NewEnvelope("hi"))
	require.Error(t, err)
	assert.Nil(t, ch)
	assert.ErrorIs(t, err, core.ErrAlignment)

	// The provider stream was never opened: no chunks could be produced.
	assert.Zero(t, gen.streams)
	assert.Nil(t, gate.reflectOutput)

	_, ok := action.FinalEnvelope()
	assert.False(t, ok)
}

func TestStreamingSupervisedAction_PostCheckDenySurfacesTerminalError(t *testing.T) {
	gen := &mockStreamGenerator{chunks: []string{"bad", " output"}}
	gate := &mockGate{reflectErr: core.ErrAlignment}
	action, err := NewStreamingSupervisedAction("stream-llm", gen, gate)
	require.NoError(t, err)

	ch, err := action.Stream(context.Background(), core.NewEnvelope("hi"))
	require.NoError(t, err)

	got := collectChunks(t, ch)
	require.Len(t, got, 3) // 2 text chunks + terminal error chunk
	assert.Equal(t, "bad", got[0].Text)
	assert.Equal(t, " output", got[1].Text)
	require.Error(t, got[2].Err)
	assert.ErrorIs(t, got[2].Err, core.ErrAlignment)

	// No final envelope is exposed when the post-check denies.
	_, ok := action.FinalEnvelope()
	assert.False(t, ok)
}

func TestStreamingSupervisedAction_Execute_NonStreamingPath(t *testing.T) {
	gen := &mockStreamGenerator{completeFn: func(_ context.Context, _ string) (string, error) {
		return "full reply", nil
	}}
	gate := &mockGate{}
	action, err := NewStreamingSupervisedAction("stream-llm", gen, gate)
	require.NoError(t, err)

	out, err := action.Execute(context.Background(), core.NewEnvelope("hi"))
	require.NoError(t, err)
	assert.Equal(t, "full reply", out.Payload)

	// Both gate phases ran on the non-streaming path.
	assert.NotNil(t, gate.assessInput)
	assert.NotNil(t, gate.reflectOutput)
}

func TestStreamingSupervisedAction_Execute_PostCheckDeny(t *testing.T) {
	gen := &mockStreamGenerator{completeFn: func(_ context.Context, _ string) (string, error) {
		return "forbidden", nil
	}}
	gate := &mockGate{reflectErr: errors.New("denied")}
	action, err := NewStreamingSupervisedAction("stream-llm", gen, gate)
	require.NoError(t, err)

	_, err = action.Execute(context.Background(), core.NewEnvelope("hi"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "post-check")
}

func TestNewStreamingSupervisedAction_Validation(t *testing.T) {
	gate := &mockGate{}

	_, err := NewStreamingSupervisedAction("a", nil, gate)
	assert.Error(t, err)

	_, err = NewStreamingSupervisedAction("a", &mockStreamGenerator{}, nil)
	assert.Error(t, err)
}
