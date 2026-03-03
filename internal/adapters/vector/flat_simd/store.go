package flat_simd

import (
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"sync"
	"syscall"

	"github.com/duynguyendang/manglekit/internal/core/ports"
)

// The MRL pipeline truncates embeddings to the first 64 dimensions, providing
// high preservation of semantic layout while dropping index size by 95%.
const DefaultVectorDim = 64

// Each vector record in vectors.bin consists of the 64 int8 values + an 8-byte ID (uint64).
// This is exactly 72 bytes per record, strictly aligned for rapid memory scans.
const recordSize = DefaultVectorDim + 8

// FlatSIMDStore implements VectorStorePort using a flat, memory-mapped INT8 file
// designed for embedded scale workloads (100k - 1M vectors).
type FlatSIMDStore struct {
	mu       sync.RWMutex
	filePath string
	file     *os.File
	mmap     []byte

	recordCount int
}

// NewStore initializes or opens a flat vector store from disk.
func NewStore(path string) (*FlatSIMDStore, error) {
	s := &FlatSIMDStore{
		filePath: path,
	}

	// Create if not exists, open for read/write
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open vector store: %w", err)
	}
	s.file = f

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	size := info.Size()

	// If empty, pre-allocate a tiny buffer (prevent mmap error on 0 bytes)
	if size == 0 {
		initialSize := int64(recordSize * 1024) // Room for 1024 vectors initially
		if err := f.Truncate(initialSize); err != nil {
			f.Close()
			return nil, err
		}
		size = initialSize
	} else if size%recordSize != 0 {
		// Validating alignment
		f.Close()
		return nil, fmt.Errorf("vector file size %d not a multiple of record size %d", size, recordSize)
	}

	s.recordCount = int(size / recordSize)

	// Memory-map the file
	m, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to mmap vectors: %w", err)
	}
	s.mmap = m

	return s, nil
}

// Close unmaps and closes the file gracefully.
func (s *FlatSIMDStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mmap != nil {
		_ = syscall.Munmap(s.mmap)
		s.mmap = nil
	}
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// Quantize handles the Float32 -> INT8 conversion loop.
// To normalize correctly, it evaluates the min/max range of the *truncated* array.
func Quantize(raw ports.Vector) ([]int8, error) {
	if len(raw) < DefaultVectorDim {
		return nil, fmt.Errorf("vector dimension %d is smaller than Matryoshka target %d", len(raw), DefaultVectorDim)
	}

	// 1. Matryoshka Truncation (Select first 64 float elements)
	truncated := raw[:DefaultVectorDim]

	// 2. Find min & max for dynamic [-128, 127] scaling
	var min, max float32 = math.MaxFloat32, -math.MaxFloat32
	for _, val := range truncated {
		if val < min {
			min = val
		}
		if val > max {
			max = val
		}
	}

	// Prevent div by zero
	rng := max - min
	if rng == 0 {
		rng = 1
	}

	// 3. Scale and cast to int8
	quantized := make([]int8, DefaultVectorDim)
	scale := 255.0 / rng
	for i, val := range truncated {
		// Shift range to 0..255, then offset back to -128..127 bounds
		normalized := (val - min) * scale
		quantized[i] = int8(math.Round(float64(normalized)) - 128)
	}

	return quantized, nil
}

// Insert adds a quantized vector with its metadata ID into the mmap'd file.
func (s *FlatSIMDStore) Insert(vector ports.Vector, metadata string) error {
	quantized, err := Quantize(vector)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the next available slot by scanning for a zero ID
	// In production, maintain a write cursor. For now, append at end.
	offset := s.recordCount * recordSize

	// Check if we need to grow the file
	if offset+recordSize > len(s.mmap) {
		// Unmap, grow, remap
		_ = syscall.Munmap(s.mmap)
		newSize := int64(len(s.mmap)) * 2
		if err := s.file.Truncate(newSize); err != nil {
			return fmt.Errorf("failed to grow vector file: %w", err)
		}
		m, err := syscall.Mmap(int(s.file.Fd()), 0, int(newSize), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
		if err != nil {
			return fmt.Errorf("failed to remap vectors: %w", err)
		}
		s.mmap = m
	}

	// Write the INT8 vector
	for i, v := range quantized {
		s.mmap[offset+i] = byte(v)
	}

	// Write a simple hash of the metadata as the 8-byte ID
	h := fnv.New64a()
	h.Write([]byte(metadata))
	id := h.Sum64()
	s.mmap[offset+DefaultVectorDim] = byte(id >> 56)
	s.mmap[offset+DefaultVectorDim+1] = byte(id >> 48)
	s.mmap[offset+DefaultVectorDim+2] = byte(id >> 40)
	s.mmap[offset+DefaultVectorDim+3] = byte(id >> 32)
	s.mmap[offset+DefaultVectorDim+4] = byte(id >> 24)
	s.mmap[offset+DefaultVectorDim+5] = byte(id >> 16)
	s.mmap[offset+DefaultVectorDim+6] = byte(id >> 8)
	s.mmap[offset+DefaultVectorDim+7] = byte(id)

	s.recordCount++
	return nil
}

// Search performs a brute-force INT8 dot-product scan over the mmap'd vectors,
// returning metadata IDs sorted by descending similarity.
func (s *FlatSIMDStore) Search(query ports.Vector, limit int, threshold float64) ([]string, error) {
	quantizedQuery, err := Quantize(query)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		id    uint64
		score float64
	}

	var results []scored

	for i := 0; i < s.recordCount; i++ {
		offset := i * recordSize

		// Read the 8-byte ID suffix
		id := uint64(s.mmap[offset+DefaultVectorDim])<<56 |
			uint64(s.mmap[offset+DefaultVectorDim+1])<<48 |
			uint64(s.mmap[offset+DefaultVectorDim+2])<<40 |
			uint64(s.mmap[offset+DefaultVectorDim+3])<<32 |
			uint64(s.mmap[offset+DefaultVectorDim+4])<<24 |
			uint64(s.mmap[offset+DefaultVectorDim+5])<<16 |
			uint64(s.mmap[offset+DefaultVectorDim+6])<<8 |
			uint64(s.mmap[offset+DefaultVectorDim+7])

		if id == 0 {
			continue // Empty slot
		}

		// INT8 dot product
		var dotProduct int32
		for j := 0; j < DefaultVectorDim; j++ {
			a := int32(int8(s.mmap[offset+j]))
			b := int32(quantizedQuery[j])
			dotProduct += a * b
		}

		// Normalize to [-1, 1] range (approximate)
		normalizedScore := float64(dotProduct) / float64(127*127*DefaultVectorDim)

		if normalizedScore >= threshold {
			results = append(results, scored{id: id, score: normalizedScore})
		}
	}

	// Sort by score descending
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].score > results[i].score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Collect top results
	var out []string
	for i := 0; i < len(results) && i < limit; i++ {
		out = append(out, fmt.Sprintf("%d", results[i].id))
	}

	return out, nil
}
