package casktempstore

import (
	"container/heap"
	"context"
	"fmt"
	"time"

	"borkshop/cask"
)

type collector struct {
	deadlines map[cask.Hash]deadline
	heap      []cask.Hash
}

type deadline struct {
	index int
	time  time.Time
}

var _ heap.Interface = (*collector)(nil)

func newCollector() *collector {
	return &collector{
		deadlines: make(map[cask.Hash]deadline),
	}
}

func (c *collector) Touch(ctx context.Context, hash cask.Hash) error {
	time, ok := ctx.Deadline()
	if !ok {
		return fmt.Errorf("temporary store requires a context deadline")
	}
	if other, ok := c.deadlines[hash]; ok {
		// Update
		if other.time.Before(time) {
			c.deadlines[hash] = deadline{
				index: other.index,
				time:  time,
			}
			heap.Fix(c, other.index)
		}
	} else {
		// Create
		index := len(c.heap)
		c.deadlines[hash] = deadline{
			index: index,
			time:  time,
		}
		c.heap = append(c.heap, hash)
		heap.Fix(c, index)
	}
	return nil
}

func (c *collector) Collect(cells map[cask.Hash]*cell) {
	now := time.Now() // TODO parameterize for tests
	for len(c.heap) > 0 {
		hash := c.heap[0]
		deadline := c.deadlines[hash]
		if deadline.time.Before(now) {
			last := len(c.heap) - 1
			if last > 0 {
				c.Swap(0, last)
			}

			c.heap = c.heap[:last]
			delete(c.deadlines, hash)
			delete(cells, hash)

			if len(c.heap) > 0 {
				heap.Fix(c, 0)
			}
		} else {
			return
		}
	}
}

func (c *collector) Len() int {
	return len(c.heap)
}

func (c *collector) Swap(i, j int) {
	// hashes
	o := c.heap[i]
	p := c.heap[j]
	// fix indexes
	c.deadlines[o] = deadline{
		time:  c.deadlines[o].time,
		index: j,
	}
	c.deadlines[p] = deadline{
		time:  c.deadlines[p].time,
		index: i,
	}
	// fix heap
	c.heap[j], c.heap[i] = c.heap[i], c.heap[j]
}

func (c *collector) Less(i, j int) bool {
	// hashes
	o := c.heap[i]
	p := c.heap[j]
	// times
	t := c.deadlines[o]
	u := c.deadlines[p]
	return t.time.Before(u.time)
}

func (c *collector) Pop() interface{} {
	panic("assertion failed: heap pop not implemented in casktempstore")
}

func (c *collector) Push(interface{}) {
	panic("assertion failed: heap push not implemented in casktempstore")
}
