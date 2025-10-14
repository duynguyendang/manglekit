package core

import (
	"context"
)

// StateProvider defines the interface for persisting and retrieving session state.
// Implementations must be safe for concurrent use.
type StateProvider interface {
	// Get retrieves the state for a given session ID.
	// If no state is found, it should return (nil, nil).
	Get(ctx context.Context, sessionID string) (interface{}, error)

	// Set saves the state for a given session ID. The state will
	// overwrite any existing state for that session.
	Set(ctx context.Context, sessionID string, state interface{}) error

	// Delete removes the state for a given session ID.
	Delete(ctx context.Context, sessionID string) error

	// Close releases any resources associated with the provider.
	Close(ctx context.Context) error
}
