package core

import (
	"context"
)

// MemoryMode defines the persistence strategy
type MemoryMode string

const (
	MemoryModeNone      MemoryMode = "none"      // Stateless (Default)
	MemoryModeTransient MemoryMode = "transient" // In-Memory (Loop scope only)
	MemoryModePersist   MemoryMode = "persist"   // Database backed
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MemoryStore is the interface for persistent storage (e.g., Redis)
type MemoryStore interface {
	Read(ctx context.Context, sessionID string) ([]ChatMessage, error)
	Write(ctx context.Context, sessionID string, msgs []ChatMessage) error
}

// NoOpStore for stateless mode
type NoOpStore struct{}

func (n NoOpStore) Read(_ context.Context, _ string) ([]ChatMessage, error) { return nil, nil }
func (n NoOpStore) Write(_ context.Context, _ string, _ []ChatMessage) error { return nil }
