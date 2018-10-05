package main

import (
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/braille"

)

var defaultBodyDef = bodyDef(braille.NewBitmapString('#',
	"##  hhHH  ##",
	"### hhHH ###",
	"  ##hhHH##  ",
	"  ##hhHH##  ",
	"LL ###### RR",
	"LL  #  #  RR",
	"LL  #  #  RR",
	"LL ###### RR",
	"  ##ttTT##  ",
	"  ##ttTT##  ",
	"### ttTT ###",
	"##  ttTT  ##",
), []bodyPartDef{
	{Point: image.Pt(0, 0)},
	{Point: image.Pt(1, 0)},
	{Point: image.Pt(2, 0)}, // h
	{Point: image.Pt(3, 0)}, // H
	{Point: image.Pt(4, 0)},
	{Point: image.Pt(5, 0)},
	{Point: image.Pt(0, 1)}, // L
	{Point: image.Pt(1, 1)},
	{Point: image.Pt(2, 1)},
	{Point: image.Pt(3, 1)},
	{Point: image.Pt(4, 1)},
	{Point: image.Pt(5, 1)}, // R
	{Point: image.Pt(0, 2)},
	{Point: image.Pt(1, 2)},
	{Point: image.Pt(2, 2)}, // t
	{Point: image.Pt(3, 2)}, // T
	{Point: image.Pt(4, 2)},
	{Point: image.Pt(5, 2)},
})

func bodyDef(bi *braille.Bitmap, parts []bodyPartDef) bodyDefinition {
	return bodyDefinition{bi, parts}
}

type bodyDefinition struct {
	*braille.Bitmap
	parts []bodyPartDef
}

type bodyPartDef struct {
	image.Point
}

type body struct {
	setup bool

	bi braille.Bitmap

	gridPos  []image.Point
	runes    []rune
	runeAttr []ansi.SGRAttr
}

func (bod *body) Init(defn *bodyDefinition) {
	if !bod.setup {
		if defn == nil {
			defn = &defaultBodyDef
		}
		bod.setup = true
	}
	if defn == nil {
		return
	}

	bod.gridPos = make([]image.Point, len(defn.parts))
	bod.runes = make([]rune, len(defn.parts))
	bod.runeAttr = make([]ansi.SGRAttr, len(defn.parts))
	for i, partDef := range defn.parts {
		bod.gridPos[i] = partDef.Point
	}

	bod.bi = *defn.Bitmap
	bod.bi.Bit = append(bod.bi.Bit[:0], bod.bi.Bit...)
}

func (bod *body) Size() image.Point { return bod.bi.RuneSize() }

func (bod *body) RenderInto(g *anansi.Grid, at image.Point, a ansi.SGRAttr) {
	bod.bi.CopyInto(g, at, true, a)
	for i, r := range bod.runes {
		if r != 0 {
			cell := g.Cell(at.Add(bod.gridPos[i]))
			cell.SetRune(r)
			if a := bod.runeAttr[i]; a != 0 {
				cell.SetAttr(a)
			}
		}
	}
}
