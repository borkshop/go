// Package cask defines the types for a 1KB blockwise content address store.
//
// Every block is 1KB and contains a combination of links to other blocks in
// the form of SHA-256 hashes, and byte content.
//
// Each block also has a height.
// Content that spans multiple blocks forms a B-tree, using links to address
// child blocks.
//
// Blobs are a B-tree where leaf blocks contain data without links.
//
// Directories are a B-tree where leaf blocks contain entries, using both links
// and data to capture the hash, type, and name of each child, ordered by name,
// and divided among leaf blocks such that each leaf contains as many entries
// as can fit.
//
// Cask supports arbitrary block types beyond blobs and directories and only
// imposes the basic block structure of height, links, and data on all types.
// This allows us to independently evolve semantics, storage, and transport.
//
// Cask blocks are 1 Kilobyte to fit in the typical Ethernet MTU, to make
// blocks particularly well-suited for UDP transport, and also so a round
// number of blocks will fit in a typical filesystem block.
// 1KB block B-trees are also well-suited for order-independent peer to peer
// file transfer.
//
// 1KB block CAS B-trees can also effeciently represent evolution of large
// immutable data structures.
// For example, adding a link to a B-tree representing an ordered set of links
// would involve the creation of new blocks proportional to the logarithm of
// the size of the set.
// Merging append-only sets to deterministically arrive at a consistent root
// hash is expensive but possible.
//
// In memory a block is laid out as a 1 byte height, followed by the number of
// links (in a byte, which cannot exceed 31), the number of content bytes (in
// two bytes and cannot exceed 1KB less the four bytes of headers), then the
// links, and finally the bytes.
//
//  height:1
//  numLinks:1
//  numBytes:2
//  links:32*numLinks
//  bytes:numBytes
package cask

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

const (
	// HashSize is the number of bytes in a SHA-256 hash (32).
	HashSize = sha256.Size
	// BlockSize is the number of bytes in a block (1024).
	BlockSize = 1024
)

// Hash represents a SHA-256, as 32 bytes.
type Hash [HashSize]byte

// ZeroHash is a blank hash.
var ZeroHash = Hash{}

// Block is a 1 kilobyte block of bytes.
type Block [BlockSize]byte

// Hash returns the SHA-256 hash of the occupied bytes in the block.
func (block *Block) Hash() Hash {
	return Hash(sha256.Sum256(block[:block.Size()]))
}

// Size returns the size of the block's effective content.
func (block *Block) Size() int {
	numLinks := int(block[1])
	numBytes := int(binary.BigEndian.Uint16(block[2:4]))
	size := 4 + numLinks*HashSize + numBytes
	if size > BlockSize {
		size = 0
	}
	return size
}

// Links reads and returns a block's links.
func (block *Block) Links() []Hash {
	numLinks := int(block[1])
	// Hedge against corruption.
	if numLinks > 31 {
		numLinks = 31
	}
	links := make([]Hash, 0, numLinks)
	size := 4 + numLinks*HashSize
	// Again, hedging against corruption.
	if size > BlockSize {
		size = BlockSize
	}
	var link Hash
	for i := 4; i < size; i += HashSize {
		copy(link[:], block[i:])
		links = append(links, link)
	}
	return links
}

// Store is an interface for storing blocks by their hash.
//
// It is the prerogative of the implementation to either trust or validate that
// the given hash corresponds to the given block, but they are accepted
// separately to avoid redundant hashing.
type Store interface {
	// Store places a block in the store at least until the context's deadline.
	Store(context.Context, Hash, *Block) error
	// Load retrieves a block from the store or fails if the context reaches
	// its deadline first.
	Load(context.Context, Hash, *Block) error
}

// Model represents a block, suitable for building and marshalling.
type Model struct {
	// Height is the height of the modeled block in the B-tree.
	//
	// Leaf blocks have a height of 0.
	Height int

	// Links is a slice of hashes for blocks retained by the modeled block.
	//
	// We use links for B-trees, directories, and any other complex structure.
	Links []Hash

	// Bytes are the content of the modeled block.
	Bytes []byte
}

// Size is the number of bytes effectively used of the modeled 1KB block,
// including the headers, links, and bytes.
func (model *Model) Size() int {
	return 4 + len(model.Bytes) + sha256.Size*len(model.Links)
}

// AppendLink adds a link to the modeled block.
func (model *Model) AppendLink(h Hash) {
	model.Links = append(model.Links, h)
}

// AppendBytes adds bytes to the modeled block.
func (model *Model) AppendBytes(buf []byte) int {
	rem := BlockSize - model.Size()
	if len(buf) < rem {
		rem = len(buf)
	}
	model.Bytes = append(model.Bytes, buf[0:rem]...)
	return rem
}

// AppendString adds a string to the modeled block.
func (model *Model) AppendString(str string) int {
	// TODO avoid copying the underlying bytes when casting from immutable
	// string to mutable bytes; write directly to model buffer.
	return model.AppendBytes([]byte(str))
}

// Store encodes and stores a block.
func (model *Model) Store(ctx context.Context, store Store) (Hash, error) {
	var b Block
	if err := model.Put(&b); err != nil {
		return Hash{}, err
	}
	k := sha256.Sum256(b[:])
	return k, store.Store(ctx, k, &b)
}

// Load loads and decodes a block.
func (model *Model) Load(ctx context.Context, store Store, k Hash) error {
	var b Block
	if err := store.Load(ctx, k, &b); err != nil {
		return err
	}
	return model.Get(&b)
}

// Put encodes the model into a block.
func (model Model) Put(buf *Block) error {
	height := model.Height
	numLinks := len(model.Links)
	numBytes := len(model.Bytes)
	if 4+numLinks*sha256.Size+numBytes > BlockSize {
		return fmt.Errorf("corrupt block: %d 32 byte links, %d bytes, 4 bytes in headers exceed block size", numLinks, numBytes)
	}

	buf[0] = byte(height)
	buf[1] = byte(numLinks)
	binary.BigEndian.PutUint16(buf[2:4], uint16(numBytes))
	at := 4
	for _, h := range model.Links {
		copy(buf[at:at+sha256.Size], h[:])
		at += sha256.Size
	}
	copy(buf[4+len(model.Links)*sha256.Size:], model.Bytes)
	return nil
}

// Get decodes a model from a block.
func (model *Model) Get(b *Block) error {
	height := int(b[0])
	numLinks := int(b[1])
	numBytes := int(binary.BigEndian.Uint16(b[2:4]))

	if 4+numLinks*sha256.Size+numBytes > BlockSize {
		return fmt.Errorf("corrupt block: %d 32 byte links, %d bytes, 4 bytes in headers exceed block size", numLinks, numBytes)
	}

	model.Height = height
	at := 4
	model.Links = make([]Hash, 0, numLinks)
	for i := 0; i < numLinks; i++ {
		var h Hash
		copy(h[:], b[at:])
		model.Links = append(model.Links, h)
		at += sha256.Size
	}
	model.Bytes = b[at : at+numBytes]
	return nil
}
