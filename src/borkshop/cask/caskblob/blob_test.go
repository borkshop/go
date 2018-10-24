package caskblob_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"

	"borkshop/cask/caskblob"
	"borkshop/cask/caskdiskstore"
	"borkshop/cask/caskmemstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/src-d/go-billy.v4/memfs"
)

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()

	file, err := os.Open("../testdata/firstredfirstand.txt")
	require.NoError(t, err)
	store := caskmemstore.New()
	h, err := caskblob.Store(ctx, store, file)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	err = caskblob.Load(ctx, store, &buf, h)
	require.NoError(t, err)

	expected, err := ioutil.ReadFile("../testdata/firstredfirstand.txt")
	require.NoError(t, err)
	assert.Equal(t, len(expected), len(buf.Bytes()), "round trip byte count")
	assert.Equal(t, expected, buf.Bytes(), "round trip")
}

func TestDiskStore(t *testing.T) {
	ctx := context.Background()

	file, err := os.Open("../testdata/firstredfirstand.txt")
	require.NoError(t, err)
	fs := memfs.New()
	store := &caskdiskstore.Store{Filesystem: fs}
	h, err := caskblob.Store(ctx, store, file)
	require.NoError(t, err)

	buf := bytes.Buffer{}
	err = caskblob.Load(ctx, store, &buf, h)
	require.NoError(t, err)

	expected, err := ioutil.ReadFile("../testdata/firstredfirstand.txt")
	require.NoError(t, err)
	assert.Equal(t, len(expected), len(buf.Bytes()), "round trip byte count")
	assert.Equal(t, expected, buf.Bytes(), "round trip")
}

func TestStringBytes(t *testing.T) {
	ctx := context.Background()
	store := caskmemstore.New()
	str1 := "Hello, World!\n"
	hash, err := caskblob.WriteString(ctx, store, str1)
	require.NoError(t, err)
	str2, err := caskblob.ReadString(ctx, store, hash)
	require.NoError(t, err)
	assert.Equal(t, str1, str2)
}
