package redis

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/duynguyendang/manglekit"
	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/state"
	"github.com/redis/go-redis/v9"
)

func init() {
	manglekit.RegisterStateProvider("redis", New)
	manglekit.RegisterOptions("redis", (*state.RedisOptions)(nil))
}

// Provider is a production-ready implementation of the core.StateProvider that
// uses Redis as its backing store. It serializes state to JSON before saving.
type Provider struct {
	client *redis.Client
}

// New creates a new Redis state provider. It initializes a connection to the
// Redis server using the provided options and pings the server to ensure
// connectivity before returning the provider instance.
func New(opts state.RedisOptions) (core.StateProvider, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})

	// Verify the connection.
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Provider{client: rdb}, nil
}

// Get retrieves the state for a session ID from Redis. It fetches the JSON
// string and unmarshals it into an `interface{}`. If the key does not exist,
// it returns (nil, nil) as per the interface contract.
func (p *Provider) Get(ctx context.Context, sessionID string) (interface{}, error) {
	val, err := p.client.Get(ctx, sessionID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Key does not exist, not an error.
		}
		return nil, err
	}

	var state interface{}
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return nil, err
	}
	return state, nil
}

// Set saves the state for a session ID to Redis. It marshals the state object
// into a JSON string before storing it.
func (p *Provider) Set(ctx context.Context, sessionID string, state interface{}) error {
	val, err := json.Marshal(state)
	if err != nil {
		return err
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
