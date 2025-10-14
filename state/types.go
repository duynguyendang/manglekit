package state

// InMemoryOptions configures the in-memory state provider. It is currently empty but
// is included for future extensibility and consistency with other providers.
type InMemoryOptions struct{}

// RedisOptions holds the configuration required to connect to a Redis server.
// It includes connection details like address, password, and database number.
type RedisOptions struct {
	Addr     string `json:"addr" yaml:"addr"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}
