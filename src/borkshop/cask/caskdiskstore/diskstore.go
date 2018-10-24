// Package caskdiskstore provides a filesystem-backed content address store.
//
// This implementation stores one block per file and supports concurrent
// reads and writes to the same block across processes by writing to temporary
// files and renaming them atomically.
//
// Ideally blocks would be sized to match the block size of the underlying file
// system, but this is not likely 1KB.
// Instead, it would be good for multiple 1KB blocks to share a filesystem block
// using a common SHA-256 hash prefix filename, but this complicates concurrent
// reads and writes accross multiple processes.
package caskdiskstore

// TODO Load should watch or retry reading until the context deadline.
// Load should assume that multiple processes may concurrently read and write
// the underlying file.

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"path"

	"borkshop/cask"
	"go.uber.org/multierr"
	billy "gopkg.in/src-d/go-billy.v4"
)

// Store is a CAS block store that uses a directory tree to store the
// blocks, using their hashes to denote their file name.
type Store struct {
	// Filesystem is an object representing the directory to use for storage.
	Filesystem billy.Filesystem
}

var _ cask.Store = (*Store)(nil)

// Store writes a block to the content address store.
//
// Store is concurrency safe assuming no hash collisions, because it writes
// first to a temporary file then renames the file to its final location.
func (s *Store) Store(ctx context.Context, h cask.Hash, b *cask.Block) error {
	hex := hex.EncodeToString(h[:])
	prefix := hex[0:2]
	suffix := hex[2:]

	err := s.Filesystem.MkdirAll(prefix, 0755)
	if err != nil {
		return err
	}

	temp, err := s.Filesystem.TempFile(prefix, suffix)
	if err != nil {
		return err
	}

	_, err = temp.Write(b[:])
	if err != nil {
		err = multierr.Append(err, s.Filesystem.Remove(temp.Name()))
		return err
	}

	loc := path.Join(prefix, suffix)
	return multierr.Append(err, s.Filesystem.Rename(temp.Name(), loc))
}

// Load reads a block from the content address store.
func (s *Store) Load(ctx context.Context, h cask.Hash, b *cask.Block) error {
	hex := hex.EncodeToString(h[:])
	prefix := hex[0:2]
	suffix := hex[2:]
	loc := path.Join(prefix, suffix)

	file, err := s.Filesystem.Open(loc)
	if err != nil {
		return err
	}

	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	copy(b[:], buf)
	return nil
}
