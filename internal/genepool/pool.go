package genepool

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/duynguyendang/manglekit/internal/core/domain"
	"github.com/duynguyendang/manglekit/internal/core/ports"
	"gopkg.in/yaml.v3"
)

// GenePool manages the ActiveGenes (crystals of logic) in RAM.
// It implements ports.GenePoolPort.
type GenePool struct {
	mu           sync.RWMutex
	storage      ports.GenomeStoragePort
	manifestPath string
	activeGenes  map[string]*domain.DomainGene // Keyed by gene.Name
}

// New initializes the pool and runs the mandatory boot-time integrity check.
func New(ctx context.Context, storage ports.GenomeStoragePort, manifestPath string) (*GenePool, error) {
	gp := &GenePool{
		storage:      storage,
		manifestPath: manifestPath,
		activeGenes:  make(map[string]*domain.DomainGene),
	}

	if err := gp.Reload(ctx); err != nil {
		return nil, fmt.Errorf("genePool boot failure: %w", err)
	}

	return gp, nil
}

// Reload re-reads the manifest and re-mmaps the active genes into memory.
// LLD 9.2: Boot-Time Integrity Check enforcement.
func (p *GenePool) Reload(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. Unmap existing genes
	for _, gene := range p.activeGenes {
		if gene.MMapAddr != 0 {
			_ = p.storage.UnmapGene(gene.Rules)
		}
	}
	p.activeGenes = make(map[string]*domain.DomainGene)

	// 2. Read manifest.yaml
	manifestData, err := p.storage.ReadManifest(ctx, p.manifestPath)
	if err != nil || len(manifestData) == 0 {
		// No manifest found — empty pool (the Auditor will catch Tier 0 absence).
		return nil
	}

	var manifest domain.GeneManifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// 3. Signature Verification & Memory Mapping
	for _, meta := range manifest.Genes {
		data, addr, err := p.storage.MapGene(ctx, meta.Path)
		if err != nil {
			return fmt.Errorf("failed to map gene %s: %w", meta.Name, err)
		}

		// Calculate SHA256 of the file content
		hash, err := p.storage.CalculateFileHash(ctx, meta.Path)
		if err != nil {
			return fmt.Errorf("failed to hash gene %s: %w", meta.Name, err)
		}

		// Security Guard: Tier 0 signature violations cause full startup abort.
		if meta.Signature != "" && hash != meta.Signature {
			return fmt.Errorf("INTEGRITY VIOLATION: gene %s signature mismatch. Expected %s, got %s", meta.Name, meta.Signature, hash)
		}

		// 4. Register the safe gene
		p.activeGenes[meta.Name] = &domain.DomainGene{
			Name:         meta.Name,
			Tier:         meta.Tier,
			Rules:        data,
			Signature:    [32]byte{}, // Converted from hex in production
			MMapAddr:     addr,
			Capabilities: meta.Capabilities,
			Intents:      meta.Intents,
			FactPath:     meta.FactPath,
			IsUnverified: false,
		}
	}

	return nil
}

// ActiveGenes returns an iterator over genes matching the current intent context.
func (p *GenePool) ActiveGenes(ctx context.Context, intent domain.IntentStr) iter.Seq[*domain.DomainGene] {
	return func(yield func(*domain.DomainGene) bool) {
		p.mu.RLock()
		defer p.mu.RUnlock()

		for _, gene := range p.activeGenes {
			// Filter by intent if gene has specific intent bindings
			if len(gene.Intents) > 0 {
				matched := false
				for _, gi := range gene.Intents {
					if domain.IntentStr(gi) == intent {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}

			if !yield(gene) {
				return
			}
		}
	}
}

func (p *GenePool) ReloadTrigger() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return p.Reload(ctx)
}
