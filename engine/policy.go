package engine

import (
	"context"

	"github.com/duynguyendang/manglekit/v2/core"
)

type PolicyEngine struct {
}

func New() *PolicyEngine {
	return &PolicyEngine{}
}

func (e *PolicyEngine) Authorize(ctx context.Context, actionMeta core.ActionMetadata, input core.Envelope) error {
	return nil
}

func (e *PolicyEngine) Validate(ctx context.Context, actionMeta core.ActionMetadata, output core.Envelope) (core.Envelope, error) {
	return output, nil
}
