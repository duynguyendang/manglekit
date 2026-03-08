package ports

import (
	"context"

	"github.com/duynguyendang/manglekit/sdk/ooda"
)

// ReasoningPort provides pure Datalog evaluation capabilities against the underlying graph store.
type ReasoningPort interface {
	// VerifyExecutes evaluates a Datalog query and returns structural Audit results.
	VerifyWithDatalog(ctx context.Context, datalogQuery string) ([]map[string]string, error)
}

// GenerativePort encapsulates interactions with Large Language Models.
type GenerativePort interface {
	// Generate produces structured output based on the assembled CognitiveFrame.
	Generate(ctx context.Context, frame *ooda.CognitiveFrame) (any, error)
}

// GenePoolPort handles fetching and mapping crystallized logic rules.
type GenePoolPort interface {
	// LoadActiveGenes fetches the relevant domain logic bounds for the current phase.
	LoadActiveGenes(ctx context.Context, frame *ooda.CognitiveFrame) ([]ooda.DomainGene, error)
}

// StoragePort guarantees persistence across the Hexagonal system.
type StoragePort interface {
	// SaveTrace records the final result of a CognitiveFrame epoch.
	SaveTrace(ctx context.Context, frame *ooda.CognitiveFrame) error
}
