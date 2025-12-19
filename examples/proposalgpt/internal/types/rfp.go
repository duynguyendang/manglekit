package types

// ExtractedFacts represents the structured data extracted from an RFP.
type ExtractedFacts struct {
	Summary    string   `json:"summary"`
	Keywords   []string `json:"keywords"`
	Budget     float64  `json:"budget,omitempty"`
	Deadline   string   `json:"deadline,omitempty"`
	PainPoints []string `json:"pain_points"`
	CloudPref  string   `json:"cloud_pref,omitempty"`
	Compliance []string `json:"compliance,omitempty"`
}

// ProposalDraft represents the generated proposal content.
type ProposalDraft struct {
	Title            string `json:"title" mangle:"title"`
	ExecutiveSummary string `json:"executive_summary" mangle:"executive_summary"`
	Architecture     string `json:"architecture" mangle:"architecture"` // Selected strategy
	Platform         string `json:"platform" mangle:"platform"`         // Selected tech stack
	RiskAnalysis     string `json:"risk_analysis" mangle:"risk_analysis"`
	Implementation   string `json:"implementation" mangle:"implementation"`
	Content          string `json:"content" mangle:"proposal_content"` // Full markdown body
}
