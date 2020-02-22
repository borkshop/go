package caskmemstore_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"borkshop/cask"
	"borkshop/cask/memstore"
	"github.com/stretchr/testify/assert"
)

func TestMemStoreStoreThenLoad(t *testing.T) {
	b := cask.Block{1}
	h := b.Hash()

	store := caskmemstore.New()

	err := store.Store(context.Background(), h, &b)
	assert.NoError(t, err)

	var b2 cask.Block
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	err = store.Load(ctx, h, &b2)
	assert.NoError(t, err)
	cancel()

	var b3 cask.Block
	err = store.Load(context.Background(), h, &b3)
	assert.NoError(t, err)
	assert.Equal(t, b, b3)
}

func TestMemStoreLoadThenStore(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	b := cask.Block{1}
	h := b.Hash()

	var b1 cask.Block
	store := caskmemstore.New()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	err := store.Load(ctx, h, &b1)
	cancel()
	assert.Error(t, err, "should time out")

	go func() {
		time.Sleep(10 * time.Millisecond)
		err = store.Store(context.Background(), h, &b)
		assert.NoError(t, err)

		wg.Done()
	}()

	var b2 cask.Block
	err = store.Load(context.Background(), h, &b2)
	assert.NoError(t, err)
	assert.Equal(t, b, b2)

	wg.Wait()
}
