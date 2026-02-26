package domain

// Quad represents a discrete Subject-Predicate-Object-Graph truth used in persistent storage.
type Quad struct {
	Subject   string // Source entity (e.g., "User:Bob")
	Predicate string // Relationship (e.g., "has_role")
	Object    string // Target value (e.g., "Admin", "42")
	Graph     string // Namespace/Context (e.g., "global")
}
