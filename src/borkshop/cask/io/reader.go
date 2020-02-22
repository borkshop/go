package caskio

import (
	"context"
	"io"

	"borkshop/cask"
)

// Reader tracks the state of a reader that loads and iterates over the leaves
// of a 1KB blockwise B-tree.
type Reader struct {
	store cask.Store
	stack readStack
}

// NewReader returns a reader for the given root hash of a 1KB block B-tree
// backed by the given store.
func NewReader(store cask.Store, h cask.Hash) *Reader {
	return &Reader{
		store: store,
		stack: readStack{readFrame{
			links: []cask.Hash{h},
		}},
	}
}

type readStack []readFrame

type readFrame struct {
	links []cask.Hash
	index int
}

// Next returns the content of the next leaf of the B-tree.
//
// The first return value is a list of links if any, the second is the data if
// any.
//
// If there are no further leaves in the B-tree, returns io.EOF for the error.
// All other errors indicate premature termination.
func (reader *Reader) Next(ctx context.Context) ([]cask.Hash, []byte, error) {
	for len(reader.stack) > 0 {
		top := &reader.stack[len(reader.stack)-1]
		index := top.index
		if top.index >= len(top.links) {
			// pop
			reader.stack = reader.stack[:len(reader.stack)-1]
			continue
		}
		top.index++
		h := top.links[index]

		var m cask.Model
		err := m.Load(ctx, reader.store, h)
		if err != nil {
			return nil, nil, err
		}

		if m.Height > 0 {
			// push
			reader.stack = append(reader.stack, readFrame{
				links: m.Links,
				index: 0,
			})
		} else {
			return m.Links, m.Bytes, nil
		}
	}
	return nil, nil, io.EOF
}
