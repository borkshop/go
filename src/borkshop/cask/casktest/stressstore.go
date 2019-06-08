package casktest

import (
	"borkshop/cask"
	"borkshop/cask/caskblob"
	"context"
	"fmt"
	"time"
)

const str = "Hello, CAS1KB!"

type StressStoreReport struct {
	Cycles      int
	WriteErrors int
	ReadErrors  int
	DataErrors  int
}

func (r *StressStoreReport) merge(s *StressStoreReport) {
	r.Cycles += s.Cycles
	r.WriteErrors += s.WriteErrors
	r.ReadErrors += s.ReadErrors
	r.DataErrors += s.ReadErrors
}

type StressStoreConfig struct {
	Concurrency int
	Duration    time.Duration
}

func (c StressStoreConfig) Stress(store cask.Store) *StressStoreReport {
	report := &StressStoreReport{}
	done := make(chan struct{}, 0)
	reports := make(chan *StressStoreReport, 0)

	for i := 0; i < c.Concurrency; i++ {
		go worker(done, reports, store)
	}

	time.Sleep(c.Duration)
	close(done)

	for i := 0; i < c.Concurrency; i++ {
		report.merge(<-reports)
	}
	return report
}

func worker(done <-chan struct{}, reports chan<- *StressStoreReport, store cask.Store) {
	report := &StressStoreReport{}
	defer func() {
		reports <- report
	}()

	ctx := context.Background()
	for {
		select {
		case <-done:
			return
		default:
		}

		hash, err := caskblob.WriteString(ctx, store, str)
		if err != nil {
			fmt.Printf("%v\n", err)
			report.WriteErrors++
			continue
		}

		rst, err := caskblob.ReadString(ctx, store, hash)
		if err != nil {
			fmt.Printf("%v\n", err)
			report.ReadErrors++
			continue
		}

		if rst != str {
			report.DataErrors++
			continue
		}

		report.Cycles++
	}
}
