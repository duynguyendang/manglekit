package ai_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/duynguyendang/manglekit/adapters/ai"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/providers/openai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// e2eGate is a StreamingGate that records calls and can deny the pre-check.
type e2eGate struct {
	assessErr    error
	reflectInput *core.Envelope
}

func (g *e2eGate) Assess(context.Context, core.ActionMetadata, core.Envelope) error {
	return g.assessErr
}

func (g *e2eGate) Reflect(_ context.Context, _ core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	g.reflectInput = &output
	return output, nil
}

// TestStreamingSupervisedAction_EndToEndWithOpenAIProvider wires a real
// OpenAI provider model (backed by a mock SSE streaming server) through the
// genkit adapter into the supervised streaming action, proving the full
// streaming path: pre-check before the stream opens, chunks through the
// Genkit streaming callback, and post-check on the assembled response.
//
// The test lives in the external ai_test package because providers/openai
// imports adapters/ai (an in-package test would create an import cycle).
func TestStreamingSupervisedAction_EndToEndWithOpenAIProvider(t *testing.T) {
	deltas := []string{"Hel", "lo ", "gated ", "world"}
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)
		for _, d := range deltas {
			fmt.Fprintf(w, "data: {\"id\":\"c\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", d)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: {\"id\":\"c\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":2,\"total_tokens\":3}}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	g := genkit.Init(context.Background())
	require.NotNil(t, g)
	require.NoError(t, openai.Init(g, "gpt-stream-test", openai.Config{APIKey: "k", BaseURL: server.URL + "/v1"}))

	model := genkit.LookupModel(g, "openai/gpt-stream-test")
	require.NotNil(t, model)
	gen := ai.NewGenkitAdapter(model, g)

	gate := &e2eGate{}
	action, err := ai.NewStreamingSupervisedAction("e2e-stream-llm", gen, gate)
	require.NoError(t, err)

	ch, err := action.Stream(context.Background(), core.NewEnvelope("hi"))
	require.NoError(t, err)

	var sb strings.Builder
	for c := range ch {
		require.NoError(t, c.Err)
		sb.WriteString(c.Text)
	}
	assert.Equal(t, strings.Join(deltas, ""), sb.String())

	// Post-check ran on the assembled full response.
	require.NotNil(t, gate.reflectInput)
	assert.Equal(t, strings.Join(deltas, ""), gate.reflectInput.Payload)

	final, ok := action.FinalEnvelope()
	require.True(t, ok)
	assert.Equal(t, strings.Join(deltas, ""), final.Payload)

	// Pre-check deny halts before the provider stream is even opened:
	// no additional HTTP request reaches the mock server.
	denyGate := &e2eGate{assessErr: core.ErrAlignment}
	denyAction, err := ai.NewStreamingSupervisedAction("e2e-stream-llm", gen, denyGate)
	require.NoError(t, err)
	before := atomic.LoadInt32(&requests)
	_, err = denyAction.Stream(context.Background(), core.NewEnvelope("hi"))
	require.Error(t, err)
	assert.ErrorIs(t, err, core.ErrAlignment)
	assert.Equal(t, before, atomic.LoadInt32(&requests), "no HTTP request should be made on pre-check deny")
	assert.Nil(t, denyGate.reflectInput, "post-check must not run when pre-check denies")
}
