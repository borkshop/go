package main

import (
	"bytes"
	"image"
	"unicode/utf8"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

type popup struct {
	// TODO replace with anansi.Screen
	anansi.ScreenState
	buf anansi.Buffer

	active bool
	at     ansi.Point
}

func (pop *popup) setAt(at ansi.Point) {
	pop.at = at
}

func (pop *popup) drawInto(grid *anansi.Grid) {
	anansi.DrawGrid(grid.SubAt(pop.at), pop.Grid)
}

func (pop *popup) processBuf() {
	b := pop.buf.Bytes()
	sz := measureTextBounds(b)
	pop.ScreenState.Clear()
	pop.ScreenState.Resize(sz)
	pop.CursorState.Attr = ansi.SGRAttrClear | ansi.RGB(0x20, 0x20, 0x40).BG()
	pop.buf.Process(&pop.ScreenState)
}

func measureTextBounds(b []byte) (sz image.Point) {
	for i := 0; len(b) > 0; b = b[i+1:] {
		b = skipEscapes(b)
		if len(b) == 0 {
			break
		}
		sz.Y++
		if i = bytes.Index(b, []byte("\r\n")); i < 0 {
			if c := utf8.RuneCount(b); sz.X < c {
				sz.X = c
			}
			break
		}
		if c := utf8.RuneCount(b[:i]); sz.X < c {
			sz.X = c
		}
	}
	return sz
}

func skipEscapes(b []byte) []byte {
	for {
		_, _, n := ansi.DecodeEscape(b)
		if n == 0 {
			break
		}
		b = b[n:]
	}
	return b
}
