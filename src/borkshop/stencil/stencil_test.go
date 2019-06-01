package stencil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSequenceAndEraseInt64Vector(t *testing.T) {
	area := 3
	seq := make([]int64, area)

	WriteSequenceInt64Vector(seq)
	assert.Equal(t, []int64{0, 1, 2}, seq)

	EraseInt64Vector(seq)
	assert.Equal(t, []int64{0, 0, 0}, seq)
}

func TestStencil3RoundTrip(t *testing.T) {
	length := 4
	area := length * length
	table := make([][2]int, area)
	before := make([]int64, area)
	before3s := make([][3]int64, area)
	after3s := make([][3]int64, area)
	after := make([]int64, area)

	WriteSequenceInt64Vector(before)
	WriteHilbertStencil3Table(table, length)
	WriteStencil3Int64Vector(before3s, before, table)
	takePlusAndMinus(after3s, before3s, table)
	AddInt64VectorStencil3(after, after3s, table)

	assert.Equal(t, before, after)
}

func takePlusAndMinus(dst [][3]int64, src [][3]int64, table [][2]int) {
	for i := 0; i < len(dst); i++ {
		dst[i][0] = src[i][0]
		dst[i][1] = 1
		dst[i][2] = -1
	}
}

func TestStencil3ShiftWest(t *testing.T) {
	length := 4
	area := length * length
	table := make([][2]int, area)
	before := make([]int64, area)
	before3s := make([][3]int64, area)
	after3s := make([][3]int64, area)
	after := make([]int64, area)
	raster := make([]int64, area)

	WriteHilbertStencil3Table(table, length)

	WriteSequenceInt64Vector(before)
	RasterHilbertInt64Vector(raster, before, length)
	assert.Equal(t, []int64{
		0, 1, 14, 15,
		3, 2, 13, 12,
		4, 7, 8, 11,
		5, 6, 9, 10,
	}, raster)

	WriteStencil3Int64Vector(before3s, before, table)
	shiftWest(after3s, before3s, table)
	AddInt64VectorStencil3(after, after3s, table)
	RasterHilbertInt64Vector(raster, after, length)
	assert.Equal(t, []int64{
		1, 14, 15, 0,
		2, 13, 12, 3,
		7, 8, 11, 4,
		6, 9, 10, 5,
	}, raster)
}

func shiftWest(dst [][3]int64, src [][3]int64, table [][2]int) {
	for i := 0; i < len(dst); i++ {
		dst[i][0] = src[i][1]
	}
}

func TestStencil3ShiftSouth(t *testing.T) {
	length := 4
	area := length * length
	table := make([][2]int, area)
	before := make([]int64, area)
	before3s := make([][3]int64, area)
	after3s := make([][3]int64, area)
	after := make([]int64, area)
	raster := make([]int64, area)

	WriteHilbertStencil3Table(table, length)

	WriteSequenceInt64Vector(before)
	RasterHilbertInt64Vector(raster, before, length)
	assert.Equal(t, []int64{
		0, 1, 14, 15,
		3, 2, 13, 12,
		4, 7, 8, 11,
		5, 6, 9, 10,
	}, raster)

	WriteStencil3Int64Vector(before3s, before, table)
	shiftSouth(after3s, before3s, table)
	AddInt64VectorStencil3(after, after3s, table)
	RasterHilbertInt64Vector(raster, after, length)
	assert.Equal(t, []int64{
		5, 6, 9, 10,
		0, 1, 14, 15,
		3, 2, 13, 12,
		4, 7, 8, 11,
	}, raster)
}

func shiftSouth(dst [][3]int64, src [][3]int64, table [][2]int) {
	for i := 0; i < len(dst); i++ {
		dst[i][2] = src[i][0]
	}
}
