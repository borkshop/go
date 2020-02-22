package casktempstore_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"borkshop/cask"
	"borkshop/cask/tempstore"
	"github.com/stretchr/testify/assert"
)

func TestTempStoreStoreThenLoad(t *testing.T) {
	store := casktempstore.New()

	// Run multiple times to cover the recycler.
	for i := 0; i < 3; i++ {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()

			b1 := cask.Block{1}
			h1 := b1.Hash()

			b2 := cask.Block{2}
			h2 := b2.Hash()

			err := store.Store(ctx, h1, &b1)
			assert.NoError(t, err)

			err = store.Store(ctx, h2, &b2)
			assert.NoError(t, err)

			var b1a cask.Block
			err = store.Load(ctx, h1, &b1a)
			assert.NoError(t, err)
			assert.Equal(t, b1, b1a)

			var b1b cask.Block
			err = store.Load(ctx, h1, &b1b)
			assert.NoError(t, err)
			assert.Equal(t, b1, b1b)

			var b2a cask.Block
			err = store.Load(ctx, h2, &b2a)
			assert.NoError(t, err)
			assert.Equal(t, b2, b2a)

			<-ctx.Done()
		})
	}
}

func TestTempStoreLoadThenStore(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	b := cask.Block{1}
	h := b.Hash()

	var b1 cask.Block
	store := casktempstore.New()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	err := store.Load(ctx, h, &b1)
	cancel()
	assert.Error(t, err, "should time out")

	go func() {
		time.Sleep(10 * time.Millisecond)
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		err = store.Store(ctx, h, &b)
		assert.NoError(t, err)

		wg.Done()
	}()

	var b2 cask.Block
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = store.Load(ctx, h, &b2)
	assert.NoError(t, err)
	assert.Equal(t, b, b2)

	wg.Wait()
}
