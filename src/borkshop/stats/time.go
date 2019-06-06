package stats

import (
	"math"
	"time"
)

// Durations implements a ringbuffer of collected time durations.
type Durations struct {
	d []time.Duration
	i int
}

func (ds *Durations) Init(n int) {
	ds.i = 0
	ds.d = make([]time.Duration, 0, n)
}

func (ds *Durations) Measure() func() {
	t0 := time.Now()
	return func() {
		t1 := time.Now()
		ds.Collect(t1.Sub(t0))
	}
}

func (ds *Durations) Collect(d time.Duration) {
	if len(ds.d) < cap(ds.d) {
		ds.d = append(ds.d, d)
	} else {
		ds.d[ds.i] = d
		ds.i = (ds.i + 1) % len(ds.d)
	}
}

func (ds *Durations) Count() int {
	return len(ds.d)
}

func (ds *Durations) Total() time.Duration {
	var total time.Duration
	for _, d := range ds.d {
		total += d
	}
	return total
}

func (ds *Durations) Average() time.Duration {
	return time.Duration(math.Round(float64(ds.Total()) / float64(ds.Count())))
}
