package stats_test

import (
	"bytes"
	"log"
	"math/rand"
	"strings"
	"testing"
	"time"

	"borkshop/stats"

	"github.com/stretchr/testify/assert"
)

func TestTimes_CountRecent(t *testing.T) {
	tc := struct {
		N        int
		Within   time.Duration
		Interval time.Duration
		Jitter   float64
	}{
		120,
		time.Second,
		time.Second / 60,
		0.1,
	}

	rand := rand.New(rand.NewSource(0))

	all := make([]time.Time, 0, 3*tc.N)
	now := time.Now() // TODO fix date?
	ts := stats.MakeTimes(tc.N)
	for len(all) < cap(all) {
		defer logBuf.Reset()

		// generate random interval
		jitter := tc.Jitter * (rand.Float64() - 0.5)
		elapsed := time.Duration(float64(tc.Interval) * (1.0 + jitter))

		// advance and record now
		now = now.Add(elapsed)
		all = append(all, now)

		// compute expectation the long way
		expected := 0
		for _, tm := range all {
			if now.Sub(tm) <= tc.Within {
				expected++
			}
		}

		// now drive the subject collection, and check its query response
		ts.Collect(now)
		if !assert.Equal(t, expected, ts.CountRecent(now, tc.Within)) {
			t.Logf("failed at %v/%v generated times", len(all), cap(all))
			dumpLogs(t)
			for i, tm := range all {
				t.Logf("all[%v]: -%v", i, now.Sub(tm))
			}
			break
		}
	}
}

var logBuf bytes.Buffer

func init() {
	log.SetOutput(&logBuf)
	log.SetFlags(0)
}

func dumpLogs(t *testing.T) {
	if logBuf.Len() > 0 {
		t.Logf("Log output:")
		lines := strings.Split(strings.TrimRight(logBuf.String(), "\n"), "\n")
		for _, line := range lines {
			t.Logf(line)
		}
		logBuf.Reset()
	}
}
