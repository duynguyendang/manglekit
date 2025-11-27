package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/mangle/ast"
	"github.com/google/mangle/parse"
)

// LineageGraph stores the relationships between data items.
type LineageGraph struct {
	mu    sync.RWMutex
	links map[string]string // childID -> parentID
}

// NewLineageGraph creates a new, empty lineage graph.
func NewLineageGraph() *LineageGraph {
	return &LineageGraph{
		links: make(map[string]string),
	}
}

// RecordLineage records that childID is derived from parentID.
func (g *LineageGraph) RecordLineage(ctx context.Context, childID, parentID string) {
	if childID == "" || parentID == "" {
		// fmt.Printf("[LineageGraph] Skipping empty ID: Child=%q Parent=%q\n", childID, parentID)
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.links[childID] = parentID
}

// GetParent returns the parent ID for a given child ID, if it exists.
func (g *LineageGraph) GetParent(childID string) (string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	parentID, ok := g.links[childID]
	return parentID, ok
}

// ToFacts generates Datalog facts for the entire lineage graph.
func (g *LineageGraph) ToFacts() ([]ast.Atom, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var atoms []ast.Atom
	for child, parent := range g.links {
		factStr := fmt.Sprintf("derived_from(%q, %q)", child, parent)
		atom, err := parse.Atom(factStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse lineage fact: %w", err)
		}
		atoms = append(atoms, atom)
	}
	return atoms, nil
}
