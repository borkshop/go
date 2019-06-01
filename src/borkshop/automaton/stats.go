package main

import "fmt"

const (
	maxUint64 = ^uint(0)
	minUint64 = 0
	maxInt64  = int64(maxUint64 >> 1)
	minInt64  = -maxInt64 - 1
)

// Stats64 capture a min and max value for a dimension of the
// automaton.
type Stats64 struct {
	Min   int64
	Max   int64
	Num   int64
	Total int64
}

// Reset spreads Min and Max to the farthest possible boundary
// values.
func (stats *Stats64) Reset() {
	stats.Min = maxInt64
	stats.Max = minInt64
	stats.Total = 0
	stats.Num = 0
}

// Add accounts for a number in the collection, raising the max or
// lowering the min.
func (stats *Stats64) Add(num int64) {
	if num > stats.Max {
		stats.Max = num
	}
	if num < stats.Min {
		stats.Min = num
	}
	stats.Num++
	stats.Total += num
}

// Spread returns the gap between the highest and lowest value.
func (stats *Stats64) Spread() int64 {
	return stats.Max - stats.Min
}

// Project projects a value in the statistical range into the target range.
func (stats *Stats64) Project(from, into int64) int64 {
	spread := stats.Spread()
	if spread == 0 {
		return 0
	}
	return (from - stats.Min) * into / spread
}

func (stats *Stats64) Mean() float64 {
	if stats.Num == 0 {
		return 0
	}
	return float64(stats.Total) / float64(stats.Num)
}

func (stats *Stats64) String() string {
	return fmt.Sprintf("%d...%f...%d", stats.Min, stats.Mean(), stats.Max)
}

func WriteStatsFromInt64Vector(stats *Stats64, nums []int64) {
	stats.Reset()
	for _, num := range nums {
		stats.Add(num)
	}
}
