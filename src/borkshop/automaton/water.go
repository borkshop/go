package main

func WatershedInt64Vector(dst [][3]int64, flow *int64, water [][3]int64, earth [][3]int64, entropy []int64) {
	for i := 0; i < len(dst); i++ {
		dst[i][0] = water[i][0]
		dst[i][1] = 0
		dst[i][2] = 0

		deltas := [2]int64{
			ShedInt64(water[i][0]/2, water[i][1]/2, earth[i][0], earth[i][1]),
			ShedInt64(water[i][0]/2, water[i][2]/2, earth[i][0], earth[i][2]),
		}
		magnitudes := []uint64{
			uint64(mag64(deltas[0])),
			uint64(mag64(deltas[1])),
		}
		total := magnitudes[0] + magnitudes[1]

		// Vote on direction based on relative magnitude.
		var choice int
		if total == 0 || uint64(entropy[i])%total < magnitudes[0] {
			choice = 0
		} else {
			choice = 1
		}
		delta := deltas[choice]

		dst[i][0] -= delta
		dst[i][1+choice] += delta
		*flow += int64(magnitudes[choice])
	}
}

func ShedInt64(leftWater, rightWater, leftEarth, rightEarth int64) int64 {
	left := leftEarth + leftWater
	right := rightEarth + rightWater
	mean := (left + right) / 2
	if left > right {
		flow := left - mean
		if flow > leftWater {
			return leftWater
		}
		return flow
	} else if right > left {
		flow := mean - right
		if -flow > rightWater {
			return -rightWater
		}
		return flow
	}
	return 0
}

func AdjustWaterInt64Vector(water []int64, changed *int64, control int64, entropy []int64, volume int64) {
	switch {
	case control > 0:
		for i := 0; i < len(water); i++ {
			if uint64(entropy[i])&0xffffffff < uint64(control) {
				water[i] += volume
				*changed += volume
			}
		}
	case control < 0:
		for i := 0; i < len(water); i++ {
			if uint64(entropy[i])&0xffffffff < uint64(-control) {
				if water[i] > volume {
					water[i] -= volume
					*changed -= volume
				}
			}
		}
	}
}

func MeasureWaterCoverage(coverage *int64, waters []int64, min int64) {
	var total int64
	for _, water := range waters {
		if water > min {
			total++
		}
	}
	*coverage = total
}
