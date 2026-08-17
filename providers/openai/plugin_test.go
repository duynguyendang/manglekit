package openai

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newStreamTestGenkit registers an OpenAI model backed by a mock server.
func newStreamTestGenkit(t *testing.T, handler http.HandlerFunc) (*genkit.Genkit, string) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	g := genkit.Init(context.Background())
	require.NotNil(t, g)

	err := Init(g, "gpt-test", Config{APIKey: "test-key", BaseURL: server.URL + "/v1"})
	require.NoError(t, err)

	return g, "openai/gpt-test"
}

// sseHandler serves an OpenAI-style chat completion stream: one SSE event per
// delta, then a usage-only event, then [DONE].
func sseHandler(t *testing.T, deltas []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		for _, d := range deltas {
			fmt.Fprintf(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[{\"index\":0,\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", d)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-test\",\"choices\":[],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":11,\"total_tokens\":18}}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}
}

func TestGenerate_Streaming_DeliversChunksAndAssembledResponse(t *testing.T) {
	deltas := []string{"Hello", ", ", "stream", "ing", "!"}
	g, modelName := newStreamTestGenkit(t, sseHandler(t, deltas))

	model := genkit.LookupModel(g, modelName)
	require.NotNil(t, model)

	req := ai.NewModelRequest(nil, ai.NewUserMessage(ai.NewTextPart("say hi")))

	var chunks []string
	cb := func(_ context.Context, chunk *ai.ModelResponseChunk) error {
		chunks = append(chunks, chunk.Text())
		return nil
	}

	resp, err := model.Generate(context.Background(), req, cb)
	require.NoError(t, err)

	// Every delta must arrive as an incremental chunk, in order.
	assert.Equal(t, deltas, chunks)

	// The returned response is the assembled full text.
	assert.Equal(t, strings.Join(deltas, ""), resp.Text())

	// Usage comes from the final usage-only event.
	require.NotNil(t, resp.Usage)
	assert.EqualValues(t, 7, resp.Usage.InputTokens)
	assert.EqualValues(t, 11, resp.Usage.OutputTokens)
	assert.EqualValues(t, 18, resp.Usage.TotalTokens)
}

func TestGenerate_Streaming_NoCallbackFallsBackToNonStreaming(t *testing.T) {
	g, modelName := newStreamTestGenkit(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"gpt-test","choices":[{"index":0,"message":{"role":"assistant","content":"full reply"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":3,"total_tokens":5}}`)
	})

	model := genkit.LookupModel(g, modelName)
	require.NotNil(t, model)

	req := ai.NewModelRequest(nil, ai.NewUserMessage(ai.NewTextPart("say hi")))
	resp, err := model.Generate(context.Background(), req, nil)
	require.NoError(t, err)
	assert.Equal(t, "full reply", resp.Text())
}

func TestGenerate_Streaming_ServerError(t *testing.T) {
	g, modelName := newStreamTestGenkit(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	})

	model := genkit.LookupModel(g, modelName)
	require.NotNil(t, model)

	req := ai.NewModelRequest(nil, ai.NewUserMessage(ai.NewTextPart("say hi")))
	_, err := model.Generate(context.Background(), req, func(context.Context, *ai.ModelResponseChunk) error {
		return nil
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "openai")
}
