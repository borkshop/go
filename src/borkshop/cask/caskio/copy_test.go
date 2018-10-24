package caskio_test

import (
	"context"
	"testing"

	"borkshop/cask/caskdir"
	"borkshop/cask/caskio"
	"borkshop/cask/caskmemstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-billy.v4/memfs"
	"gopkg.in/src-d/go-billy.v4/osfs"
)

func TestCopy(t *testing.T) {
	// This test loads a ton of blocks from test data into an in-memory store,
	// then copies those blocks to another in-memory store, then extracts
	// the same blocks into an in-memory file system, verifying the hash
	// of the root is the same before and after the copy.

	ctx := context.Background()
	source := caskmemstore.New()
	target := caskmemstore.New()
	osfs := osfs.New("..")
	memfs := memfs.New()

	hash1, err := caskdir.Store(ctx, source, osfs, "testdata/nominal")
	require.NoError(t, err)

	err = caskio.Copy(ctx, target, source, hash1)
	require.NoError(t, err)

	err = caskdir.Load(ctx, target, memfs, ".", hash1)
	require.NoError(t, err)

	hash2, err := caskdir.Store(ctx, target, memfs, "")
	require.NoError(t, err)

	assert.Equal(t, hash1, hash2)
}
