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
	"cc##  hhHH  ##CC",
	"cc### hhHH ###CC",
	"cc  ##hhHH##  CC",
	"cc  ##hhHH##  CC",
	"  LL ###### RR  ",
	"  LL  #  #  RR  ",
	"  LL  #  #  RR  ",
	"  LL ###### RR  ",
	"    ##ttTT##    ",
	"    ##ttTT##    ",
	"  ### ttTT ###  ",
	"  ##  ttTT  ##  ",
), []bodyPartDef{
	{Point: image.Pt(0, 0), name: "left hand slot"}, // c
	{Point: image.Pt(1, 0), name: "left hand"},
	{Point: image.Pt(2, 0), name: "left arm"},
	{Point: image.Pt(3, 0), name: "left head slot"}, // h
	{Point: image.Pt(4, 0), name: "right head slot"}, // H
	{Point: image.Pt(5, 0), name: "right arm"},
	{Point: image.Pt(6, 0), name: "right hand"},
	{Point: image.Pt(7, 0), name: "right hand slot"}, // C
	{Point: image.Pt(1, 1), name: "left side slot"}, // L
	{Point: image.Pt(2, 1), name: "left side"},
	{Point: image.Pt(3, 1), name: "left torso"},
	{Point: image.Pt(4, 1), name: "right torso"},
	{Point: image.Pt(5, 1), name: "right side"},
	{Point: image.Pt(6, 1), name: "right side slot"}, // R
	{Point: image.Pt(1, 2), name: "left foot"},
	{Point: image.Pt(2, 2), name: "left leg"},
	{Point: image.Pt(3, 2), name: "left tail slot"}, // t
	{Point: image.Pt(4, 2), name: "right tail slot"}, // T
	{Point: image.Pt(5, 2), name: "right leg"},
	{Point: image.Pt(6, 2), name: "right foot"},
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
	name string
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
	name      []string       // always defined
	runes     []rune         // defined for bodyRune
	runeAttr  []ansi.SGRAttr // defined for bodyRuneAttr

	parts map[string]ecs.ID
}

func (bod *body) Init(defn *bodyDefinition) {
	if !bod.setup {
		if defn == nil {
			defn = &defaultBodyDef
		}
		bod.setup = true
		bod.Watch(bodyGridPos, 0, ecs.EntityCreatedFunc(bod.alloc))
		bod.Watch(bodyGridPos, 0, ecs.EntityDestroyedFunc(bod.clearPart))
		bod.Watch(bodyRune, 0, ecs.EntityDestroyedFunc(bod.clearPos))
		bod.Watch(bodyRune, 0, ecs.EntityDestroyedFunc(bod.clearRune))
		bod.Watch(bodyRuneAttr, 0, ecs.EntityDestroyedFunc(bod.clearRuneAttr))
	}
	if defn == nil {
		return
	}

	bod.Clear()

	if bod.parts == nil {
		bod.parts = make(map[string]ecs.ID, len(defn.parts))
	} else {
		for name := range bod.parts {
			delete(bod.parts, name)
		}
	}
	for i, partDef := range defn.parts {
		part := bod.Create(bodyGridPos)
		bod.gridPos[i] = partDef.Point
		bod.name[i] = partDef.name
		if partDef.name != "" {
			bod.parts[partDef.name] = part.ID
		}
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
	for i >= len(bod.name) {
		if i < cap(bod.name) {
			bod.name = bod.name[:i+1]
		} else {
			bod.name = append(bod.name, "")
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

func (bod *body) clearPart(e ecs.Entity, _ ecs.Type) {
	i := e.Seq()
	name := bod.name[i]
	bod.name[i] = ""
	delete(bod.parts, name)
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
