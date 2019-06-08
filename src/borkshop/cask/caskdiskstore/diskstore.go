// Package caskdiskstore provides a filesystem-backed content address store.
//
// This implementation stores one block per file and supports concurrent
// reads and writes to the same block across processes by writing to temporary
// files and renaming them atomically.
//
// Ideally blocks would be sized to match the block size of the underlying file
// system, though this is not likely 1KB.
// But, who are we kidding?
// Blocks are a figment of a modern filesystem's imagination anyway.
package caskdiskstore

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"
	"time"

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
	loc := path.Join(prefix, suffix)

	err := s.Filesystem.MkdirAll(prefix, 0755)
	if err != nil {
		return err
	}

	// Retry loop
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// If the block already exists, chances are it contains the same data
		// we would write, so let's just all agree we did the work and go home
		// early.
		_, err := s.Filesystem.Stat(loc)
		if err == nil {
			return nil
		}

		// Race to set up a temporary write space for our block.
		// We use a scratch so readers can't open the file until we have
		// finished writing.
		temp, err := s.Filesystem.OpenFile(loc+".partial", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			// Yielding should be more than enough of a delay to allow our
			// opponent to finish writing their 1KB blob.
			time.Sleep(0)
			continue
		}

		// Write to scratch
		_, err = temp.Write(b[:])
		if err != nil {
			// We do try to clean up if we fail, so someone might succeed in
			// our stead afterward.
			// Would be a shame if we panicked or someone pulled the plug
			// leaving that .partial file laying around forever, preventing
			// this block from ever being written.
			err = multierr.Append(err, s.Filesystem.Remove(temp.Name()))
			return err
		}

		return multierr.Append(err, s.Filesystem.Rename(temp.Name(), loc))
	}
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
