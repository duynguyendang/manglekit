package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T, syncWrites bool) *Store {
	t.Helper()
	dir, err := os.MkdirTemp("", "manglekit-graph-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	s, err := NewStoreWithOptions(filepath.Join(dir, "store"), false, syncWrites)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func TestAddFacts_PersistsAllQuadsAndCounts(t *testing.T) {
	s := newTestStore(t, false)

	quads := make([]Quad, 100)
	for i := range quads {
		quads[i] = Quad{S: uint64(i + 1), P: 10, O: uint64(i + 1000), G: 1}
	}
	require.NoError(t, s.AddFacts(quads))

	assert.Equal(t, uint64(100), s.NumFacts())

	// Spot-check via SPOg scan that the first quad is present.
	found := false
	err := s.Scan(PrefixSPOg, EncodeQuadKey(PrefixSPOg, 1, 10, 1000, 1), 1, func(s_, p, o, g uint64) bool {
		found = s_ == 1 && p == 10 && o == 1000 && g == 1
		return false
	})
	require.NoError(t, err)
	assert.True(t, found, "batched quad must be readable via Scan")
}

func TestAddFacts_DuplicatesDoNotInflateCount(t *testing.T) {
	s := newTestStore(t, false)

	quads := []Quad{{S: 1, P: 2, O: 3, G: 4}, {S: 1, P: 2, O: 3, G: 4}, {S: 1, P: 2, O: 3, G: 4}}
	require.NoError(t, s.AddFacts(quads))
	assert.Equal(t, uint64(1), s.NumFacts(), "duplicate quads in one batch count once")

	// Re-adding the same batch is also idempotent.
	require.NoError(t, s.AddFacts(quads))
	assert.Equal(t, uint64(1), s.NumFacts())

	// Mixed new + old counts only the new ones.
	require.NoError(t, s.AddFacts([]Quad{{S: 1, P: 2, O: 3, G: 4}, {S: 9, P: 9, O: 9, G: 9}}))
	assert.Equal(t, uint64(2), s.NumFacts())
}

func TestAddFacts_Empty(t *testing.T) {
	s := newTestStore(t, false)
	require.NoError(t, s.AddFacts(nil))
	require.NoError(t, s.AddFacts([]Quad{}))
	assert.Equal(t, uint64(0), s.NumFacts())
}

func TestNewStoreWithOptions_SyncWrites(t *testing.T) {
	// The durable variant must round-trip facts (functionally identical
	// in-process; the sync stance is passed to Badger).
	s := newTestStore(t, true)
	require.NoError(t, s.AddFact(1, 2, 3, 4))
	require.NoError(t, s.AddFacts([]Quad{{S: 5, P: 6, O: 7, G: 8}}))
	assert.Equal(t, uint64(2), s.NumFacts())
}

func BenchmarkAddFacts10k(b *testing.B) {
	dir, err := os.MkdirTemp("", "manglekit-graph-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	s, err := NewStoreWithOptions(filepath.Join(dir, "store"), false, false)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()

	quads := make([]Quad, 10000)
	for i := range quads {
		quads[i] = Quad{S: uint64(i + 1), P: 42, O: uint64(i), G: 7}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := s.AddFacts(quads); err != nil {
			b.Fatal(err)
		}
	}
}
