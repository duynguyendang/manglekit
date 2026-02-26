package proposer

import (
	"context"

	"github.com/duynguyendang/manglekit-wip/internal/core/domain"
	"github.com/duynguyendang/manglekit-wip/internal/core/ports"
)

// Service manages the "Orient" phase of the biological OODA loop by recalling
// multi-stream context (Semantic Vectors + Hard N-Quads Facts).
// Documented in HLD 5.6: Tri-Stream Context Recall.
type Service struct {
	llm      ports.GenerativePort
	vector   ports.VectorStorePort
	storage  ports.GenomeStoragePort
	embedder ports.EmbeddingPort
}

// NewService creates a properly initialized Proposer.
func NewService(llm ports.GenerativePort, vector ports.VectorStorePort, storage ports.GenomeStoragePort, embedder ports.EmbeddingPort) *Service {
	return &Service{llm: llm, vector: vector, storage: storage, embedder: embedder}
}

// Generate fulfills the GenerativePort adapter while mixing in recalled context.
func (s *Service) Generate(ctx context.Context, intent domain.IntentStr, compiledPrompt string, baseContext []domain.Atom, genes []domain.DomainGene) (*ports.Plan, error) {

	// 1. Semantic Stream (Vector Chunk Recall)
	queryVector, err := s.embedder.Embed(ctx, string(intent))
	if err == nil {
		// limit = 5 as per HLD 5.6
		matches, _ := s.vector.Search(queryVector, 5, 0.75)
		for _, match := range matches {
			baseContext = append(baseContext, domain.Atom{
				Predicate:    "semantic_similarity",
				Subject:      "Intent",
				Object:       match,
				Weight:       0.8,
				OriginIntent: intent,
			})
		}
	}

	// 2. Fact Stream (N-Quads deterministic knowledge)
	// Fetches the pre-compiled N-Quads file for the specific intent boundary.
	nqPath := s.storage.ResolvePath("knowledge", string(intent))
	facts, err := s.storage.ReadFile(ctx, nqPath)
	if err == nil && len(facts) > 0 {
		baseContext = append(baseContext, domain.Atom{
			Predicate: "structural_fact",
			Subject:   "HardContext",
			Object:    string(facts),
			Weight:    1.0,
		})
	}

	// 3. Delegate to true LLM synthesis (Gemini Adapter) with the compiled prompt
	return s.llm.Generate(ctx, intent, compiledPrompt, baseContext, genes)
}

func (s *Service) Embed(ctx context.Context, text string) (ports.Vector, error) {
	return s.llm.Embed(ctx, text)
}
func (s *Service) Induce(ctx context.Context, input string) (string, error) {
	return s.llm.Induce(ctx, input)
}
