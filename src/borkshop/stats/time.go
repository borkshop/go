package stats

import (
	"math"
	"time"
)

// Durations implements a ringbuffer of collected time durations with an
// optional reporting function and interval.
type Durations struct {
	Report      func(db *Durations, i int)
	ReportEvery int

	d []time.Duration
	i int
}

func (db *Durations) Init(n, every int, report func(db *Durations, i int)) {
	db.i = 0
	db.d = make([]time.Duration, n)
	db.ReportEvery = every
	db.Report = report
}

func (db *Durations) Collect(d time.Duration) {
	if len(db.d) < cap(db.d) {
		i := len(db.d)
		db.d = append(db.d, d)
		db.observe(i)
	} else {
		db.d[db.i] = d
		db.i = (db.i + 1) % len(db.d)
		db.observe(db.i)
	}
}

func (db *Durations) observe(i int) {
	if db.Report != nil &&
		db.ReportEvery != 0 &&
		i%db.ReportEvery == 0 {
		db.Report(db, i)
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
