package redis

import (
	"context"
	"errors"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/core/diapi"
	"github.com/duynguyendang/manglekit/state"
	"github.com/redis/go-redis/v9"
)

func Register(r *manglekit.Registry) {
	manglekit.Register(r, &state.RedisOptions{},
		func(ctx context.Context, deps diapi.NoopDeps, cfg *state.RedisOptions) (core.StateProvider, error) {
			return New(ctx, *cfg)
		},
	)
}

// Provider is a production-ready implementation of the core.StateProvider that
// uses Redis as its backing store. It serializes state to JSON before saving.
type Provider struct {
	client *redis.Client
}

// New creates a new Redis state provider. It initializes a connection to the
// Redis server using the provided options and pings the server to ensure
// connectivity before returning the provider instance.
func New(ctx context.Context, opts state.RedisOptions) (core.StateProvider, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})

	// Verify the connection.
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Provider{client: rdb}, nil
}

// Get retrieves the state for a session ID from Redis. It fetches the raw bytes.
// If the key does not exist, it returns (nil, nil) as per the interface contract.
func (p *Provider) Get(ctx context.Context, sessionID string) (interface{}, error) {
	val, err := p.client.Get(ctx, sessionID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Key does not exist, not an error.
		}
		return nil, err
	}
	return val, nil
}

// Set saves the state for a session ID to Redis. It expects the state to be []byte.
func (p *Provider) Set(ctx context.Context, sessionID string, state interface{}) error {
	val, ok := state.([]byte)
	if !ok {
		return errors.New("redis provider expects state to be []byte")
	}
	return p.client.Set(ctx, sessionID, val, 0).Err()
}

// Delete removes the state for a session ID from Redis.
func (p *Provider) Delete(ctx context.Context, sessionID string) error {
	return p.client.Del(ctx, sessionID).Err()
}

// Close gracefully terminates the connection to the Redis server. This is a
// critical step to release network resources.
func (p *Provider) Close(ctx context.Context) error {
	return p.client.Close()
}
