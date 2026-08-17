package graph

import (
	"context"
	"encoding/binary"
	"errors"
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
// Writes are asynchronous (WithSyncWrites(false)) — the right default for
// cache-style, re-derivable state. For session persistence that must survive
// a crash, use NewStoreWithOptions with syncWrites=true.
func NewStore(path string, readOnly bool) (*Store, error) {
	return NewStoreWithOptions(path, readOnly, false)
}

// NewStoreWithOptions is NewStore with an explicit durability stance.
// syncWrites=true makes Badger fsync on write (WithSyncWrites(true)), the
// opt-in durability option for session persistence; the default (false)
// keeps asynchronous writes for cache-style usage.
func NewStoreWithOptions(path string, readOnly, syncWrites bool) (*Store, error) {
	opts := badger.DefaultOptions(path)

	if readOnly {
		opts = opts.WithReadOnly(true)
	} else {
		// Asynchronous writes are critical to preventing the OODA loop from
		// blocking on disk I/O when writing transient agent states — unless
		// the caller explicitly opted into durable session persistence.
		opts = opts.WithSyncWrites(syncWrites)
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
		// Only count genuinely new facts. Value-less keys are idempotent on
		// re-insert, so a duplicate AddFact must not inflate the counter.
		spogKey := EncodeQuadKey(PrefixSPOg, s_, p, o, g)
		_, err := txn.Get(spogKey)
		isNew := errors.Is(err, badger.ErrKeyNotFound)
		if err != nil && !isNew {
			return err
		}

		prefixes := []byte{PrefixSPOg, PrefixPOSg, PrefixPSOg, PrefixGSPO}
		for _, prefix := range prefixes {
			key := EncodeQuadKey(prefix, s_, p, o, g)
			if err := txn.Set(key, nil); err != nil { // Value-less keys
				return err
			}
		}

		// Update counter only for new facts, persisted atomically with the fact.
		if isNew {
			s.numFacts.Add(1)
			countBuf := make([]byte, 8)
			binary.BigEndian.PutUint64(countBuf, s.numFacts.Load())
			if err := txn.Set(KeyFactCount, countBuf); err != nil {
				return err
			}
		}
		return nil
	})
}

// Quad is a batch-friendly tuple of the four quad components. See AddFact
// for the per-index layout.
type Quad struct {
	S, P, O, G uint64
}

// AddFacts persists a batch of quads using a single Badger write batch
// instead of one transaction per quad. The fact counter is updated once,
// atomically with the batch, counting only genuinely new facts (duplicate
// SPOg keys within or before the batch do not inflate it).
func (s *Store) AddFacts(quads []Quad) error {
	if len(quads) == 0 {
		return nil
	}

	// Pre-scan for duplicates so the counter stays exact.
	var newCount uint64
	newSet := make(map[[4]uint64]struct{}, len(quads))
	err := s.db.View(func(txn *badger.Txn) error {
		for _, q := range quads {
			key := [4]uint64{q.S, q.P, q.O, q.G}
			if _, dup := newSet[key]; dup {
				continue
			}
			newSet[key] = struct{}{}
			_, err := txn.Get(EncodeQuadKey(PrefixSPOg, q.S, q.P, q.O, q.G))
			if err != nil && !errors.Is(err, badger.ErrKeyNotFound) {
				return err
			}
			if errors.Is(err, badger.ErrKeyNotFound) {
				newCount++
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to pre-scan batch for duplicates: %w", err)
	}

	wb := s.db.NewWriteBatch()
	defer wb.Cancel()

	prefixes := []byte{PrefixSPOg, PrefixPOSg, PrefixPSOg, PrefixGSPO}
	for q := range newSet {
		for _, prefix := range prefixes {
			if err := wb.Set(EncodeQuadKey(prefix, q[0], q[1], q[2], q[3]), nil); err != nil {
				return fmt.Errorf("failed to queue batch write: %w", err)
			}
		}
	}

	if newCount > 0 {
		countBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(countBuf, s.numFacts.Add(newCount))
		if err := wb.Set(KeyFactCount, countBuf); err != nil {
			return fmt.Errorf("failed to queue fact count write: %w", err)
		}
	}

	if err := wb.Flush(); err != nil {
		return fmt.Errorf("failed to flush batch: %w", err)
	}
	return nil
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
