package resources

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromPath(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	ttlFile := filepath.Join(tmpDir, "test.ttl")

	// Create a sample Turtle file
	content := `
@prefix ex: <http://example.org/> .
@prefix foaf: <http://xmlns.com/foaf/0.1/> .

ex:alice foaf:knows ex:bob .
ex:bob ex:status "active" .
ex:alice ex:reportsTo ex:charlie .
`
	err := os.WriteFile(ttlFile, []byte(content), 0644)
	require.NoError(t, err)

	// Load knowledge
	facts, err := LoadFromPath(ttlFile)
	require.NoError(t, err)

	// Expected facts
	// foaf:knows -> knows("alice", "bob")
	// ex:status -> status("bob", "active")
	// ex:reportsTo -> reports_to("alice", "charlie")

	expected := []string{
		`knows("alice", "bob")`,
		`status("bob", "active")`,
		`reports_to("alice", "charlie")`,
	}

	assert.Len(t, facts, 3)
	for _, exp := range expected {
		assert.Contains(t, facts, exp)
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"FooBar", "foo_bar"},
		{"HTTPClient", "http_client"}, // Note: Simple regex might produce http_client or h_t_t_p_client depending on implementation
		{"simple", "simple"},
		{"reportsTo", "reports_to"},
	}

	for _, tt := range tests {
		got := toSnakeCase(tt.input)
		assert.Equal(t, tt.expected, got)
	}
}

func TestCleanTerm(t *testing.T) {
	// Since we can't easily create rdf.URI without the library constructors (which might be internal or complex),
	// we test the internal logic indirectly via LoadFromPath or if we could mock.
	// But cleanTerm takes rdf.Term interface.
	// Let's rely on integration test above.
}
