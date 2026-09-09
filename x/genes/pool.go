package genes

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/duynguyendang/manglekit/core"
	"gopkg.in/yaml.v3"
)

// manifest is the on-disk format of a gene pool.
type manifest struct {
	Version int            `yaml:"version"`
	Genes   []manifestGene `yaml:"genes"`
}

type manifestGene struct {
	Name      string    `yaml:"name"`
	Tier      core.Tier `yaml:"tier"`
	Source    string    `yaml:"source"`
	Intents   []string  `yaml:"intents"`
	Rules     string    `yaml:"rules"`
	Signature string    `yaml:"signature"`
}

// Pool is an immutable, signature-verified set of genes.
type Pool struct {
	genes []Gene
}

// Genes returns genes sorted by name (deterministic compile order).
func (p *Pool) Genes() []Gene {
	out := make([]Gene, len(p.genes))
	copy(out, p.genes)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Active returns genes matching an intent (a gene with no Intents matches all).
func (p *Pool) Active(intent string) []Gene {
	var out []Gene
	for _, g := range p.Genes() {
		if len(g.Intents) == 0 {
			out = append(out, g)
			continue
		}
		for _, in := range g.Intents {
			if in == intent {
				out = append(out, g)
				break
			}
		}
	}
	return out
}

// LoadPoolFile reads and validates a YAML manifest from path. Every gene must
// be present, signed, and signature-intact — a corrupted pool fails closed.
func LoadPoolFile(path string) (*Pool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("x/genes: open manifest: %w", err)
	}
	defer f.Close()
	return LoadPool(f)
}

// LoadPool reads and validates a YAML manifest from r.
func LoadPool(r io.Reader) (*Pool, error) {
	var m manifest
	if err := yaml.NewDecoder(r).Decode(&m); err != nil {
		if err == io.EOF {
			return &Pool{}, nil
		}
		return nil, fmt.Errorf("x/genes: parse manifest: %w", err)
	}
	if m.Version != 1 {
		return nil, fmt.Errorf("x/genes: unsupported manifest version %d (want 1)", m.Version)
	}
	seen := map[string]bool{}
	pool := &Pool{}
	for _, mg := range m.Genes {
		if mg.Name == "" || seen[mg.Name] {
			return nil, fmt.Errorf("x/genes: gene with empty or duplicate name %q", mg.Name)
		}
		seen[mg.Name] = true
		if !validTier(mg.Tier) {
			return nil, fmt.Errorf("x/genes: gene %q has invalid tier %q (want T0..T3)", mg.Name, mg.Tier)
		}
		if mg.Rules == "" {
			return nil, fmt.Errorf("x/genes: gene %q has no rules", mg.Name)
		}
		g := Gene{
			Name:    mg.Name,
			Tier:    mg.Tier,
			Source:  mg.Source,
			Intents: mg.Intents,
			Rules:   mg.Rules,
		}
		if mg.Signature == "" {
			return nil, fmt.Errorf("x/genes: gene %q is unsigned — run Gene.Sign() and write the manifest again", mg.Name)
		}
		if err := g.SetSignatureHex(mg.Signature); err != nil {
			return nil, fmt.Errorf("x/genes: gene %q: %w", g.Name, err)
		}
		if err := g.Verify(); err != nil {
			return nil, fmt.Errorf("x/genes: %w", err)
		}
		pool.genes = append(pool.genes, g)
	}
	return pool, nil
}

// WritePool serializes genes (each must be signed) to a manifest.
func WritePool(w io.Writer, genes []Gene) error {
	m := manifest{Version: 1}
	for _, g := range genes {
		if !g.Signed {
			return fmt.Errorf("x/genes: gene %q is not signed", g.Name)
		}
		m.Genes = append(m.Genes, manifestGene{
			Name:      g.Name,
			Tier:      g.Tier,
			Source:    g.Source,
			Intents:   g.Intents,
			Rules:     g.Rules,
			Signature: g.SignatureHex(),
		})
	}
	enc := yaml.NewEncoder(w)
	defer enc.Close()
	return enc.Encode(m)
}
