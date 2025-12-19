package actions

import (
	"fmt"

	"github.com/duynguyendang/manglekit/adapters/extractor"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/examples/proposalgpt/internal/types"
)

// NewExtractor creates an action that extracts RFP facts.
func NewExtractor(llm core.Action) (core.Action, error) {
	// We extract into types.ExtractedFacts
	action, err := extractor.New("rfp_extractor", llm, types.ExtractedFacts{})
	if err != nil {
		return nil, fmt.Errorf("failed to create extractor: %w", err)
	}
	return action, nil
}
