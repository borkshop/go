package main

func WatershedInt64Vector(waterDst, earthDst [][3]int64, totalWatershed, totalErode *int64, water [][3]int64, earth [][3]int64, entropy []int64) {
	for i := 0; i < len(waterDst); i++ {
		waterDst[i][0] = water[i][0]
		waterDst[i][1] = 0
		waterDst[i][2] = 0
		earthDst[i][0] = earth[i][0]
		earthDst[i][1] = 0
		earthDst[i][2] = 0

		flows := [2]int64{
			ShedInt64(water[i][0]/2, water[i][1]/2, earth[i][0], earth[i][1]),
			ShedInt64(water[i][0]/2, water[i][2]/2, earth[i][0], earth[i][2]),
		}
		flowMagnitudes := []uint64{
			uint64(mag64(flows[0])),
			uint64(mag64(flows[1])),
		}
		totalWatershedMagnitudes := flowMagnitudes[0] + flowMagnitudes[1]

		// Vote on direction based on relative magnitude.
		var choice int
		if totalWatershedMagnitudes == 0 {
			choice = int(entropy[i] & 1)
		} else if uint64(entropy[i])%totalWatershedMagnitudes < flowMagnitudes[0] {
			choice = 0
		} else {
			choice = 1
		}

		flow := flows[choice]
		waterDst[i][0] -= flow
		waterDst[i][1+choice] += flow
		*totalWatershed += int64(flowMagnitudes[choice])

		erode := mulFrac64(flow, 1, 3, uint64(entropy[0]))
		earthDst[i][0] -= erode
		earthDst[i][1+choice] += erode
		*totalErode += mag64(erode)
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

const handicap = 0xffffff

func AdjustWaterInt64Vector(water []int64, precipitation, evaporation *int64, precipitationControl int64, entropy []int64, volume int64) {
	if precipitationControl > 0 {
		for i := 0; i < len(water); i++ {
			volume := mulFrac64(volume, precipitationControl, 32, uint64(entropy[i]))
			water[i] += volume
			*precipitation += volume
		}
	}

	// Evaporation
	for i := 0; i < len(water); i++ {
		next := water[i] * 0xfe / 0xff
		diff := next - water[i]
		water[i] = next
		*evaporation -= diff
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
