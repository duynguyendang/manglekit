package resources

import _ "embed"

//go:embed planner.dl
var plannerRules string

// GetPlannerRules returns the core Datalog rules for the Logic Planner.
func GetPlannerRules() string {
	return plannerRules
}
