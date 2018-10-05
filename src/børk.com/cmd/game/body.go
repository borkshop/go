package main

import (
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/braille"

	"børk.com/ecs"
)

const (
	bodyGridPos ecs.Type = 1 << iota
	bodyRune
	bodyRuneAttr
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

func (defn *bodyDefinition) apply(s *shard, e ecs.Entity) {
	i, _ := s.bodIndex.GetID(e.ID)
	s.bod[i].Init(defn)
}

type body struct {
	setup bool

	bi braille.Bitmap

	ecs.Scope                // direct indexing into:
	gridPos   []image.Point  // always defined
	runes     []rune         // defined for bodyRune
	runeAttr  []ansi.SGRAttr // defined for bodyRuneAttr
}

func (bod *body) Init(defn *bodyDefinition) {
	if !bod.setup {
		if defn == nil {
			defn = &defaultBodyDef
		}
		bod.setup = true
		bod.Watch(bodyGridPos, 0, ecs.EntityCreatedFunc(bod.alloc))
		bod.Watch(bodyRune, 0, ecs.EntityDestroyedFunc(bod.clearPos))
		bod.Watch(bodyRune, 0, ecs.EntityDestroyedFunc(bod.clearRune))
		bod.Watch(bodyRuneAttr, 0, ecs.EntityDestroyedFunc(bod.clearRuneAttr))
	}
	if defn == nil {
		return
	}

	bod.Clear()

	for i, partDef := range defn.parts {
		bod.Create(bodyGridPos)
		bod.gridPos[i] = partDef.Point
	}

	bod.bi = *defn.Bitmap
	bod.bi.Bit = append(bod.bi.Bit[:0], bod.bi.Bit...)
}

func (bod *body) alloc(e ecs.Entity, _ ecs.Type) {
	i := int(e.Seq())
	for i >= len(bod.gridPos) {
		if i < cap(bod.gridPos) {
			bod.gridPos = bod.gridPos[:i+1]
		} else {
			bod.gridPos = append(bod.gridPos, image.ZP)
		}
	}
	for i >= len(bod.runes) {
		if i < cap(bod.runes) {
			bod.runes = bod.runes[:i+1]
		} else {
			bod.runes = append(bod.runes, 0)
		}
	}
	for i >= len(bod.runeAttr) {
		if i < cap(bod.runeAttr) {
			bod.runeAttr = bod.runeAttr[:i+1]
		} else {
			bod.runeAttr = append(bod.runeAttr, 0)
		}
	}
}

func (bod *body) clearPos(e ecs.Entity, _ ecs.Type)      { bod.gridPos[e.Seq()] = image.ZP }
func (bod *body) clearRune(e ecs.Entity, _ ecs.Type)     { bod.runes[e.Seq()] = 0 }
func (bod *body) clearRuneAttr(e ecs.Entity, _ ecs.Type) { bod.runeAttr[e.Seq()] = 0 }

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
