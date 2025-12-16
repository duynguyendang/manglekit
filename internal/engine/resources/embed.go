package resources

import _ "embed"

//go:embed planner.dl
var plannerRules string

//go:embed std.dl
var stdLib string

// GetPlannerRules returns the core Datalog rules for the Logic Planner.
func GetPlannerRules() string {
	return plannerRules
}

// StdLib returns the Manglekit Standard Library rules.
func StdLib() string {
	return stdLib
}
