package main

import (
	"math/rand"

	"github.com/jcorbin/anansi/ansi"

	"deathroom/internal/ecs"
	"deathroom/internal/markov"
	"deathroom/internal/point"
)

const spawnColor = ansi.SGRCube53

// TODO convert to and better utilize 24-bit
var (
	aiColors = []ansi.SGRColor{
		ansi.SGRCube123, // XXX maybe off by -1
		ansi.SGRCube159,
		ansi.SGRCube195,
		ansi.SGRCube201,
		ansi.SGRCube207,
		ansi.SGRCube213,
	}
	soulColors = []ansi.SGRColor{
		ansi.SGRCube18, // XXX maybe off by -1
		ansi.SGRCube19,
		ansi.SGRCube20,
		ansi.SGRCube26,
		ansi.SGRCube32,
		ansi.SGRCube38,
	}
	itemColors = []ansi.SGRColor{
		ansi.SGRCube21, // XXX maybe off by -1
		ansi.SGRCube22,
		ansi.SGRCube28,
		ansi.SGRCube34,
		ansi.SGRCube40,
		ansi.SGRCube46,
	}

	wallColors = []ansi.SGRColor{
		ansi.SGRGray2,
		ansi.SGRGray3,
		ansi.SGRGray4,
		ansi.SGRGray5,
		ansi.SGRGray6,
		ansi.SGRGray7,
		ansi.SGRGray8,
	}
	floorColors = []ansi.SGRColor{
		ansi.SGRGray1,
		ansi.SGRGray2,
		ansi.SGRGray3,
	}

	wallTable  = newColorTable()
	floorTable = newColorTable()
)

func init() {
	wallTable.addLevelTransitions(wallColors, 12, 2, 2, 12, 2)
	floorTable.addLevelTransitions(floorColors, 24, 1, 30, 2, 1)
}

const (
	componentTableColor ecs.ComponentType = 1 << iota
)

type colorTable struct {
	ecs.Core
	*markov.Table
	color  []ansi.SGRColor
	lookup map[ansi.SGRColor]ecs.EntityID
}

func newColorTable() *colorTable {
	ct := &colorTable{
		// TODO: consider eliminating the padding for EntityID(0)
		color:  []ansi.SGRColor{0},
		lookup: make(map[ansi.SGRColor]ecs.EntityID, 1),
	}
	ct.Table = markov.NewTable(&ct.Core)
	ct.RegisterAllocator(componentTableColor, ct.allocTableColor)
	ct.RegisterDestroyer(componentTableColor, ct.destroyTableColor)
	return ct
}

func (ct *colorTable) allocTableColor(id ecs.EntityID, t ecs.ComponentType) {
	ct.color = append(ct.color, 0)
}

func (ct *colorTable) destroyTableColor(id ecs.EntityID, t ecs.ComponentType) {
	delete(ct.lookup, ct.color[id])
	ct.color[id] = 0
}

func (ct *colorTable) addLevelTransitions(
	colors []ansi.SGRColor,
	zeroOn, zeroUp int,
	oneDown, oneOn, oneUp int,
) {
	n := len(colors)
	c0 := colors[0]

	for i, c1 := range colors {
		if c1 == c0 {
			continue
		}

		ct.addTransition(c0, c0, (n-i)*zeroOn)
		ct.addTransition(c0, c1, (n-i)*zeroUp)

		ct.addTransition(c1, c0, (n-1)*oneDown)
		ct.addTransition(c1, c1, (n-1)*oneOn)

		for _, c2 := range colors {
			if c2 != c1 && c2 != c0 {
				ct.addTransition(c1, c2, (n-1)*oneUp)
			}
		}
	}
}

func (ct *colorTable) toEntity(a ansi.SGRColor) ecs.Entity {
	if id, def := ct.lookup[a]; def {
		return ct.Ref(id)
	}
	ent := ct.AddEntity(componentTableColor)
	id := ent.ID()
	ct.color[id] = a
	ct.lookup[a] = id
	return ent
}

func (ct *colorTable) toColor(ent ecs.Entity) (ansi.SGRColor, bool) {
	if !ent.Type().All(componentTableColor) {
		return 0, false
	}
	return ct.color[ent.ID()], true
}

func (ct *colorTable) addTransition(a, b ansi.SGRColor, w int) (ae, be ecs.Entity) {
	ae, be = ct.toEntity(a), ct.toEntity(b)
	ct.AddTransition(ae, be, w)
	return
}

func (ct *colorTable) genTile(
	rng *rand.Rand,
	box point.Box,
	f func(point.Point, ansi.SGRColor),
) {
	// TODO: better 2d generation
	last := floorTable.Ref(1)
	var pos point.Point
	for pos.Y = box.TopLeft.Y + 1; pos.Y < box.BottomRight.Y; pos.Y++ {
		first := last
		for pos.X = box.TopLeft.X + 1; pos.X < box.BottomRight.X; pos.X++ {
			c, _ := floorTable.toColor(last)
			f(pos, c)
			last = floorTable.ChooseNext(rng, last)
		}
		last = floorTable.ChooseNext(rng, first)
	}
}
