package stencil

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHilbertPoints(t *testing.T) {
	length := 4
	area := length * length
	points := make([]image.Point, area)
	WriteHilbertPoints(points, length)
	assert.Equal(t, []image.Point{
		// 0 1 E F
		// 3 2 D C
		// 4 7 8 B
		// 5 6 9 A
		{0, 0}, {1, 0}, {1, 1}, {0, 1},
		{0, 2}, {0, 3}, {1, 3}, {1, 2},
		{2, 2}, {2, 3}, {3, 3}, {3, 2},
		{3, 1}, {2, 1}, {2, 0}, {3, 0},
	}, points)
}

func TestHilbertRasterInt64(t *testing.T) {
	length := 4
	area := length * length
	table := make([][2]int, area)
	seq := make([]int64, area)
	raster := make([]int64, area)

	WriteSequenceInt64Vector(seq)
	WriteHilbertStencil3Table(table, length)
	RasterHilbertInt64Vector(raster, seq, length)
	assert.Equal(t, []int64{
		0, 1, 14, 15,
		3, 2, 13, 12,
		4, 7, 8, 11,
		5, 6, 9, 10,
	}, raster)
}

func TestWriteHilbertStencil3Table(t *testing.T) {
	{
		length := 2
		area := length * length
		table := make([][2]int, area)
		WriteHilbertStencil3Table(table, length)
		assert.Equal(t, [][2]int{
			//  0  1
			//  3  2
			{3, 1}, // 0
			{2, 0}, // 1
			{1, 3}, // 2
			{0, 2}, // 3
		}, table)
	}

	{
		length := 4
		area := length * length
		table := make([][2]int, area)
		WriteHilbertStencil3Table(table, length)
		assert.Equal(t, [][2]int{
			//  0  1  E  F
			//  3  2  D  C
			//  4  7  8  B
			//  5  6  9  A
			{1, 3},   // 0
			{14, 2},  // 1
			{13, 7},  // 2
			{2, 4},   // 3
			{7, 5},   // 4
			{6, 0},   // 5
			{9, 1},   // 6
			{8, 6},   // 7
			{11, 9},  // 8
			{10, 14}, // 9
			{5, 15},  // 10
			{4, 10},  // 11
			{3, 11},  // 12
			{12, 8},  // 13
			{15, 13}, // 14
			{0, 12},  // 15
		}, table)
	}
}
