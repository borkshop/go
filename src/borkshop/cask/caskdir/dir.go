package caskdir

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"sort"

	"borkshop/cask"
	"borkshop/cask/caskblob"
	"borkshop/cask/caskio"

	billy "gopkg.in/src-d/go-billy.v4"
)

// Mode represents the type of a directory entry.
type Mode int16

const (
	// NoMode indicates an absent mode value.
	NoMode Mode = iota + 1
	// FileMode indicates a file.
	FileMode
	// DirMode indicates a directory.
	DirMode
	// ExecMode indicates an executable file.
	ExecMode
)

// entries are sorted by the byte value of their name to facilitate fast
// search.

// Entry represents an entry in a directory.
type Entry struct {
	// Hash is the SHA-256 hash of the root block of the entry's corresponding
	// B-tree.
	Hash cask.Hash

	// Mode is the type of the entry, file or directory.
	Mode Mode

	// Name is the name of the entry, as bytes.
	Name []byte
}

// Entries are ordered by name.

func (entry Entry) before(other Entry) bool {
	return bytes.Compare(entry.Name, other.Name) < 0
}

type entries []Entry

var _ sort.Interface = entries{}

func (e entries) Len() int {
	return len(e)
}

func (e entries) Less(i, j int) bool {
	return e[i].before(e[j])
}

func (e entries) Swap(i, j int) {
	e[j], e[i] = e[i], e[j]
}

// Store reads a directory tree from a filesystem and writes it as blocks to a
// content address store.
func Store(ctx context.Context, store cask.Store, fs billy.Filesystem, p string) (cask.Hash, error) {
	writer := caskio.NewWriter(store)

	dirEnts, err := fs.ReadDir(p)
	if err != nil {
		return cask.ZeroHash, err
	}

	entries := make(entries, 0, len(dirEnts))
	for _, dirEnt := range dirEnts {
		var err error
		mode := NoMode
		hash := cask.ZeroHash
		name := path.Join(p, dirEnt.Name())
		if dirEnt.IsDir() {
			mode = DirMode
			hash, err = Store(ctx, store, fs, name)
			if err != nil {
				return cask.ZeroHash, err
			}
		} else if dirEnt.Mode().IsRegular() {
			if dirEnt.Mode()&0111 == 0 {
				mode = FileMode
			} else {
				mode = ExecMode
			}
			reader, err := fs.Open(name)
			if err != nil {
				return cask.ZeroHash, err
			}
			hash, err = caskblob.Store(ctx, store, reader)
			if err != nil {
				return cask.ZeroHash, err
			}
		} else {
			continue
		}
		entries = append(entries, Entry{
			Name: []byte(dirEnt.Name()),
			Mode: mode,
			Hash: hash,
		})
	}

	sort.Sort(entries)
	for _, entry := range entries {
		if writer.Size()+cask.HashSize+4+len(entry.Name) > cask.BlockSize {
			if err := writer.Flush(ctx); err != nil {
				return cask.ZeroHash, err
			}
		}
		err := writer.Link(ctx, entry.Hash)
		if err != nil {
			return cask.ZeroHash, err
		}
		var buf [4]byte
		binary.BigEndian.PutUint16(buf[0:2], uint16(entry.Mode))
		binary.BigEndian.PutUint16(buf[2:4], uint16(len(entry.Name)))
		err = writer.Copy(ctx, buf[:])
		if err != nil {
			return cask.ZeroHash, err
		}
		err = writer.Copy(ctx, entry.Name)
		if err != nil {
			return cask.ZeroHash, err
		}
	}

	return writer.Sum(ctx)
}

// Load reads blocks from a content address store and builds a directory tree
// on a given filesystem.
func Load(ctx context.Context, store cask.Store, fs billy.Filesystem, p string, h cask.Hash) error {
	reader := caskio.NewReader(store, h)
	for {
		links, buf, err := reader.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		at := 0
		for _, link := range links {
			mode := Mode(binary.BigEndian.Uint16(buf[at : at+2]))
			namelen := int(binary.BigEndian.Uint16(buf[at+2 : at+4]))
			name := string(buf[at+4 : at+4+namelen])

			var perm os.FileMode
			switch mode {
			case DirMode, ExecMode:
				perm = 0755
			case FileMode:
				perm = 0644
			}

			switch mode {
			case DirMode:
				err := fs.MkdirAll(path.Join(p, name), perm)
				if err != nil {
					return err
				}
				err = Load(ctx, store, fs, path.Join(p, name), link)
				if err != nil {
					return err
				}
			case FileMode, ExecMode:
				writer, err := fs.OpenFile(path.Join(p, name), os.O_WRONLY|os.O_CREATE, perm)
				if err != nil {
					return err
				}
				err = caskblob.Load(ctx, store, writer, link)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("unexpected mode")
			}

			at += 4 + namelen
		}
	}
	return nil
}
