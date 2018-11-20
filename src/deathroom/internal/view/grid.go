package view

import (
	"fmt"
	"image"
	"unicode/utf8"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

// Grid represents a sized buffer of terminal cells.
type Grid struct {
	anansi.Grid
}

// MakeGrid makes a new Grid with the given size.
func MakeGrid(sz image.Point) Grid {
	var g Grid
	g.Resize(sz)
	return g
}

// Merge merges data into a cell in the grid.
func (g Grid) Merge(pt ansi.Point, ch rune, attr ansi.SGRAttr) {
	if i, valid := g.CellOffset(pt); valid {
		if ch != 0 {
			g.Rune[i] = ch
		}
		if attr != 0 {
			g.Attr[i] = g.Attr[i].Merge(attr)
		}
	}
}

// Copy copies another grid into this one, centered and clipped as necessary.
func (g Grid) Copy(og Grid) {
	var (
		gsz    = g.Bounds().Size()
		ogsz   = og.Bounds().Size()
		offset = gsz.Sub(ogsz).Div(2)
	)
	anansi.DrawGrid(
		g.SubAt(ansi.PtFromImage(offset)),
		og.Grid,
	)
}

// WriteString writes a string into the grid at the given position, returning
// how many cells were affected.
func (g Grid) WriteString(pt ansi.Point, mess string, args ...interface{}) int {
	if len(args) > 0 {
		mess = fmt.Sprintf(mess, args...)
	}
	var n int
	if i, valid := g.CellOffset(pt); valid {
		gsz := g.Bounds().Size()
		for j := i; len(mess) > 0 && pt.X < gsz.X; {
			r, n := utf8.DecodeRuneInString(mess)
			mess = mess[n:]
			g.Rune[j] = r
			pt.X++
			j++
			n++
		}
	}
	return n
}

// WriteStringRTL is like WriteString except it gose Right-To-Left (in both the
// string and the grid).
func (g Grid) WriteStringRTL(pt ansi.Point, mess string, args ...interface{}) int {
	if len(args) > 0 {
		mess = fmt.Sprintf(mess, args...)
	}
	var n int
	if i, valid := g.CellOffset(pt); valid {
		for j := i; len(mess) > 0 && pt.X > 0; {
			r, n := utf8.DecodeLastRuneInString(mess)
			mess = mess[:len(mess)-n]
			g.Rune[j] = r
			pt.X--
			j--
			n++
		}
	}
	return n
}

// Lines returns a slice of row strings from the grid, filling in any
// zero runes with the given one.
func (g Grid) Lines(fillZero rune) []string {
	gsz := g.Bounds().Size()
	lines := make([]string, gsz.Y)
	line := make([]rune, gsz.X)
	for y, i := 1, 0; y < gsz.Y; y++ {
		for x := 1; x < gsz.X; x++ {
			if ch := g.Rune[i]; ch != 0 {
				line[x] = ch
			} else {
				line[x] = fillZero
			}
			i++
		}
		lines[y] = string(line)
	}
	return lines
}
