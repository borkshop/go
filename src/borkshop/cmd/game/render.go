package main

import (
	"fmt"
	"image"
	"sort"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"

	"borkshop/ecs"
)

type render struct {
	pos *position
	ecs.ArrayIndex
	cell []cell
	zord renderZord
}

type renderZord struct {
	z  []int
	ri []int
	pi []int
}

func (rz renderZord) Len() int           { return len(rz.ri) }
func (rz renderZord) Less(i, j int) bool { return rz.z[rz.ri[i]] > rz.z[rz.ri[j]] }
func (rz renderZord) Swap(i, j int) {
	rz.ri[i], rz.ri[j] = rz.ri[j], rz.ri[i]
	rz.pi[i], rz.pi[j] = rz.pi[j], rz.pi[i]
}

type cell struct {
	r, r2 rune
	a     ansi.SGRAttr
}

func (ren *render) Init(scope *ecs.Scope, t ecs.Type, pos *position) {
	ren.pos = pos
	ren.ArrayIndex.Init(scope)
	scope.Watch(t, 0, ren)
}

func (ren *render) drawRegionInto(view image.Rectangle, grid *anansi.Grid) {
	ren.rezort(ren.pos.Within(view))
	ren.drawZordOff(view.Min, grid)
}

func (ren *render) drawZordOff(off image.Point, grid *anansi.Grid) {
	for ii := range ren.zord.ri {
		ri := ren.zord.ri[ii]
		pi := ren.zord.pi[ii]
		posd := positioned{ren.pos, pi}
		pt := ansi.PtFromImage(posd.Point().Sub(off))
		pt.X *= 2

		i1, ok := grid.CellOffset(pt)
		if !ok {
			continue
		}
		i2, ok := grid.CellOffset(pt.Add(image.Pt(1, 0)))
		if !ok {
			continue
		}

		c := ren.cell[ri]
		if grid.Rune[i1] == 0 {
			grid.Rune[i1], grid.Rune[i2] = c.r, c.r2
			grid.Attr[i1], grid.Attr[i2] = c.a, c.a
		} else {
			a := grid.Attr[i1]
			if _, bgSet := a.BG(); !bgSet {
				if color, haveBG := c.a.BG(); haveBG {
					a |= color.BG()
					grid.Attr[i1], grid.Attr[i2] = a, a
				}
			}
		}
	}
}

func (ren *render) rezort(pq positionQuery) {
	if ren.zord.ri != nil {
		ren.zord.ri = ren.zord.ri[:0]
		ren.zord.pi = ren.zord.pi[:0]
	}
	for pq.Next() {
		if h := pq.handle(); !h.zero() {
			if i, def := ren.ArrayIndex.GetID(h.ID()); def {
				ren.zord.ri = append(ren.zord.ri, i)
				ren.zord.pi = append(ren.zord.pi, h.pi)
			}
		}
	}
	sort.Stable(ren.zord)
}

func (ren *render) EntityCreated(ent ecs.Entity, _ ecs.Type) {
	i := ren.ArrayIndex.Insert(ent)

	for i >= len(ren.cell) {
		if i < cap(ren.cell) {
			ren.cell = ren.cell[:i+1]
		} else {
			ren.cell = append(ren.cell, cell{})
		}
	}
	ren.cell[i] = cell{}

	for i >= len(ren.zord.z) {
		if i < cap(ren.zord.z) {
			ren.zord.z = ren.zord.z[:i+1]
		} else {
			ren.zord.z = append(ren.zord.z, 0)
		}
	}
	ren.zord.z[i] = 0
}

type renderable struct {
	positioned
	ren *render
	ri  int
}

func (ren *render) Get(ent ecs.Entity) renderable {
	if ri, def := ren.ArrayIndex.Get(ent); def {
		return renderable{ren.pos.GetID(ent.ID), ren, ri}
	}
	return renderable{}
}

func (ren *render) GetID(id ecs.ID) renderable {
	if ri, def := ren.ArrayIndex.GetID(id); def {
		return renderable{ren.pos.GetID(id), ren, ri}
	}
	return renderable{}
}

func (rend renderable) Z() int {
	if rend.ren == nil {
		return 0
	}
	return rend.ren.zord.z[rend.ri]
}
func (rend renderable) SetZ(z int) {
	if rend.ren != nil {
		rend.ren.zord.z[rend.ri] = z
	}
}

func (rend renderable) Cell() (r, r2 rune, _ ansi.SGRAttr) {
	if rend.ren == nil {
		return 0, 0, 0
	}
	cell := rend.ren.cell[rend.ri]
	return cell.r, cell.r2, cell.a
}
func (rend renderable) SetCell(r, r2 rune, a ansi.SGRAttr) {
	if rend.ren != nil {
		rend.ren.cell[rend.ri] = cell{r, r2, a}
	}
}

func (rend renderable) Entity() ecs.Entity {
	return rend.ren.Scope.Entity(rend.ren.ID(rend.ri))
}

func (rend renderable) Apply(st renderStyle) {
	rend.SetZ(st.z)
	rend.SetCell(st.r, st.r2, st.a)
}

func (rend renderable) String() string {
	if rend.ren == nil {
		return fmt.Sprintf("no-render")
	}
	a := rend.ren.cell[rend.ri].a
	fg, _ := a.FG()
	bg, _ := a.BG()
	fl := a.SansBG().SansFG()
	return fmt.Sprintf("z:%v rune:%q fg:%v bg:%v attr:%q",
		rend.ren.zord.z[rend.ri],
		rend.ren.cell[rend.ri].r,
		fg, bg,
		fl,
	)
}

func renStyle(z int, r, r2 rune, a ansi.SGRAttr) renderStyle {
	return renderStyle{z, r, r2, a}
}

type renderStyle struct {
	z     int
	r, r2 rune
	a     ansi.SGRAttr
}

func (st renderStyle) String() string {
	return fmt.Sprintf("z:%v rune:%q attr:%v", st.z, st.r, st.a)
}

func (st renderStyle) apply(s *shard, ent ecs.Entity) { s.ren.Get(ent).Apply(st) }
