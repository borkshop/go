package bottle

const (
	maxUint = ^uint(0)
	minUint = 0
	maxInt  = int(maxUint >> 1)
	minInt  = -maxInt - 1
)

// Stats capture a min and max value for a dimension of the
// automaton.
type Stats struct {
	Min int
	Max int
}

// Reset spreads Min and Max to the farthest possible boundary
// values.
func (stats *Stats) Reset() {
	stats.Min = maxInt
	stats.Max = minInt
}

// Add accounts for a number in the collection, raising the max or
// lowering the min.
func (stats *Stats) Add(num int) {
	if num > stats.Max {
		stats.Max = num
	}
	if num < stats.Min {
		stats.Min = num
	}
}

// Spread returns the gap between the highest and lowest value.
func (stats Stats) Spread() int {
	return stats.Max - stats.Min
}

// Project projects a value in the statistical range into the target range.
func (stats Stats) Project(from, into int) int {
	spread := stats.Spread()
	if spread == 0 {
		return 0
	}
	return (from - stats.Min) * into / spread
}
