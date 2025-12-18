package core

import "context"

// StateProvider defines the interface for storing and retrieving arbitrary session state.
// Unlike MemoryStore (which is specific to chat history), StateProvider handles generic state data.
type StateProvider interface {
	// Get retrieves the state associated with a session ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - sessionID: The unique identifier for the session.
	//
	// Returns:
	//   - The state object (as any), or an error if retrieval fails.
	Get(ctx context.Context, sessionID string) (any, error)

	// Set stores the state for a session ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - sessionID: The unique identifier for the session.
	//   - state: The state object to store.
	//
	// Returns:
	//   - An error if the operation fails.
	Set(ctx context.Context, sessionID string, state any) error

	// Delete removes the state for a session ID.
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//   - sessionID: The unique identifier for the session.
	//
	// Returns:
	//   - An error if the operation fails.
	Delete(ctx context.Context, sessionID string) error

	// Close cleans up any resources held by the provider (e.g., database connections).
	//
	// Parameters:
	//   - ctx: Context for cancellation.
	//
	// Returns:
	//   - An error if cleanup fails.
	Close(ctx context.Context) error
}
