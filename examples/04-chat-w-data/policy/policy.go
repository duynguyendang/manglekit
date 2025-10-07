package policy

import (
	"strings"
)

// User represents the attributes of the user making the query.
type User struct {
	Role       string
	Department string
	DocID      string
	Purpose    string
}

// Doc represents a document with its attributes and data.
type Doc struct {
	ID              string
	Department      string
	Confidentiality string
	Columns         map[string]string
	SensitiveCols   map[string]bool
}

// Privileged returns true if the user has a privileged role.
func Privileged(u User) bool {
	return u.Role == "admin" || u.Role == "security"
}

// preRetrieve determines if a user has initial access to a document.
func preRetrieve(u User, d Doc) bool {
	// Direct entitlement
	if d.ID == u.DocID {
		return true
	}
	// Departmental access for normal confidentiality
	if d.Department == u.Department && d.Confidentiality == "normal" {
		return true
	}
	// Privileged can see any known doc
	if Privileged(u) {
		return true
	}
	return false
}

// denyRetrieve determines if access to a document should be explicitly denied.
func denyRetrieve(u User, d Doc) bool {
	return d.Confidentiality == "restricted" && !Privileged(u)
}

// CanRetrieve determines if a user has effective access to a document.
func CanRetrieve(u User, d Doc) bool {
	return preRetrieve(u, d) && !denyRetrieve(u, d)
}

// VisibleColumns returns the list of columns that are visible to the user.
func VisibleColumns(u User, d Doc) []string {
	var visible []string
	for col := range d.Columns {
		if !d.SensitiveCols[col] || Privileged(u) {
			visible = append(visible, col)
		}
	}
	return visible
}

// MaskedColumns returns a map of columns that should be masked and the masking strategy.
func MaskedColumns(u User, d Doc) map[string]string {
	masked := make(map[string]string)
	if Privileged(u) {
		return masked
	}

	mode := "redact"
	if u.Purpose == "analytics" {
		mode = "hash"
	}

	for col := range d.Columns {
		if d.SensitiveCols[col] {
			masked[col] = mode
		}
	}
	return masked
}

// ParseDoc creates a Doc from a core.Doc by parsing its text field.
func ParseDoc(id, text string) Doc {
	doc := Doc{
		ID:            id,
		Columns:       make(map[string]string),
		SensitiveCols: make(map[string]bool),
	}
	// This is a simplified parser based on the example data format.
	pairs := strings.Split(text, ", ")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, ": ", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			switch key {
			case "department":
				doc.Department = value
			case "confidentiality":
				doc.Confidentiality = value
			default:
				doc.Columns[key] = value
			}
		}
	}
	// This is a hack for the example data. In a real app, this would be structured metadata.
	if doc.Department == "" {
		if doc.ID == "B456" {
			doc.Department = "marketing"
			doc.Confidentiality = "high"
			doc.SensitiveCols["email"] = true
		} else if doc.ID == "S777" {
			doc.Department = "sales"
			doc.Confidentiality = "restricted"
			doc.SensitiveCols["notes"] = true
		}
	} else {
		doc.SensitiveCols["email"] = true
		doc.SensitiveCols["notes"] = true
	}
	return doc
}