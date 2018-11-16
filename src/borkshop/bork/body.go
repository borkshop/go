package main

import (
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"

	"borkshop/ecs"
)

const (
	bodyGridPos ecs.Type = 1 << iota
	bodyRune
	bodyRuneAttr
	bodySlot
	bodyHand
)

var defaultBodyDef = bodyDef(anansi.NewBitmap(anansi.MustParseBitmap("##",
	"cccc####    hhhhHHHH    ####CCCC",
	"cccc######  hhhhHHHH  ######CCCC",
	"cccc    ####hhhhHHHH####    CCCC",
	"cccc    ####hhhhHHHH####    CCCC",
	"    LLLL  ############  RRRR    ",
	"    LLLL    ##    ##    RRRR    ",
	"    LLLL    ##    ##    RRRR    ",
	"    LLLL  ############  RRRR    ",
	"        ####ttttTTTT####        ",
	"        ####ttttTTTT####        ",
	"    ######  ttttTTTT  ######    ",
	"    ####    ttttTTTT    ####    ",
)), []bodyPartDef{
	{Point: image.Pt(0, 0), name: "left hand slot", t: bodySlot | bodyHand}, // c
	{Point: image.Pt(1, 0), name: "left hand"},
	{Point: image.Pt(2, 0), name: "left arm"},
	{Point: image.Pt(3, 0), name: "left head slot", t: bodySlot},  // h
	{Point: image.Pt(4, 0), name: "right head slot", t: bodySlot}, // H
	{Point: image.Pt(5, 0), name: "right arm"},
	{Point: image.Pt(6, 0), name: "right hand"},
	{Point: image.Pt(7, 0), name: "right hand slot", t: bodySlot | bodyHand}, // C
	{Point: image.Pt(1, 1), name: "left side slot", t: bodySlot},             // L
	{Point: image.Pt(2, 1), name: "left side"},
	{Point: image.Pt(3, 1), name: "left torso"},
	{Point: image.Pt(4, 1), name: "right torso"},
	{Point: image.Pt(5, 1), name: "right side"},
	{Point: image.Pt(6, 1), name: "right side slot", t: bodySlot}, // R
	{Point: image.Pt(1, 2), name: "left foot"},
	{Point: image.Pt(2, 2), name: "left leg"},
	{Point: image.Pt(3, 2), name: "left tail slot", t: bodySlot},  // t
	{Point: image.Pt(4, 2), name: "right tail slot", t: bodySlot}, // T
	{Point: image.Pt(5, 2), name: "right leg"},
	{Point: image.Pt(6, 2), name: "right foot"},
})

func bodyDef(bi *anansi.Bitmap, parts []bodyPartDef) bodyDefinition {
	return bodyDefinition{bi, parts}
}

type bodyDefinition struct {
	*anansi.Bitmap
	parts []bodyPartDef
}

type bodyPartDef struct {
	image.Point
	name string
	t    ecs.Type
}

func (defn *bodyDefinition) apply(s *shard, e ecs.Entity) {
	i, _ := s.bodIndex.GetID(e.ID)
	s.bod[i].Init(defn)
}

type body struct {
	setup bool

	bi anansi.Bitmap

	ecs.Scope                // direct indexing into:
	gridPos   []image.Point  // always defined
	name      []string       // always defined
	runes     []rune         // defined for bodyRune
	runeAttr  []ansi.SGRAttr // defined for bodyRuneAttr

	parts map[string]ecs.ID
	slots ecs.ArrayIndex
	hands ecs.ArrayIndex
}

func (bod *body) Init(defn *bodyDefinition) {
	if !bod.setup {
		if defn == nil {
			defn = &defaultBodyDef
		}
		bod.setup = true
		bod.slots.Init(&bod.Scope)
		bod.hands.Init(&bod.Scope)
		bod.Watch(bodyGridPos, 0, ecs.EntityCreatedFunc(bod.alloc))
		bod.Watch(bodyGridPos, 0, ecs.EntityDestroyedFunc(bod.clearPart))
		bod.Watch(bodyRune, 0, ecs.EntityDestroyedFunc(bod.clearPos))
		bod.Watch(bodyRune, 0, ecs.EntityDestroyedFunc(bod.clearRune))
		bod.Watch(bodyRuneAttr, 0, ecs.EntityDestroyedFunc(bod.clearRuneAttr))
		bod.Watch(bodySlot, 0, &bod.slots)
		bod.Watch(bodyHand, 0, &bod.hands)
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
		if partDef.t != 0 {
			part.AddType(partDef.t)
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

func (bod *body) RenderInto(g anansi.Grid, at ansi.Point, ba ansi.SGRAttr) {
	style := anansi.Styles(
		anansi.TransparentAttrBG,
		anansi.StyleFunc(func(_ ansi.Point, _ rune, r rune, _, a ansi.SGRAttr) (rune, ansi.SGRAttr) {
			if c, set := a.BG(); set {
				a = a.SansBG() | darkenBy(c, 48).BG()
				// TODO evaluate fg contrast, maybe lighten
			}
			return r, a
		}),
	)

	anansi.DrawBitmap(g.SubAt(at), &bod.bi,
		anansi.TransparentBrailleRunes,
		anansi.AttrStyle(ba),
		style,
	)

	for i, r := range bod.runes {
		if r != 0 {
			pt := at.Add(bod.gridPos[i])
			if j, ok := g.CellOffset(pt); ok {
				pr, pa := g.Rune[j], g.Attr[j]
				a := bod.runeAttr[i]
				r, a = style.Style(pt, pr, r, pa, a)
				g.Rune[j], g.Attr[j] = r, a
			}
		}
	}
}

func darkenBy(c ansi.SGRColor, by uint8) ansi.SGRColor {
	// TODO better darken function
	r, g, b := c.RGB()
	if r >= by {
		r -= by
	} else {
		r = 0
	}
	if g >= by {
		g -= by
	} else {
		g = 0
	}
	if b >= by {
		b -= by
	} else {
		b = 0
	}
	return ansi.RGB(r, g, b)
}
