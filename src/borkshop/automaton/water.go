package main

func WatershedInt64Vector(dst [][3]int64, flow *int64, water [][3]int64, earth [][3]int64, entropy []int64, other int) {
	*flow = 0
	for i := 0; i < len(dst); i++ {
		dst[i][0] = water[i][0]
		dst[i][1] = 0
		dst[i][2] = 0

		delta := ShedInt64(water[i][0]/2, water[i][other]/2, earth[i][0], earth[i][other])
		dst[i][0] -= delta
		dst[i][other] += delta
		*flow += mag64(delta)
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

func AdjustWaterInt64Vector(water []int64, control int64, entropy []int64, volume int64) {
	switch {
	case control > 0:
		for i := 0; i < len(water); i++ {
			if uint64(entropy[i])&0xffffffff < uint64(control) {
				water[i] += volume
			}
		}
	case control < 0:
		for i := 0; i < len(water); i++ {
			if uint64(entropy[i])&0xffffffff < uint64(-control) {
				if water[i] > volume {
					water[i] -= volume
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
