package core

import "context"

type lineageKey struct{}

// WithParentID injects the ID of the data that triggered the current flow.
func WithParentID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, lineageKey{}, id)
}

// GetParentID retrieves the ID of the parent data.
func GetParentID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(lineageKey{}).(string)
	return id, ok
}
