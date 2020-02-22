package caskblob

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"borkshop/cask"
	"borkshop/cask/io"
)

const storeBlobBufferSize = 4096

// Write stores bytes as blocks in the given store and returns the root hash.
func Write(ctx context.Context, store cask.Store, buf []byte) (cask.Hash, error) {
	return Store(ctx, store, bytes.NewBuffer(buf))
}

// WriteString stores a string as blocks in the given store and returns the root hash.
func WriteString(ctx context.Context, store cask.Store, str string) (cask.Hash, error) {
	// TODO avoid duplicating the bytes underlying the immutable string.
	return Write(ctx, store, []byte(str))
}

// Read reads all of the bytes of the addressed blob.
func Read(ctx context.Context, store cask.Store, hash cask.Hash) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := Load(ctx, store, buf, hash); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ReadString reads the entire string of the addressed blob.
func ReadString(ctx context.Context, store cask.Store, hash cask.Hash) (string, error) {
	buf, err := Read(ctx, store, hash)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// Store reads a stream and writes the corresponding B-tree of 1KB blocks to a
// content address store, returning the hash of the root block.
func Store(ctx context.Context, store cask.Store, reader io.Reader) (cask.Hash, error) {
	writer := caskio.NewWriter(store)
	for {
		var buf [storeBlobBufferSize]byte
		n, err := reader.Read(buf[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return cask.ZeroHash, fmt.Errorf("read error: %s", err)
		}
		if err := writer.Copy(ctx, buf[:n]); err != nil {
			return cask.ZeroHash, err
		}
	}
	return writer.Sum(ctx)
}

// Load reads the blocks of an object from the given content address store and
// writes them to the given stream.
func Load(ctx context.Context, store cask.Store, writer io.Writer, h cask.Hash) error {
	reader := caskio.NewReader(store, h)
	for {
		links, buf, err := reader.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(links) > 0 {
			return fmt.Errorf("unexpected links in blob block")
		}
		_, err = writer.Write(buf)
		if err != nil {
			return err
		}
	}
	return nil
}
