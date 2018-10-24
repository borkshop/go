package caskio

import (
	"context"
	"fmt"

	"borkshop/cask"
)

// Copy transfers a block and its transitive links from one store to another,
// to be retained at least until the context deadline.
func Copy(ctx context.Context, target, source cask.Store, hash cask.Hash) error {
	// TODO concurrent copy (scattered order)
	hashes, err := BOM(ctx, source, hash)
	if err != nil {
		return err
	}
	block := &cask.Block{}
	for hash := range hashes {
		fmt.Printf("COPY %x\n", hash)
		if err := source.Load(ctx, hash, block); err != nil {
			return err
		}
		if err := target.Store(ctx, hash, block); err != nil {
			return err
		}
	}
	return nil
}

// BOM returns a "bill of materials" for a given hash, collecting the transitive
// links of a root block.
func BOM(ctx context.Context, store cask.Store, hash cask.Hash) (map[cask.Hash]struct{}, error) {
	// TODO concurrent collect, maybe. BOM may be okay as-is since it would
	// only be used for local, low-latency stores, which might in turn be
	// slower with concurrency.
	links := make(map[cask.Hash]struct{}, 1)
	if err := bom(ctx, store, hash, links); err != nil {
		return nil, err
	}
	return links, nil
}

func bom(ctx context.Context, store cask.Store, hash cask.Hash, links map[cask.Hash]struct{}) error {
	if _, ok := links[hash]; ok {
		return nil
	}
	links[hash] = struct{}{}
	block := &cask.Block{}
	if err := store.Load(ctx, hash, block); err != nil {
		return err
	}
	for _, link := range block.Links() {
		bom(ctx, store, link, links)
	}
	return nil
}
