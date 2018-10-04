package main

import (
	"image"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/braille"
)

type body struct {
	bi        braille.Bitmap
	slotPos   []image.Point
	slotRunes []rune
	slotAttrs []ansi.SGRAttr
}

func (bod *body) Init() {
	if bod.bi.Bit == nil {
		bod.bi = *bodyBits
		bod.bi.Bit = append([]bool(nil), bod.bi.Bit...)
		bod.slotPos = bodySlotPositions[:]
		bod.slotRunes = make([]rune, len(bod.slotPos))
		bod.slotAttrs = make([]ansi.SGRAttr, len(bod.slotPos))
	}
}

func (bod *body) Size() image.Point { return bod.bi.RuneSize() }

func (bod *body) RenderInto(g *anansi.Grid, at image.Point, a ansi.SGRAttr) {
	bod.bi.CopyInto(g, at, true, a)
	for i, r := range bod.slotRunes {
		if r != 0 {
			cell := g.Cell(at.Add(bod.slotPos[i]))
			cell.SetRune(r)
			if a := bod.slotAttrs[i]; a != 0 {
				cell.SetAttr(a)
			}
		}
	}
}

func (bod *body) Slot(i int) (rune, ansi.SGRAttr) {
	return bod.slotRunes[i], bod.slotAttrs[i]
}

func (bod *body) SetSlot(i int, r rune, a ansi.SGRAttr) {
	bod.slotRunes[i] = r
	bod.slotAttrs[i] = a
}

const (
	bodyLeftHeadSlot  = iota // h
	bodyRightHeadSlot        // H
	bodyLeftSlot             // L
	bodyRightSlot            // R
	bodyLeftTailSlot         // t
	bodyRightTailSlot        // T

	bodyNumSlots
)

var (
	// 6x3 runes => 12x12 bits
	bodyBits = braille.NewBitmapString('#',
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
		"##  ttTT  ##")

	bodyPosLeftHand   = image.Pt(0, 0)
	bodyPosLeftArm    = image.Pt(1, 0)
	bodyPosLeftHead   = image.Pt(2, 0) // h
	bodyPosRightHead  = image.Pt(3, 0) // H
	bodyPosRightArm   = image.Pt(4, 0)
	bodyPosRightHand  = image.Pt(5, 0)
	bodyPosLeftSide   = image.Pt(0, 1) // L
	bodyPosLeftHip    = image.Pt(1, 1)
	bodyPosLeftTorso  = image.Pt(2, 1)
	bodyPosRightTorso = image.Pt(3, 1)
	bodyPosRightHip   = image.Pt(4, 1)
	bodyPosRightSide  = image.Pt(5, 1) // R
	bodyPosLeftFoot   = image.Pt(0, 2)
	bodyPosLeftLeg    = image.Pt(1, 2)
	bodyPosLeftTail   = image.Pt(2, 2) // t
	bodyPosRightTail  = image.Pt(3, 2) // T
	bodyPosRightLeg   = image.Pt(4, 2)
	bodyPosRightFoot  = image.Pt(5, 2)
)

var bodySlotPositions = [bodyNumSlots]image.Point{
	bodyPosLeftHead,  // h
	bodyPosRightHead, // H
	bodyPosLeftSide,  // L
	bodyPosRightSide, // R
	bodyPosLeftTail,  // t
	bodyPosRightTail, // T
}
