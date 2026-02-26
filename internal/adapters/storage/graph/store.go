package graph

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/dgraph-io/badger/v4"
)

// Store provides the persistent BadgerDB backend for the Sovereign Logic Kernel.
type Store struct {
	db   *badger.DB
	path string

	numFacts atomic.Uint64 // Lock-free read for high-throughput queries
	closed   uint32        // 0 = open, 1 = closed
}

// NewStore initializes a high-performance BadgerDB instance optimized for
// Manglekit's read-heavy Datalog workloads and asynchronous agent write-behinds.
func NewStore(path string, readOnly bool) (*Store, error) {
	opts := badger.DefaultOptions(path)

	if readOnly {
		opts = opts.WithReadOnly(true)
	} else {
		// Asynchronous writes are critical to preventing the OODA loop from
		// blocking on disk I/O when writing transient agent states.
		opts = opts.WithSyncWrites(false)
	}

	opts.Logger = nil // Disable default Badger spam

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open graph store at %s: %w", path, err)
	}

	s := &Store{
		db:   db,
		path: path,
	}

	if err := s.initializeCounters(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// initializeCounters reads system metadata on boot to populate atomic counters.
func (s *Store) initializeCounters() error {
	var count uint64
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(KeyFactCount)
		if err == badger.ErrKeyNotFound {
			count = 0
			return nil
		}
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			if len(val) == 8 {
				count = binary.BigEndian.Uint64(val)
			}
			return nil
		})
	})

	if err != nil {
		return fmt.Errorf("failed to read KeyFactCount: %w", err)
	}

	s.numFacts.Store(count)
	return nil
}

// Close gracefully terminates the Badger instance, flushing memtables.
func (s *Store) Close() error {
	if !atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		return nil // Already closed
	}
	return s.db.Close()
}

// AddFact persists a single quad into all four index permutations atomically.
func (s *Store) AddFact(s_, p, o, g uint64) error {
	return s.db.Update(func(txn *badger.Txn) error {
		prefixes := []byte{PrefixSPOg, PrefixPOSg, PrefixPSOg, PrefixGSPO}
		for _, prefix := range prefixes {
			key := EncodeQuadKey(prefix, s_, p, o, g)
			if err := txn.Set(key, nil); err != nil { // Value-less keys
				return err
			}
		}

		// Update counter
		s.numFacts.Add(1)
		countBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(countBuf, s.numFacts.Load())
		return txn.Set(KeyFactCount, countBuf)
	})
}

// Scan performs a prefix scan over a specific index permutation.
// The callback receives the decoded (s, p, o, g) tuple for each matching key.
func (s *Store) Scan(prefix byte, seekKey []byte, limit int, fn func(s, p, o, g uint64) bool) error {
	return s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // Keys-only scan
		opts.Prefix = []byte{prefix}

		it := txn.NewIterator(opts)
		defer it.Close()

		count := 0
		for it.Seek(seekKey); it.ValidForPrefix([]byte{prefix}); it.Next() {
			key := it.Item().Key()
			s_, p_, o_, g_, err := DecodeQuadKey(key)
			if err != nil {
				continue
			}

			if !fn(s_, p_, o_, g_) {
				break
			}

			count++
			if limit > 0 && count >= limit {
				break
			}
		}
		return nil
	})
}

// NumFacts returns the total number of stored facts (lock-free read).
func (s *Store) NumFacts() uint64 {
	return s.numFacts.Load()
}

// RunVLogGC is the Incremental Garbage Collection loop required by LLD 4.2.
// Long-running agents constantly update temporal state, bloating the BadgerDB
// value logs. This MUST be called as a background goroutine.
func (s *Store) RunVLogGC(ctx context.Context) {
	// Probe every 15 minutes as per spec.
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Run GC until it returns errNoRewrite (no more files meet threshold)
			for {
				if atomic.LoadUint32(&s.closed) == 1 {
					return
				}

				// Target files with >= 50% stale data (0.5 discard ratio)
				err := s.db.RunValueLogGC(0.5)
				if err != nil {
					break // ErrNoRewrite or other error stops the burst
				}

				// Throttle successive sweeps to prevent container CPU throttling
				time.Sleep(1 * time.Second)
			}
		}
	}
}
