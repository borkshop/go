package main

func mag64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func clamp64(n, min, max int64) int64 {
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
