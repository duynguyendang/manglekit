package sdk

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/duynguyendang/manglekit/core"
)

// HybridMemory implements core.AgentMemory by combining:
// 1. HistoryStore (Sequential Chat Logs)
// 2. VectorStore (Semantic Search / RAG)
type HybridMemory struct {
	History        core.HistoryStore
	Vectors        core.VectorStore
	Embedder       core.Embedder // Kept for interface compatibility, but VectorStore handles embedding.
	CollectionName string        // Deprecated: VectorStore handles collections internally or ignores them.
	TopK           int           // Number of results to retrieve (default: 3)
}

// NewHybridMemory creates a new memory orchestrator with default RAG settings.
func NewHybridMemory(h core.HistoryStore, v core.VectorStore, e core.Embedder) *HybridMemory {
	return &HybridMemory{
		History:        h,
		Vectors:        v,
		Embedder:       e,
		CollectionName: "default",
		TopK:           3,
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
	if m.Vectors == nil {
		return "", nil
	}

	// 1. Search Vector DB
	topK := m.TopK
	if topK <= 0 {
		topK = 3
	}

	ids, err := m.Vectors.Search(ctx, query, topK)
	if err != nil {
		return "", fmt.Errorf("failed to search vectors: %w", err)
	}

	if len(ids) == 0 {
		return "", nil
	}

	// 2. Retrieve Content
	var sb strings.Builder
	for _, id := range ids {
		content, err := m.Vectors.Get(ctx, id)
		if err != nil {
			// If a document is missing, we skip it but log/warn if we had a logger.
			// For now, continue.
			continue
		}
		sb.WriteString(content)
		sb.WriteString("\n---\n")
	}

	return sb.String(), nil
}

// Memorize stores a new interaction (Input/Output) for future recall.
func (m *HybridMemory) Memorize(ctx context.Context, query string, answer string) error {
	if m.Vectors == nil {
		return nil
	}

	content := fmt.Sprintf("Q: %s\nA: %s", query, answer)

	// Generate a deterministic ID based on content hash to avoid collisions
	hash := sha256.Sum256([]byte(content))
	docID := fmt.Sprintf("%x", hash[:8]) // Use first 16 hex chars (8 bytes)

	return m.Vectors.Upsert(ctx, docID, content)
}

// Init performs any necessary setup.
func (m *HybridMemory) Init(ctx context.Context) error {
	return nil
}
