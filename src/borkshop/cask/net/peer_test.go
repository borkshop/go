package casknet_test

import (
	"context"
	"testing"

	"borkshop/cask"
	"borkshop/cask/memstore"
	"borkshop/cask/net"
	"github.com/stretchr/testify/require"
)

func TestCasknet(t *testing.T) {
	ctx := context.Background()

	server := &casknet.Server{
		Addr:  "127.0.0.1:0",
		Store: caskmemstore.New(),
	}
	err := server.Start(ctx)
	require.NoError(t, err)
	defer server.Stop(ctx)

	client := &casknet.Server{
		Addr:  "127.0.0.1:0",
		Store: caskmemstore.New(),
	}
	err = client.Start(ctx)
	require.NoError(t, err)
	defer client.Stop(ctx)

	// Create a block
	storedmodel := &cask.Model{}
	storedmodel.AppendString("hello world!\n")
	storedblock := &cask.Block{}
	err = storedmodel.Put(storedblock)
	require.NoError(t, err)

	peer := client.Peer(server.LocalAddr())

	err = peer.Store(ctx, storedblock.Hash(), storedblock)
	require.NoError(t, err)

	loadedblock := &cask.Block{}
	err = peer.Load(ctx, storedblock.Hash(), loadedblock)
	require.NoError(t, err)

	require.Equal(t, storedblock, loadedblock)
}
