package sdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/duynguyendang/manglekit/config"
	"github.com/duynguyendang/manglekit/core"
)

// ProviderFactory defines the constructor signature for creating an action from config.
type ProviderFactory func(ctx context.Context, name string, cfg config.ActionConfig) (core.Action, error)

var (
	registryMu       sync.RWMutex
	providerRegistry = make(map[string]ProviderFactory)
)

// RegisterProvider allows external packages (plugins) to register their factories.
// Example: sdk.RegisterProvider("google", GoogleFactory)
func RegisterProvider(name string, factory ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	providerRegistry[name] = factory
}

// Provider retrieves a registered factory.
func Provider(name string) (ProviderFactory, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	factory, ok := providerRegistry[name]
	if !ok {
		return nil, fmt.Errorf("provider '%s' is not registered", name)
	}
	return factory, nil
}

// WithProviderConfig creates a ClientOption that initializes a provider from configuration.
// It looks up the factory, creates the action, supervises it, and registers it to the client.
func WithProviderConfig(name string, cfg config.ActionConfig) ClientOption {
	return func(c *Client) error {
		registryMu.RLock()
		factory, ok := providerRegistry[cfg.Provider]
		registryMu.RUnlock()

		if !ok {
			// If fail_on_startup is true, we error out.
			if cfg.FailOnStartup {
				return fmt.Errorf("provider factory '%s' not found for action '%s'", cfg.Provider, name)
			}
			// Otherwise log warning and skip
			c.logger.Warn("provider factory not found", "provider", cfg.Provider, "action", name)
			return nil
		}

		// Create the action using the factory.
		// Note: We use context.Background() here because ClientOption doesn't accept context.
		// This implies providers should not rely on the context for long-lived cancellation during init.
		action, err := factory(context.Background(), name, cfg)
		if err != nil {
			if cfg.FailOnStartup {
				return fmt.Errorf("failed to create action '%s': %w", name, err)
			}
			c.logger.Warn("failed to create action", "name", name, "error", err)
			return nil
		}

		// Supervise the action (apply governance)
		supervised := c.Supervise(action)

		// Register to the client
		c.registry[name] = supervised

		// If this is an LLM and we don't have a default LLM yet, set it.
		if cfg.Type == "llm" && c.llm == nil {
			if gen, ok := action.(core.TextGenerator); ok {
				c.llm = gen
			}
		}

		return nil
	}
}

// MemoryFactory defines the constructor for memory providers.
type MemoryFactory func(ctx context.Context, cfg config.MemoryConfig) (core.AgentMemory, error)

var (
	memoryRegistryMu sync.RWMutex
	memoryRegistry   = make(map[string]MemoryFactory)
)

// RegisterMemoryProvider allows plugins to register memory backends (e.g., "qdrant", "simple").
func RegisterMemoryProvider(name string, factory MemoryFactory) {
	memoryRegistryMu.Lock()
	defer memoryRegistryMu.Unlock()
	memoryRegistry[name] = factory
}

// MemoryProvider retrieves a registered memory factory.
func MemoryProvider(name string) (MemoryFactory, error) {
	memoryRegistryMu.RLock()
	defer memoryRegistryMu.RUnlock()
	factory, ok := memoryRegistry[name]
	if !ok {
		return nil, fmt.Errorf("memory provider '%s' is not registered", name)
	}
	return factory, nil
}
