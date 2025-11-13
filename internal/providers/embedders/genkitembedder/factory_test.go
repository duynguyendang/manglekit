package genkitembedder

import (
"testing"

"github.com/duynguyendang/manglekit"
"github.com/duynguyendang/manglekit/core"
"github.com/duynguyendang/manglekit/embed"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func TestRegister_Success(t *testing.T) {
r := manglekit.NewRegistry()
err := Register(r)
require.NoError(t, err, "Register should not fail")
}

func TestProviderName(t *testing.T) {
opts := &embed.GenkitEmbedderOptions{
Provider: "openai",
Model:    "text-embedding-3-small",
}

assert.Equal(t, "genkit-embedder", opts.ProviderName())
}

func TestProviderKind(t *testing.T) {
opts := &embed.GenkitEmbedderOptions{
Provider: "openai",
Model:    "text-embedding-3-small",
}

assert.Equal(t, core.KindEmbedder, opts.ProviderKind())
}

func TestOptions_Fields(t *testing.T) {
tests := []struct {
name     string
provider string
model    string
apiKey   string
baseURL  string
}{
{
name:     "OpenAI",
provider: "openai",
model:    "text-embedding-3-small",
apiKey:   "sk-test",
baseURL:  "https://api.openai.com",
},
{
name:     "Google",
provider: "google",
model:    "embedding-001",
apiKey:   "test-key",
baseURL:  "",
},
{
name:     "Groq",
provider: "groq",
model:    "nomic-embed-text",
apiKey:   "gsk-test",
baseURL:  "",
},
{
name:     "Cohere",
provider: "cohere",
model:    "embed-english-v3.0",
apiKey:   "test-key",
baseURL:  "",
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
opts := &embed.GenkitEmbedderOptions{
Provider: tt.provider,
Model:    tt.model,
APIKey:   tt.apiKey,
BaseURL:  tt.baseURL,
}

assert.Equal(t, tt.provider, opts.Provider)
assert.Equal(t, tt.model, opts.Model)
assert.Equal(t, tt.apiKey, opts.APIKey)
assert.Equal(t, tt.baseURL, opts.BaseURL)
})
}
}

func TestProviderConfig_Map(t *testing.T) {
opts := &embed.GenkitEmbedderOptions{
Provider: "openai",
Model:    "text-embedding-3-small",
ProviderConfig: map[string]interface{}{
"timeout": 30,
"retries": 3,
},
}

assert.Equal(t, 30, opts.ProviderConfig["timeout"])
assert.Equal(t, 3, opts.ProviderConfig["retries"])
}
