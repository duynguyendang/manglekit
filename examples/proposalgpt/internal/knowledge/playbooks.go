package knowledge

// Playbook represents a solution design strategy.
type Playbook struct {
	ID           string `mangle:"playbook_id"`
	Name         string
	Description  string
	CriticalNFRs []string `mangle:"nfrs"`
	Risks        []string
	ArchPattern  string `mangle:"arch_pattern"`
}

// LoadPlaybooks returns the available playbooks.
func LoadPlaybooks() map[string]Playbook {
	return map[string]Playbook{
		"modernization": {
			ID:          "modernization",
			Name:        "Legacy Modernization",
			Description: "Refactor legacy monoliths into microservices using the Strangler Fig pattern.",
			CriticalNFRs: []string{
				"Reliability",
				"Maintainability",
				"Scalability",
			},
			Risks: []string{
				"Big Bang Integration Failure",
				"Data Consistency during migration",
			},
			ArchPattern: "Strangler Fig",
		},
		"greenfield": {
			ID:          "greenfield",
			Name:        "Greenfield Cloud Native",
			Description: "Build a new application from scratch using cloud-native principles.",
			CriticalNFRs: []string{
				"Time to Market",
				"Agility",
			},
			Risks: []string{
				"Scope Creep",
				"Over-engineering",
			},
			ArchPattern: "Microservices",
		},
		"data_platform_mod": {
			ID:          "data_platform_mod",
			Name:        "Data Platform Modernization",
			Description: "Upgrade legacy data warehouses to modern Lakehouse architectures.",
			CriticalNFRs: []string{
				"Data Integrity",
				"Performance",
			},
			Risks: []string{
				"Data Loss",
				"Downtime",
			},
			ArchPattern: "Lakehouse",
		},
	}
}
