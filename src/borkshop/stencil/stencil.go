// Package stencil provides functions for manipulating vectors using small read
// and write stencils.
//
// Stencil9:
//
//   +---+---+---+
//   | 6 | 3 | 7 |
//   +---+---+---+
//   | 2 |   | 0 |
//   +---+---+---+
//   | 5 | 1 | 4 |
//   +---+---+---+
//
// Stencil5:
//
//       +---+
//       | 3 |
//   +---+---+---+
//   | 2 |   | 0 |
//   +---+---+---+
//       | 1 |
//       +---+
//
// Stencil3:
//
//       +---+---+
//       |   | 0 |
//       +---+---+
//       | 1 |
//       +---+

package stencil

func WriteSequenceInt64Vector(dst []int64) {
	for i := 0; i < len(dst); i++ {
		dst[i] = int64(i)
	}
}

func EraseInt64Vector(dst []int64) {
	InitInt64Vector(dst, 0)
}

func InitInt64Vector(dst []int64, num int64) {
	for i := 0; i < len(dst); i++ {
		dst[i] = num
	}
}

func WriteStencil3Int64Vector(dst [][3]int64, src []int64, table [][2]int) {
	for i, stencil := range table {
		e, s := stencil[0], stencil[1]
		dst[i][0] = src[i]
		dst[i][1] = src[e]
		dst[i][2] = src[s]
	}
}

func AddInt64VectorStencil3(dst []int64, src [][3]int64, table [][2]int) {
	for i, stencil := range table {
		e, s := stencil[0], stencil[1]
		dst[i] += src[i][0]
		dst[e] += src[i][1]
		dst[s] += src[i][2]
	}
}

func WriteStencil5Int64Vector(dst [][5]int64, src []int64, table [][4]int) {
	for i, stencil := range table {
		dst[i][0] = src[i]
		for j := 0; j < 4; j++ {
			dst[i][j+1] = src[stencil[j]]
		}
	}
}

func AddInt64VectorStencil5(dst []int64, src [][9]int64, table [][4]int) {
	for i, stencil := range table {
		dst[i] += src[i][0]
		for j := 0; j < 4; j++ {
			dst[stencil[j]] += src[i][j+1]
		}
	}
}

func WriteStencil9Int64Vector(dst [][9]int64, src []int64, table [][8]int) {
	for i, stencil := range table {
		dst[i][0] = src[i]
		for j := 0; j < 8; j++ {
			dst[i][j+1] = src[stencil[j]]
		}
	}
}

func AddInt64VectorStencil9(dst []int64, src [][9]int64, table [][8]int) {
	for i, stencil := range table {
		dst[i] += src[i][0]
		for j := 0; j < 8; j++ {
			dst[stencil[j]] += src[i][j+1]
		}
	}
}
