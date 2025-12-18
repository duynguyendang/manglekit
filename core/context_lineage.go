package core

import "context"

type lineageKey struct{}

// WithParentID injects the ID of the parent data into the context.
// This is used to track the genealogy of data as it flows through the system,
// enabling lineage tracking and debugging.
//
// Parameters:
//   - ctx: The parent context.
//   - id: The unique ID of the parent data envelope or span.
//
// Returns:
//   - A new context containing the parent ID.
func WithParentID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, lineageKey{}, id)
}

// GetParentID retrieves the ID of the parent data from the context.
//
// Parameters:
//   - ctx: The context to search.
//
// Returns:
//   - The parent ID string and true if found, or empty string and false if not.
func GetParentID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(lineageKey{}).(string)
	return id, ok
}
