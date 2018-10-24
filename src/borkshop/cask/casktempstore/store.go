// Package casktempstore provides an ephemeral in-memory content address store
// implementation.
//
// This store collects cells if after the deadline expires for all load or
// store calls that refer to it.
package casktempstore

import (
	"context"
	"sync"

	"borkshop/cask"
)

// New returns a simple, in-memory, 1KB blockwise temporary content address store.
func New() *Store {
	return &Store{
		cells:     make(map[cask.Hash]*cell),
		collector: newCollector(),
	}
}

// Store is a simple, in-memory, 1KB blockwise content address store.
type Store struct {
	lock      sync.Mutex
	cells     map[cask.Hash]*cell
	collector *collector
}

// Store captures a block in memory.
func (store *Store) Store(ctx context.Context, hash cask.Hash, block *cask.Block) error {
	cell, err := store.cell(ctx, hash)
	if err != nil {
		return err
	}

	cell.store(ctx, hash, block)
	return nil
}

// Load retrieves a block from memory.
func (store *Store) Load(ctx context.Context, hash cask.Hash, block *cask.Block) error {
	cell, err := store.cell(ctx, hash)
	if err != nil {
		return err
	}
	return cell.load(ctx, hash, block)
}

func (store *Store) cell(ctx context.Context, hash cask.Hash) (*cell, error) {
	store.lock.Lock()
	defer store.lock.Unlock()

	if err := store.collector.Touch(ctx, hash); err != nil {
		return nil, err
	}
	store.collector.Collect(store.cells)

	cell, ok := store.cells[hash]
	if !ok {
		cell = newCell()
		store.cells[hash] = cell
	}
	return cell, nil
}

func (cell *cell) store(ctx context.Context, hash cask.Hash, block *cask.Block) {
	cell.lock.Lock()
	cell.block = *block
	if !cell.stored {
		cell.stored = true
		close(cell.ready)
	}
	cell.lock.Unlock()
}

func (cell *cell) load(ctx context.Context, hash cask.Hash, block *cask.Block) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-cell.ready:
	}

	*block = cell.block
	return nil
}

// cell captures a block and synchronizes concurrent reads and writes.
type cell struct {
	lock   sync.RWMutex
	block  cask.Block // write-once, then close(ready)
	stored bool       // close(ready) only once
	ready  chan struct{}
}

func newCell() *cell {
	return &cell{
		ready: make(chan struct{}, 0),
	}
}
