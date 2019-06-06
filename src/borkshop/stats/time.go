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

func MakeDurations(n int) Durations {
	return Durations{i: 0, d: make([]time.Duration, 0, n)}
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

// Times implements a ringbuffer of collected times.
type Times struct {
	t []time.Time
	i int
}

func MakeTimes(n int) Times {
	return Times{i: 0, t: make([]time.Time, 0, n)}
}

func (ts *Times) Collect(t time.Time) {
	if len(ts.t) < cap(ts.t) {
		ts.t = append(ts.t, t)
	} else {
		ts.t[ts.i] = t
		ts.i = (ts.i + 1) % len(ts.t)
	}
}

func (ts *Times) CountRecent(now time.Time, within time.Duration) int {
	i, j := 0, len(ts.t)
	if len(ts.t) == cap(ts.t) {
		// when in normal full-ring mode, we start at the next position (the
		// oldest value), and wrap back around until it.
		i += ts.i
		j += ts.i
	}

	for j-i > 1 {
		h := (i/2 + j/2)
		since := now.Sub(ts.t[h%len(ts.t)])
		if since <= within {
			j = h
		} else if i != h {
			i = h
		} else {
			i++
		}
	}
	if now.Sub(ts.t[i%len(ts.t)]) > within {
		i = j
	}

	return ts.i + len(ts.t) - i
}
