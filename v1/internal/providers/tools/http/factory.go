package http

import (
	"context"
	"fmt"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
)

func NewFactory() func(ctx context.Context, deps diapi.CoreDeps, cfg Options) (core.Tool, error) {
	return func(ctx context.Context, deps diapi.CoreDeps, cfg Options) (core.Tool, error) {
		t, err := NewTool(deps, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create http tool: %w", err)
		}
		return t, nil
	}
}
