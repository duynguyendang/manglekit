package core

import "context"

// StateProvider defines the interface for storing and retrieving session state.
type StateProvider interface {
	Get(ctx context.Context, sessionID string) (any, error)
	Set(ctx context.Context, sessionID string, state any) error
	Delete(ctx context.Context, sessionID string) error
	Close(ctx context.Context) error
}
