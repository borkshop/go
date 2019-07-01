package main

func mag64(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}

func signMag64(n int64) (int64, int64) {
	if n < 0 {
		return -1, -n
	}
	return 1, n
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

func mulFrac64(num, fac int64, fractionalBits uint, entropy uint64) int64 {
	sign, mag := signMag64(num)
	product := mag * fac
	whole := product >> fractionalBits
	mask := uint64(1<<fractionalBits) - 1
	var part int64
	if uint64(product)&mask > entropy&mask {
		part = 1
	}
	return sign * (whole + part)
}
