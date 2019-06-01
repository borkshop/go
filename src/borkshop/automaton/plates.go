package main

import (
	"image"
	"image/color"
)

func MeasurePlateSizes(sizes []int64, plate []int64) {
	// Reset
	for i := 0; i < len(sizes); i++ {
		sizes[i] = 0
	}
	// Count
	for i := 0; i < len(plate); i++ {
		sizes[plate[i]]++
	}
}

func drawPlates(dst *image.RGBA, plates []int64, points []image.Point, colors []color.RGBA) {
	for i := 0; i < len(plates); i++ {
		plate := plates[i]
		pt := points[i]
		dst.SetRGBA(pt.X, pt.Y, colors[plate])
	}
}

func WriteNextPlateVector(plates []int64, plate5s [][5]int64, entropy []int64, sizes []int64, weights []int64) {
	area := len(plates)
	numPlates := len(sizes)
	for i := 0; i < area; i++ {
		srcStencil := plate5s[i]

		// Construct a histogram: the number of tickets each tectonic plate
		// enters in the lottery for the next generation of this cell.
		// Build weight histogram. (reset, count)
		for plate := 0; plate < numPlates; plate++ {
			weights[plate] = 0
		}
		for neighbor := 0; neighbor < 5; neighbor++ {
			weights[srcStencil[neighbor]]++
		}

		// Allocate ballots for the lottery.
		total := 0
		for plate := 0; plate < numPlates; plate++ {
			weight := int(weights[plate])
			// The weight of each plate type is the square of the number of
			// neighbors of that plate, times how rare the plate is
			// globally in the previous generation.
			handicap := area - int(sizes[plate]) + 1
			weight = weight * weight * handicap * handicap
			weights[plate] = int64(weight)
			total += weight
		}

		// Chose a random plate, weighted by local and global statistics.
		choice := int(uint64(entropy[i]) % uint64(total))
		plate := 0
		thresh := 0
		for ; plate < numPlates; plate++ {
			thresh += int(weights[plate])
			if choice < thresh {
				plates[i] = int64(plate)
				break
			}
		}
	}
}

func WriteRandomPlateVector(plate []int64, entropy []int64, numPlates int) {
	for i := 0; i < len(plate); i++ {
		plate[i] = int64(uint64(entropy[i]) % uint64(numPlates))
	}
}
