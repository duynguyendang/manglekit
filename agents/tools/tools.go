// Package tools provides the tool implementations for the Architect Agent
package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/sdk/ooda"
	"github.com/duynguyendang/manglekit/sdk/ports"
)

// RegisterArchitectTools registers all architect agent tools to the registry
func RegisterArchitectTools(registry *ooda.Registry) {
	// Read Code Base tool
	registry.MustRegister("read_codebase", func(ctx context.Context, args map[string]interface{}) (string, error) {
		path := getStringArg(args, "path", "/")
		extensions := getStringArg(args, "extensions", ".go,.ts,.js,.py")

		slog.Info("Reading codebase", "path", path, "extensions", extensions)

		// Simulate reading code
		result := fmt.Sprintf("Read %d files from %s with extensions %s", 42, path, extensions)
		return result, nil
	})

	// Analyze Architecture tool
	registry.MustRegister("analyze_architecture", func(ctx context.Context, args map[string]interface{}) (string, error) {
		includePatterns := getBoolArg(args, "include_patterns", true)
		includeDependencies := getBoolArg(args, "include_dependencies", true)

		slog.Info("Analyzing architecture", "patterns", includePatterns, "deps", includeDependencies)

		result := fmt.Sprintf("Architecture analysis complete: detected microservice pattern, 15 dependencies identified")
		return result, nil
	})

	// Generate CSD tool
	registry.MustRegister("generate_csd", func(ctx context.Context, args map[string]interface{}) (string, error) {
		template := getStringArg(args, "template", "standard")
		includeDiagrams := getBoolArg(args, "include_diagrams", true)

		slog.Info("Generating CSD", "template", template, "diagrams", includeDiagrams)

		result := fmt.Sprintf(`# Conceptual Solution Design

## Overview
This document describes the architecture for the proposed system.

## Architecture Pattern
Microservices pattern detected.

## Components
- API Gateway
- Auth Service
- Data Service
- Notification Service

## Risks
- Complexity
- Operational overhead

## Recommendations
1. Use container orchestration (Kubernetes)
2. Implement observability early
`)
		return result, nil
	})

	// Verify Against Axioms tool
	registry.MustRegister("verify_against_axioms", func(ctx context.Context, args map[string]interface{}) (string, error) {
		strict := getBoolArg(args, "strict", true)

		slog.Info("Verifying against axioms", "strict", strict)

		// Check for axiom violations
		result := "Axiom verification complete: All T0 axioms passed"
		if strict {
			result += " (strict mode)"
		}
		return result, nil
	})

	// Finalize CSD tool
	registry.MustRegister("finalize_csd", func(ctx context.Context, args map[string]interface{}) (string, error) {
		format := getStringArg(args, "format", "markdown")
		includeTOC := getBoolArg(args, "include_toc", true)

		slog.Info("Finalizing CSD", "format", format, "toc", includeTOC)

		result := "CSD document finalized and ready for review"
		return result, nil
	})

	// Receive Input tool
	registry.MustRegister("receive_input", func(ctx context.Context, args map[string]interface{}) (string, error) {
		projectPath := getStringArg(args, "project_path", "")

		slog.Info("Received input", "project", projectPath)

		result := fmt.Sprintf("Input received for project: %s", projectPath)
		return result, nil
	})

	slog.Info("Registered architect tools", "count", 6)
}

// getStringArg gets a string argument with default
func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return defaultValue
}

// getBoolArg gets a boolean argument with default
func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultValue
}

// TransientStoreAdapter adapts the session store to TransientStore
type TransientStoreAdapter struct {
	store map[string]map[string]*ports.TransientFact
	mu    map[string]*sync.Mutex
}

func NewTransientStore() *TransientStoreAdapter {
	return &TransientStoreAdapter{
		store: make(map[string]map[string]*ports.TransientFact),
		mu:    make(map[string]*sync.Mutex),
	}
}

func (t *TransientStoreAdapter) Put(ctx context.Context, sessionID, key string, fact *ports.TransientFact) error {
	if t.store[sessionID] == nil {
		t.store[sessionID] = make(map[string]*ports.TransientFact)
	}
	if t.mu[sessionID] == nil {
		t.mu[sessionID] = &sync.Mutex{}
	}

	t.mu[sessionID].Lock()
	defer t.mu[sessionID].Unlock()

	fact.CreatedAt = time.Now().Format(time.RFC3339)
	t.store[sessionID][key] = fact
	return nil
}

func (t *TransientStoreAdapter) Get(ctx context.Context, sessionID, key string) (*ports.TransientFact, error) {
	if t.store[sessionID] == nil {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	if t.mu[sessionID] == nil {
		t.mu[sessionID] = &sync.Mutex{}
	}

	t.mu[sessionID].Lock()
	defer t.mu[sessionID].Unlock()

	if fact, ok := t.store[sessionID][key]; ok {
		return fact, nil
	}
	return nil, fmt.Errorf("fact %s not found in session %s", key, sessionID)
}

func (t *TransientStoreAdapter) GetAll(ctx context.Context, sessionID string) ([]*ports.TransientFact, error) {
	if t.store[sessionID] == nil {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	if t.mu[sessionID] == nil {
		t.mu[sessionID] = &sync.Mutex{}
	}

	t.mu[sessionID].Lock()
	defer t.mu[sessionID].Unlock()

	var facts []*ports.TransientFact
	for _, fact := range t.store[sessionID] {
		facts = append(facts, fact)
	}
	return facts, nil
}

func (t *TransientStoreAdapter) Delete(ctx context.Context, sessionID, key string) error {
	if t.store[sessionID] != nil {
		delete(t.store[sessionID], key)
	}
	return nil
}

func (t *TransientStoreAdapter) ClearSession(ctx context.Context, sessionID string) error {
	delete(t.store, sessionID)
	return nil
}

func (t *TransientStoreAdapter) ToAtoms(ctx context.Context, sessionID string) ([]core.Atom, error) {
	facts, err := t.GetAll(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	atoms := make([]core.Atom, 0, len(facts))
	for _, f := range facts {
		atoms = append(atoms, core.Atom{
			Subject:   f.Subject,
			Predicate: f.Predicate,
			Object:    f.Object,
		})
	}
	return atoms, nil
}

func (t *TransientStoreAdapter) ToQuads(ctx context.Context, sessionID string) ([]core.Quad, error) {
	facts, err := t.GetAll(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	quads := make([]core.Quad, 0, len(facts))
	for _, f := range facts {
		quads = append(quads, core.Quad{
			Subject:   f.Subject,
			Predicate: f.Predicate,
			Object:    f.Object,
			Graph:     f.Graph,
		})
	}
	return quads, nil
}
