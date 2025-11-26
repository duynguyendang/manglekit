package core

import "os"

// ProviderDependency describes what environment variables and conditions are required for a provider.
type ProviderDependency struct {
	// Name is the provider's registration name (e.g., "google", "openai")
	Name string

	// Kind is the component kind this provider handles (e.g., KindLLM, KindRetriever)
	Kind Kind

	// RequiredEnvVars lists environment variables that must be set for this provider to work.
	// If multiple vars are listed, at least one must be set (OR logic).
	// Empty list means no environment variables are required.
	RequiredEnvVars []string

	// Description explains what the provider does (for error messages)
	Description string
}

// ProviderDependencyRegistry maps provider names to their dependencies.
// This allows the builder to validate that required configurations are in place.
type ProviderDependencyRegistry struct {
	dependencies map[string]*ProviderDependency
}

// NewProviderDependencyRegistry creates a new registry with standard providers.
func NewProviderDependencyRegistry() *ProviderDependencyRegistry {
	return &ProviderDependencyRegistry{
		dependencies: map[string]*ProviderDependency{
			// LLM Providers
			"google": {
				Name:            "google",
				Kind:            KindLLM,
				RequiredEnvVars: []string{"GOOGLE_API_KEY"},
				Description:     "Google Gemini LLM provider",
			},
			"openai": {
				Name:            "openai",
				Kind:            KindLLM,
				RequiredEnvVars: []string{"OPENAI_API_KEY"},
				Description:     "OpenAI LLM provider (ChatGPT, GPT-4, etc.)",
			},

			// Retriever Providers (no special requirements)
			"bm25": {
				Name:            "bm25",
				Kind:            KindRetriever,
				RequiredEnvVars: []string{}, // No env vars needed
				Description:     "BM25 keyword-based retriever",
			},
			"hybrid": {
				Name:            "hybrid",
				Kind:            KindRetriever,
				RequiredEnvVars: []string{}, // No env vars needed (uses embedded models)
				Description:     "Hybrid retriever combining BM25 and semantic search",
			},

			// Rules Providers (no special requirements)
			"mangle": {
				Name:            "mangle",
				Kind:            KindRules,
				RequiredEnvVars: []string{}, // No env vars needed
				Description:     "Mangle declarative rules engine",
			},

			// State Providers (no special requirements)
			"inmemory": {
				Name:            "inmemory",
				Kind:            KindStateProvider,
				RequiredEnvVars: []string{}, // No env vars needed
				Description:     "In-memory session state provider",
			},

			// Orchestrator Providers (no special requirements)
			"sandwich": {
				Name:            "sandwich",
				Kind:            KindOrchestrator,
				RequiredEnvVars: []string{}, // No env vars needed (dependencies are checked separately)
				Description:     "Sandwich RAG orchestrator pipeline",
			},
		},
	}
}

// Register adds or updates a provider's dependency information.
func (r *ProviderDependencyRegistry) Register(dep *ProviderDependency) {
	if r.dependencies == nil {
		r.dependencies = make(map[string]*ProviderDependency)
	}
	r.dependencies[dep.Name] = dep
}

// GetDependency retrieves a provider's dependency information.
func (r *ProviderDependencyRegistry) GetDependency(providerName string) *ProviderDependency {
	if r.dependencies == nil {
		return nil
	}
	return r.dependencies[providerName]
}

// ValidateProvider checks if a provider has all required environment variables set.
// Returns:
//   - nil if all requirements are met
//   - error describing what's missing if requirements are not met
func (r *ProviderDependencyRegistry) ValidateProvider(providerName string) error {
	dep := r.GetDependency(providerName)
	if dep == nil {
		// Provider not in registry - might be custom provider, allow it
		return nil
	}

	if len(dep.RequiredEnvVars) == 0 {
		// No requirements
		return nil
	}

	// Check if at least one required env var is set
	for _, envVar := range dep.RequiredEnvVars {
		if os.Getenv(envVar) != "" {
			return nil // At least one is set
		}
	}

	// None are set - return error
	return newProviderDependencyError(dep)
}

// providerDependencyError describes what's missing for a provider.
type providerDependencyError struct {
	Provider string
	Missing  []string
	Kind     Kind
}

func newProviderDependencyError(dep *ProviderDependency) *providerDependencyError {
	return &providerDependencyError{
		Provider: dep.Name,
		Missing:  dep.RequiredEnvVars,
		Kind:     dep.Kind,
	}
}

func (e *providerDependencyError) Error() string {
	if len(e.Missing) == 1 {
		return "missing required environment variable for " + string(e.Kind) +
			" provider '" + e.Provider + "': " + e.Missing[0]
	}

	// Multiple options - show as "one of X, Y, or Z"
	msg := "missing required environment variable for " + string(e.Kind) +
		" provider '" + e.Provider + "'. Set one of:"
	for i, v := range e.Missing {
		if i == 0 {
			msg += " " + v
		} else if i == len(e.Missing)-1 {
			msg += ", or " + v
		} else {
			msg += ", " + v
		}
	}
	return msg
}

// Is implements error.Is for type assertion.
func (e *providerDependencyError) Is(target error) bool {
	_, ok := target.(*providerDependencyError)
	return ok
}
