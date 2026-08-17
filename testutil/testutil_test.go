package testutil_test

import (
	"context"
	"sync"
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockLLM_ScriptedInOrder(t *testing.T) {
	ctx := context.Background()
	llm := testutil.NewMockLLM("first", "second")

	got, err := llm.Complete(ctx, "p1")
	require.NoError(t, err)
	assert.Equal(t, "first", got)

	resp, err := llm.Generate(ctx, "p2")
	require.NoError(t, err)
	assert.Equal(t, "second", resp.Text)
	assert.Equal(t, 2, resp.Usage["prompt"])

	// Script exhausted: last response repeats.
	got, err = llm.Complete(ctx, "p3")
	require.NoError(t, err)
	assert.Equal(t, "second", got)

	assert.Equal(t, 3, llm.Calls())
	assert.Equal(t, []string{"p1", "p2", "p3"}, llm.Prompts())
}

func TestMockLLM_Stream(t *testing.T) {
	llm := testutil.NewMockLLM("chunk text")
	ch, err := llm.Stream(context.Background(), "prompt")
	require.NoError(t, err)
	var text string
	for c := range ch {
		require.NoError(t, c.Err)
		text += c.Text
	}
	assert.Equal(t, "chunk text", text)
}

func TestMockLLM_EmptyScript(t *testing.T) {
	llm := testutil.NewMockLLM()
	got, err := llm.Complete(context.Background(), "anything")
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestMockLLM_Concurrent(t *testing.T) {
	llm := testutil.NewMockLLM("a", "b", "c")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := llm.Complete(context.Background(), "p")
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	assert.Equal(t, 20, llm.Calls())
}

func TestDeterministicEmbedder_Length(t *testing.T) {
	e := testutil.NewLengthEmbedder()
	assert.Equal(t, 2, e.Dimension())

	v1, err := e.Embed(context.Background(), "hello")
	require.NoError(t, err)
	v2, err := e.Embed(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, v1, v2, "same text must embed identically")

	v3, err := e.Embed(context.Background(), "hello world")
	require.NoError(t, err)
	assert.NotEqual(t, v1, v3, "different length should differ")

	batch, err := e.EmbedBatch(context.Background(), []string{"a", "bb"})
	require.NoError(t, err)
	assert.Len(t, batch, 2)
	assert.Equal(t, 2, len(batch[0]))
}

func TestDeterministicEmbedder_Keyword(t *testing.T) {
	e := testutil.NewKeywordEmbedder(
		map[string][]float32{
			"launch":    {0.9, 0.1},
			"Project X": {0.9, 0.1},
		},
		[]float32{0.1, 0.9},
	)
	hit, _ := e.Embed(context.Background(), "the launch codes")
	assert.InDeltaSlice(t, []float32{0.9, 0.1}, hit, 1e-6)

	miss, _ := e.Embed(context.Background(), "something else")
	assert.InDeltaSlice(t, []float32{0.1, 0.9}, miss, 1e-6)
}

func TestInMemoryStateProvider_CRUD(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewInMemoryStateProvider()

	raw, err := p.Get(ctx, "missing")
	require.NoError(t, err)
	assert.Nil(t, raw)

	state := &core.SessionState{SessionID: "s1", LogicalFacts: []string{"f(a)."}}
	require.NoError(t, p.Set(ctx, "s1", state))

	raw, err = p.Get(ctx, "s1")
	require.NoError(t, err)
	require.NotNil(t, raw)
	bytes, ok := raw.([]byte)
	require.True(t, ok, "Get must return JSON bytes, got %T", raw)

	var restored core.SessionState
	require.NoError(t, restored.UnmarshalJSON(bytes))
	assert.Equal(t, "s1", restored.SessionID)
	assert.Equal(t, []string{"f(a)."}, restored.LogicalFacts)

	require.NoError(t, p.Delete(ctx, "s1"))
	raw, err = p.Get(ctx, "s1")
	require.NoError(t, err)
	assert.Nil(t, raw)
	assert.Equal(t, 0, p.Size())
}

func TestInMemoryStateProvider_CloseResets(t *testing.T) {
	ctx := context.Background()
	p := testutil.NewInMemoryStateProvider()
	require.NoError(t, p.Set(ctx, "x", &core.SessionState{SessionID: "x"}))
	require.NoError(t, p.Close(ctx))
	raw, err := p.Get(ctx, "x")
	require.NoError(t, err)
	assert.Nil(t, raw)
}

func TestWorkflowSessionStore(t *testing.T) {
	ctx := context.Background()
	ss := testutil.NewWorkflowSessionStore()

	inst := core.NewWorkflowInstance("wf", "sess")
	require.NoError(t, ss.Create(ctx, inst))
	assert.True(t, ss.Exists(ctx, inst.SessionKey()))

	got, err := ss.Get(ctx, inst.SessionKey())
	require.NoError(t, err)
	assert.Equal(t, "wf", got.WorkflowID)

	inst.CurrentNodeID = "n2"
	require.NoError(t, ss.Update(ctx, inst))
	got, _ = ss.Get(ctx, inst.SessionKey())
	assert.Equal(t, "n2", got.CurrentNodeID)

	list, err := ss.List(ctx, "sess")
	require.NoError(t, err)
	assert.Len(t, list, 1)

	require.NoError(t, ss.ClearSession(ctx, "sess"))
	assert.False(t, ss.Exists(ctx, inst.SessionKey()))
}
