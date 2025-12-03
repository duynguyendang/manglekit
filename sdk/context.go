package sdk

import "context"

type factsKeyType struct{}

var factsKey = factsKeyType{}

// WithFact injects a key-value pair into the context.
// These facts are automatically extracted by the Reflector and made available
// to Datalog policies (e.g., user_role, system_mode).
func WithFact(ctx context.Context, key, value string) context.Context {
	m := ContextFacts(ctx)
	newM := make(map[string]string, len(m)+1)
	for k, v := range m {
		newM[k] = v
	}
	newM[key] = value
	return context.WithValue(ctx, factsKey, newM)
}

// ContextFacts retrieves all injected facts from the context.
func ContextFacts(ctx context.Context) map[string]string {
	if m, ok := ctx.Value(factsKey).(map[string]string); ok {
		return m
	}
	return nil
}
