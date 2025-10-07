package main

import (
	"testing"

	"github.com/duynguyendang/manglekit/core"
	"github.com/duynguyendang/manglekit/examples/04-chat-w-data/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoPolicy(t *testing.T) {
	// 1. Define the documents, matching the new main.go.
	docs := []core.Doc{
		{
			ID:   "A123",
			Text: "customer_name: Acme Corp, department: sales, confidentiality: normal, email: contact@acme.com, revenue: 100000, notes: Initial deal",
		},
		{
			ID:   "B456",
			Text: "lead_name: Globex Inc, department: marketing, confidentiality: high, email: sales@globex.inc, score: 95",
		},
		{
			ID:   "S777",
			Text: "account: Initech, department: sales, confidentiality: restricted, deal_size: 250000, owner: bsmith, notes: Q3 expansion plan",
		},
	}

	// 2. Define the user context, matching main.go.
	query := core.Query{
		Text: "Summarize the documents about sales and marketing",
		Meta: map[string]any{
			"user_attribute": []map[string]string{
				{"key": "user_id", "value": "alice"},
				{"key": "role", "value": "analyst"},
				{"key": "department", "value": "sales"},
				{"key": "doc_id", "value": "A123"},
				{"key": "purpose", "value": "analytics"},
			},
		},
	}

	// 3. Create a User struct from the query metadata.
	userMeta, ok := query.Meta["user_attribute"].([]map[string]string)
	require.True(t, ok, "invalid user_attribute format in query metadata")

	user := policy.User{}
	for _, attr := range userMeta {
		switch attr["key"] {
		case "role":
			user.Role = attr["value"]
		case "department":
			user.Department = attr["value"]
		case "doc_id":
			user.DocID = attr["value"]
		case "purpose":
			user.Purpose = attr["value"]
		}
	}

	// 4. Process documents using the Go policy.
	var retrievedDocs []core.Doc
	for _, doc := range docs {
		parsedDoc := policy.ParseDoc(doc.ID, doc.Text)
		if policy.CanRetrieve(user, parsedDoc) {
			retrievedDocs = append(retrievedDocs, doc)
		}
	}

	// 5. Assert that the correct document was retrieved.
	// The policy should only allow user "alice" to retrieve doc "A123".
	assert.Len(t, retrievedDocs, 1, "expected exactly one document to be retrieved")
	if len(retrievedDocs) == 1 {
		assert.Equal(t, "A123", retrievedDocs[0].ID, "expected to retrieve document A123")
	}
}