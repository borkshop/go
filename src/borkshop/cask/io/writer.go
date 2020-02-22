package caskio

import (
	"context"

	"borkshop/cask"
)

// Writer represents the state for iteratively constructing and storing a new
// B-tree of 1KB blocks.
type Writer struct {
	store cask.Store
	stack writeStack
}

// NewWriter creates a block writer for the given content store.
func NewWriter(store cask.Store) *Writer {
	return &Writer{
		store: store,
		stack: writeStack{cask.Model{}},
	}
}

// Size returns the size of the current accumulated leaf node.
func (writer *Writer) Size() int {
	return writer.stack[0].Size()
}

// Flush constructs a root node that captures all of the accumulated leaf nodes
// so far.
func (writer *Writer) Flush(ctx context.Context) error {
	for i := 0; i < len(writer.stack)-1; i++ {
		var err error
		writer.stack, err = writer.stack.Flush(ctx, writer.store, i)
		if err != nil {
			return err
		}
	}
	return nil
}

// Copy writes bytes into a B-tree of blocks.
func (writer *Writer) Copy(ctx context.Context, buf []byte) error {
	i := 0
	for i < len(buf) {
		if writer.stack[0].Size() == cask.BlockSize {
			var err error
			writer.stack, err = writer.stack.Flush(ctx, writer.store, 0)
			if err != nil {
				return err
			}
		}
		i += writer.stack[0].AppendBytes(buf[i:])
	}
	return nil
}

// Link adds a link onto a B-tree of blocks.
func (writer *Writer) Link(ctx context.Context, h cask.Hash) error {
	if writer.stack[0].Size()+cask.HashSize > cask.BlockSize {
		var err error
		writer.stack, err = writer.stack.Flush(ctx, writer.store, 0)
		if err != nil {
			return err
		}
	}
	writer.stack[0].AppendLink(h)
	return nil
}

// Sum returns the address of the root node of the B-tree accumulated so far.
func (writer *Writer) Sum(ctx context.Context) (cask.Hash, error) {
	return writer.stack.Sum(ctx, writer.store)
}

type writeStack []cask.Model

func (stack writeStack) Flush(ctx context.Context, store cask.Store, i int) (writeStack, error) {
	if i+1 >= len(stack) {
		stack = append(stack, cask.Model{Height: i + 1})
	} else if stack[i+1].Size()+cask.HashSize >= cask.BlockSize {
		var err error
		stack, err = stack.Flush(ctx, store, i+1)
		if err != nil {
			return stack[0:0], err
		}
	}

	h, err := stack[i].Store(ctx, store)
	if err != nil {
		return stack, err
	}

	// Add the model to the parent
	stack[i+1].AppendLink(h)
	stack[i] = cask.Model{Height: i}
	return stack, nil
}

func (stack writeStack) Sum(ctx context.Context, store cask.Store) (cask.Hash, error) {
	for i := 0; i < len(stack)-1; i++ {
		var err error
		stack, err = stack.Flush(ctx, store, i)
		if err != nil {
			return cask.ZeroHash, err
		}
	}
	return stack[len(stack)-1].Store(ctx, store)
}
