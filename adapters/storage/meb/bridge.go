// Package meb provides a MEB (Mangle Extension for Badger) storage adapter
// that implements the KnowledgeStore interface for the OODA loop.
//
// This adapter bridges the MEB quad store to Manglekit's core types,
// providing zero-copy fact streaming and graph-scoped queries.
package meb

import (
	"context"
	"fmt"
	"iter"
	"sync"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

// MEBFact represents the fact type from MEB store.
// This is a type alias to ensure proper type matching.
type MEBFact interface {
	GetSubject() string
	GetPredicate() string
	GetObject() any
	GetGraph() string
}

// mebFactAdapter wraps meb.Fact to implement MEBFact interface.
type mebFactAdapter struct {
	Subject   string
	Predicate string
	Object    any
	Graph     string
}

func (m *mebFactAdapter) GetSubject() string   { return m.Subject }
func (m *mebFactAdapter) GetPredicate() string { return m.Predicate }
func (m *mebFactAdapter) GetObject() any       { return m.Object }
func (m *mebFactAdapter) GetGraph() string     { return m.Graph }

// GraphID scopes queries to a specific knowledge tier/graph.
// Common graph IDs:
//   - "t0": Immutable core axioms (Tier 0)
//   - "t1": Governance/human operator rules (Tier 1)
//   - "t2": Induced/learned logic (Tier 2)
//   - "t3": Untrusted external input (Tier 3)
//   - "playbook_v1": Specific playbook knowledge
const (
	GraphIDDefault = "default"
	GraphIDT0      = "t0"
	GraphIDT1      = "t1"
	GraphIDT2      = "t2"
	GraphIDT3      = "t3"
)

// KnowledgeBridge adapts MEBStore to satisfy the ports.KnowledgeStore interface.
// It provides graph-scoped queries and zero-copy fact streaming.
type KnowledgeBridge struct {
	store interface {
		Scan(s, p, o, g string) iter.Seq2[MEBFact, error]
	}
	dict interface {
		GetID(key string) (uint64, error)
		GetString(id uint64) (string, error)
	}
	graphID string
	mu      sync.RWMutex
}

// NewKnowledgeBridge creates a new MEB-backed KnowledgeStore.
// The store parameter must implement the MEB query interface returning MEBFact.
func NewKnowledgeBridge(store interface {
	Scan(s, p, o, g string) iter.Seq2[MEBFact, error]
}, dict interface {
	GetID(key string) (uint64, error)
	GetString(id uint64) (string, error)
}, graphID string) *KnowledgeBridge {
	if graphID == "" {
		graphID = GraphIDDefault
	}
	return &KnowledgeBridge{
		store:   store,
		dict:    dict,
		graphID: graphID,
	}
}

// Compile-time interface check
var _ ports.KnowledgeStore = (*KnowledgeBridge)(nil)

// Recall retrieves the top-K most relevant facts for a query.
// Since MEB doesn't support semantic search, this implements a simple
// subject-prefix match. For vector similarity, use MEB's vector search separately.
//
// The graphID parameter scopes the search to a specific knowledge tier/graph.
func (b *KnowledgeBridge) Recall(ctx context.Context, query string, topK int, graphID string) ([]core.Atom, error) {
	if graphID == "" {
		graphID = b.graphID
	}

	var atoms []core.Atom
	count := 0

	// Get facts matching query as subject prefix, limited by topK
	for fact, err := range b.store.Scan(query, "", "", graphID) {
		if err != nil {
			break
		}
		if count >= topK {
			break
		}
		atoms = append(atoms, b.toAtom(fact))
		count++
	}

	// If not enough, also scan for predicate matches
	if count < topK {
		for fact, err := range b.store.Scan("", query, "", graphID) {
			if err != nil {
				break
			}
			if count >= topK {
				break
			}
			f := b.toAtom(fact)
			// Avoid duplicates
			if !containsAtom(atoms, f) {
				atoms = append(atoms, f)
				count++
			}
		}
	}

	return atoms, nil
}

// GetFacts retrieves facts matching a specific pattern (subject/predicate/object).
// The graphID parameter scopes the search to a specific knowledge tier/graph.
func (b *KnowledgeBridge) GetFacts(ctx context.Context, subject, predicate, object, graphID string) ([]core.Quad, error) {
	if graphID == "" {
		graphID = b.graphID
	}

	var quads []core.Quad

	for fact, err := range b.store.Scan(subject, predicate, object, graphID) {
		if err != nil {
			break
		}
		quads = append(quads, b.toQuad(fact))
	}

	return quads, nil
}

// StreamFacts streams facts matching a pattern using zero-copy iteration.
// The graphID parameter scopes the search to a specific knowledge tier/graph.
// This uses iter.Seq2 for memory-efficient streaming.
func (b *KnowledgeBridge) StreamFacts(ctx context.Context, subject, predicate, object, graphID string) func(func(core.Quad) bool) {
	if graphID == "" {
		graphID = b.graphID
	}

	return func(yield func(core.Quad) bool) {
		for fact, err := range b.store.Scan(subject, predicate, object, graphID) {
			if err != nil {
				return
			}
			if !yield(b.toQuad(fact)) {
				return
			}
		}
	}
}

// WithGraphID returns a new bridge with the specified default graph ID.
func (b *KnowledgeBridge) WithGraphID(graphID string) *KnowledgeBridge {
	b.mu.Lock()
	defer b.mu.Unlock()
	bridge := *b
	bridge.graphID = graphID
	return &bridge
}

// ToMangleFact converts a MEB fact (Quad: S, P, O, G) to core.Fact format.
// This is the main entry point for mapping MEB data to Manglekit facts.
func (b *KnowledgeBridge) ToMangleFact(fact MEBFact) core.Atom {
	return b.toAtom(fact)
}

// ToMangleQuad converts a MEB fact to core.Quad format with Graph.
func (b *KnowledgeBridge) ToMangleQuad(fact MEBFact) core.Quad {
	return b.toQuad(fact)
}

// toAtom converts an MEB fact to a core.Atom (Subject-Predicate-Object only).
func (b *KnowledgeBridge) toAtom(fact MEBFact) core.Atom {
	return core.Atom{
		Subject:   fact.GetSubject(),
		Predicate: fact.GetPredicate(),
		Object:    fmt.Sprintf("%v", fact.GetObject()),
	}
}

// toQuad converts an MEB fact to a core.Quad (Subject-Predicate-Object-Graph).
func (b *KnowledgeBridge) toQuad(fact MEBFact) core.Quad {
	return core.Quad{
		Subject:   fact.GetSubject(),
		Predicate: fact.GetPredicate(),
		Object:    fmt.Sprintf("%v", fact.GetObject()),
		Graph:     fact.GetGraph(),
	}
}

// containsAtom checks if an atom already exists in the slice.
func containsAtom(atoms []core.Atom, atom core.Atom) bool {
	for _, a := range atoms {
		if a.Subject == atom.Subject && a.Predicate == atom.Predicate && a.Object == atom.Object {
			return true
		}
	}
	return false
}
