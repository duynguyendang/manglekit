package dict

import (
	"hash/fnv"
	"sync"
	"sync/atomic"
)

const numShards = 32

// ShardedDict provides lock-free(ish) string interning for high-concurrency
// knowledge graph operations. It hashes strings into one of 32 shards to
// avoid global map RWMutex contention.
type ShardedDict struct {
	shards [numShards]*dictShard
	nextID atomic.Uint64
}

type dictShard struct {
	mu      sync.RWMutex
	strToID map[string]uint64
	idToStr map[uint64]string
}

// NewShardedDict initializes a new sharded dictionary.
func NewShardedDict() *ShardedDict {
	d := &ShardedDict{}
	for i := 0; i < numShards; i++ {
		d.shards[i] = &dictShard{
			strToID: make(map[string]uint64),
			idToStr: make(map[uint64]string),
		}
	}
	// Reserve IDs 0-99 for system prefixes if needed later.
	d.nextID.Store(100)
	return d
}

// hashFNV1a computes a simple fast hash to select a shard.
func hashString(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// shardFor returns the specific shard for a given string.
func (d *ShardedDict) shardFor(s string) *dictShard {
	idx := hashString(s) % numShards
	return d.shards[idx]
}

// GetOrCreateID returns the uint64 ID for a string, creating it if it doesn't exist.
func (d *ShardedDict) GetOrCreateID(s string) uint64 {
	shard := d.shardFor(s)

	// Optimistic Read
	shard.mu.RLock()
	if id, exists := shard.strToID[s]; exists {
		shard.mu.RUnlock()
		return id
	}
	shard.mu.RUnlock()

	// Write Lock
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Double check after acquiring write lock
	if id, exists := shard.strToID[s]; exists {
		return id
	}

	newID := d.nextID.Add(1)
	shard.strToID[s] = newID
	shard.idToStr[newID] = s
	return newID
}

// GetString optimally retrieves a string by traversing shards.
// Given that we don't know the string a priori to hash it, we must scan.
// In a full DB implementation (like Badger), this would ideally be
// backed by a persistent `vocab/` DB store to avoid linear RAM scans,
// but for in-memory caching, the RLock iteration is extremely fast.
func (d *ShardedDict) GetString(id uint64) (string, bool) {
	for _, shard := range d.shards {
		shard.mu.RLock()
		if s, ok := shard.idToStr[id]; ok {
			shard.mu.RUnlock()
			return s, true
		}
		shard.mu.RUnlock()
	}
	return "", false
}

// Load precompiles the dictionary from a persistent store (e.g. Badger).
func (d *ShardedDict) Load(s string, id uint64) {
	shard := d.shardFor(s)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.strToID[s] = id
	shard.idToStr[id] = s

	// Ensure nextID stays ahead of highest loaded ID
	for {
		current := d.nextID.Load()
		if id < current || d.nextID.CompareAndSwap(current, id+1) {
			break
		}
	}
}
