package state

import "github.com/duynguyendang/manglekit/core"

// InMemoryOptions configures the in-memory state provider. It is currently empty but
// is included for future extensibility and consistency with other providers.
type InMemoryOptions struct{}

func (o *InMemoryOptions) ProviderName() string { return "in-memory" }
func (o *InMemoryOptions) ProviderKind() core.Kind   { return core.KindStateProvider }

// RedisOptions holds the configuration required to connect to a Redis server.
// It includes connection details like address, password, and database number.
type RedisOptions struct {
	Addr     string `json:"addr" yaml:"addr"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}

func (o *RedisOptions) ProviderName() string { return "redis" }
func (o *RedisOptions) ProviderKind() core.Kind   { return core.KindStateProvider }
