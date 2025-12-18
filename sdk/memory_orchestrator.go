package sdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
)

// HybridMemory implements core.AgentMemory by combining:
// 1. HistoryStore (Sequential Chat Logs)
// 2. VectorStore (Semantic Search / RAG)
// HybridMemory implements core.AgentMemory by combining:
// 1. HistoryStore (Sequential Chat Logs)
// 2. VectorStore (Semantic Search / RAG)
type HybridMemory struct {
	History  core.HistoryStore
	Vectors  core.VectorStore
	Embedder core.Embedder
}

// NewHybridMemory creates a new memory orchestrator.
func NewHybridMemory(h core.HistoryStore, v core.VectorStore, e core.Embedder) *HybridMemory {
	return &HybridMemory{
		History:  h,
		Vectors:  v,
		Embedder: e,
	}
}

// Ensure HybridMemory implements core.AgentMemory
var _ core.AgentMemory = (*HybridMemory)(nil)

// --- Sequential History ---

// Read retrieves the chat history for a given session.
func (m *HybridMemory) Read(ctx context.Context, sessionID string) ([]core.Message, error) {
	if m.History == nil {
		return nil, nil
	}
	return m.History.Read(ctx, sessionID)
}

// Append adds new messages to the history.
func (m *HybridMemory) Append(ctx context.Context, sessionID string, msgs []core.Message) error {
	if m.History == nil {
		return nil
	}
	return m.History.Append(ctx, sessionID, msgs)
}

// --- Semantic Memory (RAG) ---

// Recall retrieves relevant context based on the current query.
func (m *HybridMemory) Recall(ctx context.Context, query string) (string, error) {
	if m.Vectors == nil || m.Embedder == nil {
		return "", nil
	}

	// 1. Embed Query
	vec, err := m.Embedder.Embed(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to embed query: %w", err)
	}

	// 2. Search Vector DB
	docs, err := m.Vectors.Search(ctx, "default", vec, 3) // Top 3
	if err != nil {
		return "", fmt.Errorf("failed to search vectors: %w", err)
	}

	if len(docs) == 0 {
		return "", nil
	}

	// 3. Format Context
	var sb strings.Builder
	for _, doc := range docs {
		sb.WriteString(doc.Content)
		sb.WriteString("\n---\n")
	}

	return sb.String(), nil
}

// Memorize stores a new interaction (Input/Output) for future recall.
func (m *HybridMemory) Memorize(ctx context.Context, query string, answer string) error {
	if m.Vectors == nil || m.Embedder == nil {
		return nil
	}

	content := fmt.Sprintf("Q: %s\nA: %s", query, answer)

	// 1. Embed Content
	vec, err := m.Embedder.Embed(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to embed content: %w", err)
	}

	// 2. Upsert to Vector DB
	doc := core.Document{
		ID:       fmt.Sprintf("%d", len(content)), // Simple ID, would imply hashing in real sys
		Content:  content,
		Vector:   vec,
		Metadata: map[string]any{"type": "chat"},
	}

	return m.Vectors.Upsert(ctx, "default", []core.Document{doc})
}

// Init performs any necessary setup.
func (m *HybridMemory) Init(ctx context.Context) error {
	return nil
}
