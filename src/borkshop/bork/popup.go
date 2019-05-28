package main

import (
	"bytes"
	"image"
	"unicode/utf8"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
)

type popup struct {
	anansi.ScreenDiffer
	active bool
	at     ansi.Point
}

func (pop *popup) drawInto(grid *anansi.Grid) {
	anansi.DrawGrid(grid.SubAt(pop.at), pop.Grid)
}

func (pop *popup) Reset() {
	pop.active = false
	pop.at = ansi.ZP
	pop.ScreenDiffer.Reset()
}

func (pop *popup) Reload(contents []byte, at ansi.Point) {
	// TODO satisfice size wrt at within bounds
	sz := measureTextBounds(contents)
	// +1,1 because at is the location of the subject, which we don't want to occlude
	pop.at = at.Add(image.Pt(1, 1))
	pop.ScreenDiffer.Reset()
	pop.ScreenDiffer.Resize(sz)
	pop.Cursor.Attr = ansi.SGRAttrClear | ansi.RGB(0x20, 0x20, 0x40).BG()
	pop.ScreenDiffer.Write(contents)
	pop.active = true
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
