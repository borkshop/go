// Package caskmemstore provides an in-memory content address store
// implementation.
package caskmemstore

import (
	"context"
	"sync"

	"borkshop/cask"
)

// TODO come up with better test and benchmark infrastructure and evolve a
// better data structure and algorithm for high performance concurrent reads
// and writes.
// The current algorithm keeps the global lock too long during a concurrent
// read-then-write of a particular block.

// New returns a simple, in-memory, 1KB blockwise content address store.
func New() *MemStore {
	return &MemStore{
		cells: make(map[cask.Hash]*cell),
	}
}

// MemStore is a simple, in-memory, 1KB blockwise content address store.
type MemStore struct {
	lock  sync.RWMutex
	cells map[cask.Hash]*cell
}

var _ cask.Store = (*MemStore)(nil)

// cell captures a block and synchronizes concurrent reads and writes.
type cell struct {
	block  cask.Block
	stored bool
	ready  chan struct{}
}

// Store captures a block in memory.
func (s *MemStore) Store(_ context.Context, h cask.Hash, b *cask.Block) error {
	s.lock.Lock()
	c, ok := s.cells[h]
	if !ok {
		c = &cell{
			block:  *b,
			stored: true,
			ready:  make(chan struct{}),
		}
		s.cells[h] = c
		close(c.ready)
	} else if !c.stored {
		c.stored = true
		c.block = *b
		close(c.ready)
	}
	s.lock.Unlock()

	return nil
}

// Load retrieves a block from memory.
func (s *MemStore) Load(ctx context.Context, h cask.Hash, b *cask.Block) error {
	s.lock.RLock()
	c, ok := s.cells[h]
	if !ok {
		c = &cell{
			ready: make(chan struct{}),
		}
		s.cells[h] = c
	}
	s.lock.RUnlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ready:
		*b = c.block
		return nil
	}
}
