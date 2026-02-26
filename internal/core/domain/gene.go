package domain

// DomainGene represents a unit of crystallized logic with integrity validation.
type DomainGene struct {
	Name         string    `json:"name"`
	Tier         TrustTier `json:"tier"` // TIER_0 .. TIER_3
	TierID       string    `json:"tier_id"`
	Rules        []byte    `json:"rules"`     // Compiled Datalog content
	Signature    [32]byte  `json:"signature"` // SHA256 integrity hash
	MMapAddr     uintptr   `json:"-"`         // Zero-copy mmap pointer
	Capabilities []string  `json:"capabilities"`
	Intents      []string  `json:"intents"`
	FactPath     string    `json:"fact_path,omitempty"`
	SourcePath   string    `json:"source_path,omitempty"`
	IsUnverified bool      `json:"is_unverified"`
}

// GeneManifest defines the registry of authorized logic units loaded at boot.
// This is the root structure for manifest.yaml parsing.
type GeneManifest struct {
	Version string         `yaml:"version"`
	Genes   []GeneMetadata `yaml:"genes"`
}

// GeneMetadata contains the identity and integrity data for a single gene.
type GeneMetadata struct {
	Name         string    `yaml:"name"`
	Tier         TrustTier `yaml:"tier"`
	Path         string    `yaml:"path"`
	FactPath     string    `yaml:"fact_path,omitempty"`
	Signature    string    `yaml:"signature"` // SHA256 hex string
	Capabilities []string  `yaml:"capabilities"`
	Intents      []string  `yaml:"intents"`
}
