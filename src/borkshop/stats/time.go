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

func (db *Durations) Init(n int) {
	db.i = 0
	db.d = make([]time.Duration, 0, n)
}

func (db *Durations) Measure() func() {
	t0 := time.Now()
	return func() {
		t1 := time.Now()
		db.Collect(t1.Sub(t0))
	}
}

func (db *Durations) Collect(d time.Duration) {
	if len(db.d) < cap(db.d) {
		db.d = append(db.d, d)
	} else {
		db.d[db.i] = d
		db.i = (db.i + 1) % len(db.d)
	}
}

func (db *Durations) Count() int {
	return len(db.d)
}

func (db *Durations) Total() time.Duration {
	var total time.Duration
	for _, d := range db.d {
		total += d
	}
	return total
}

func (db *Durations) Average() time.Duration {
	return time.Duration(math.Round(float64(db.Total()) / float64(db.Count())))
}
