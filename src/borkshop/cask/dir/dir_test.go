package caskdir_test

import (
	"context"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"borkshop/cask/dir"
	"borkshop/cask/memstore"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

func TestStoreDir(t *testing.T) {
	ctx := context.Background()
	store := caskmemstore.New()
	osfs := osfs.New("..")
	memfs := memfs.New()

	hash1, err := caskdir.Store(ctx, store, osfs, "testdata")
	require.NoError(t, err)

	err = caskdir.Load(ctx, store, memfs, ".", hash1)
	require.NoError(t, err)

	hash2, err := caskdir.Store(ctx, store, memfs, "")
	require.NoError(t, err)

	// Verify integrity of a directory listing.
	dir, err := memfs.ReadDir("nominal")
	require.NoError(t, err)
	require.Len(t, dir, 10)

	// Verify integrity of a directory listing.
	dir, err = memfs.ReadDir("nominal/0")
	require.NoError(t, err)
	require.Len(t, dir, 10)

	// Verify integrity of a nested file.
	f, err := memfs.Open("nominal/0/0.names")
	require.NoError(t, err)
	body, err := ioutil.ReadAll(f)
	require.NoError(t, err)
	lines := strings.Split(string(body), "\n")
	assert.Equal(t, "nil", lines[0])

	// Verify integrity of a large file.
	f, err = memfs.Open("firstredfirstand.txt")
	require.NoError(t, err)
	body, err = ioutil.ReadAll(f)
	require.NoError(t, err)
	lines = strings.Split(string(body), "\n")
	assert.Equal(t, "100000", lines[99999])

	assert.Equal(t, hash1, hash2)
}

func TestResolveAndList(t *testing.T) {
	ctx := context.Background()
	store := caskmemstore.New()
	osfs := osfs.New("..")

	dataHash, err := caskdir.Store(ctx, store, osfs, "testdata")
	require.NoError(t, err)

	nominalEntry, err := caskdir.Resolve(ctx, store, dataHash, "nominal/0")
	require.NoError(t, err)

	entries, err := caskdir.List(ctx, store, nominalEntry.Hash)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		assert.Equal(t, strconv.Itoa(i)+".names", string(entries[i].Name))
	}
}

func TestResolveNotFound(t *testing.T) {
	ctx := context.Background()
	store := caskmemstore.New()
	osfs := osfs.New("..")

	dataHash, err := caskdir.Store(ctx, store, osfs, "testdata")
	require.NoError(t, err)

	_, err = caskdir.Resolve(ctx, store, dataHash, "bogus")
	require.Error(t, err, "not found")
}

func TestResolveFileAsDir(t *testing.T) {
	ctx := context.Background()
	store := caskmemstore.New()
	osfs := osfs.New("..")

	dataHash, err := caskdir.Store(ctx, store, osfs, "testdata")
	require.NoError(t, err)

	_, err = caskdir.Resolve(ctx, store, dataHash, "nominal/0/0.names/bogus")
	require.Error(t, err, "not dir")
}
