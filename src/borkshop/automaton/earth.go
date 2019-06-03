package main

import (
	"image"
	"math"
)

const quakeVectorResolution = 0x1000

func WriteQuakeVectors(vectors []image.Point) {
	numPlates := len(vectors)
	arc := math.Pi * 2 / float64(numPlates)
	for i := 0; i < numPlates; i++ {
		vector := arc * float64(i)
		x := int(math.Cos(vector) * quakeVectorResolution)
		y := int(math.Sin(vector) * quakeVectorResolution)
		vectors[i] = image.Point{X: x, Y: y}
	}
}

func Quake(dst [][3]int64, quake *int64, src [][3]int64, plates []int64, quakeVectors []image.Point, num, den int64, sig uint, entropy []int64) {
	*quake = 0
	mag := clamp64(num>>sig, 1, den)
	for i := 0; i < len(dst); i++ {
		dst[i][0] = src[i][0]
		dst[i][1] = 0
		dst[i][2] = 0
		if (num>>sig) > 0 || uint64(entropy[i])%uint64(den>>sig) < uint64(num) {
			plate := plates[i]
			vector := quakeVectors[plate]
			x, y := int64(vector.X), int64(vector.Y)
			mag := int64(uint64(entropy[i]) % uint64(mag))
			if entropy[i]%(mag64(x)+mag64(y)) < mag64(x) {
				del := clamp64(x, -mag, mag)
				dst[i][0] -= del
				dst[i][1] += del
				*quake += mag64(del)
			} else {
				del := clamp64(y, -mag, mag)
				dst[i][0] -= del
				dst[i][2] += del
				*quake += mag64(del)
			}
		}
	}
}

func SlideInt64Vector(dst [][3]int64, slide *int64, src [][3]int64, repose []int64, entropy []int64, mute int64, other int) {
	for i := 0; i < len(dst); i++ {
		dst[i][0] = src[i][0]
		dst[i][1] = 0
		dst[i][2] = 0

		delta := SlideInt64(src[i][0], src[i][other], repose[i]) / mute
		dst[i][0] -= delta
		dst[i][other] += delta
		*slide += mag64(delta)
	}
}

// SlideInt64 takes two columns and normalizes them about their mean such
// that the differences in the heights of the column does not exceed
// the angle of repose.
func SlideInt64(left, right, repose int64) int64 {
	mean := (left+right)/2 - repose/2
	if left > right {
		return mean - right
	} else {
		return left - mean
	}
}
