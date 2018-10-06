package main

import (
	"fmt"
	"image"

	"børk.com/ecs"
	"børk.com/quadindex"
)

type position struct {
	// An entity may be positioned within some space; directly indexed, since
	// most entities have a position, and since each space has to pay indirect
	// indexing costs locally anyhow.
	posSpace []int

	// An entity may have a space attached to it; indirectly indexed, since
	// most entities don't have an associated space.
	spaceIndex ecs.ArrayIndex

	// Space definitions themselves; space 0 is special: associated with no
	// entity, and the default space for any positioned entity. Since most
	// space-associated entities SHOULD themselves be positioned in some other
	// space, this will mostly end up being a tree. NOTE however that this is
	// not a requirement, so it's possible to have forest, with spatial roots
	// other than 0.
	space []space
	// TODO track roots?
}

type space struct {
	bounds image.Rectangle
	ecs.ArrayIndex
	pt []image.Point
	qi quadindex.Index
}

type positioned struct {
	spc *space
	pi  int
}

func (pos *position) Init(scope *ecs.Scope, spaceType, posType ecs.Type) {
	pos.spaceIndex.Init(scope)
	scope.Watch(spaceType, 0, ecs.Watchers{
		&pos.spaceIndex,
		ecs.EntityWatcher(pos.spaceCreated, pos.spaceDestroyed),
	})
	scope.Watch(posType, 0, ecs.EntityWatcher(pos.posCreated, pos.posDestroyed))
	pos.space = make([]space, 0, 1024)
	pos.spaceIndex.Insert(ecs.ZE)
	pos.spaceCreated(ecs.ZE, 0)
}

func (pos *position) spaceCreated(e ecs.Entity, _ ecs.Type) {
	si, _ := pos.spaceIndex.GetID(e.ID)
	for si >= len(pos.space) {
		if si < cap(pos.space) {
			pos.space = pos.space[:si+1]
		} else {
			pos.space = append(pos.space, space{})
		}
	}
	pos.space[si].init(pos.spaceIndex.Scope)
}

func (pos *position) spaceDestroyed(e ecs.Entity, _ ecs.Type) {
	si, _ := pos.spaceIndex.GetID(e.ID)
	pos.space[si].ArrayIndex.Reset()
}

func (spc *space) init(scope *ecs.Scope) {
	spc.ArrayIndex.Init(scope)
	spc.bounds = image.ZR
	spc.pt = spc.pt[:0]
	spc.qi.Reset()
}

func (pos *position) posCreated(e ecs.Entity, t ecs.Type) {
	i := int(e.Seq())
	for i >= len(pos.posSpace) {
		if i < cap(pos.posSpace) {
			pos.posSpace = pos.posSpace[:i+1]
		} else {
			pos.posSpace = append(pos.posSpace, 0)
		}
	}
	pos.posSpace[i] = 0
	pos.space[0].posCreated(e, t)
}

func (pos *position) posDestroyed(e ecs.Entity, t ecs.Type) {
	i := e.Seq()
	si := pos.posSpace[i]
	pos.space[si].posDestroyed(e, t)
	pos.posSpace[i] = 0
}

func (pos *position) Get(e ecs.Entity) positioned {
	si := pos.posSpace[e.Seq()]
	return pos.space[si].Get(e)
}

func (pos *position) GetID(id ecs.ID) positioned {
	si := pos.posSpace[id.Seq()]
	return pos.space[si].GetID(id)
}

// Space returns any space associated with the entity; its "inner" space
// (inventory, body, a compound cell, etc).
func (pos *position) Space(e ecs.Entity) *space {
	si, def := pos.spaceIndex.Get(e)
	if def {
		return &pos.space[si]
	}
	return nil
}

// Spatial returns the spatial entity that owns the given positioned entity.
// Returns the zero entity if the given entity isn't positioned or if it's in the root space.
func (pos *position) Spatial(e ecs.Entity) ecs.Entity {
	si := pos.posSpace[e.Seq()]
	return pos.spaceIndex.Entity(si)
}

// SetSpatial sets the spatial entity that owns the given positioned entity.
// The entitie's position is initialized to the space's minimum bounds.
func (pos *position) SetSpatial(e, spatial ecs.Entity) {
	if spc := pos.Space(spatial); spc != nil {
		osi := pos.posSpace[e.Seq()]
		pos.space[osi].posDestroyed(e, 0)
		spc.posCreated(e, 0)
		spc.GetID(e.ID).SetPoint(spc.bounds.Min)
	}
}

func (spc *space) Bounds(e ecs.Entity) (image.Rectangle, bool) { return spc.bounds, true }
func (spc *space) SetBounds(e ecs.Entity, bounds image.Rectangle) { spc.bounds = bounds }

func (pos *position) At(p image.Point) (pq positionQuery)         { return pos.space[0].At(p) }
func (pos *position) Within(r image.Rectangle) (pq positionQuery) { return pos.space[0].Within(r) }

func (spc *space) posCreated(e ecs.Entity, t ecs.Type) {
	pi := spc.ArrayIndex.Insert(e)
	for pi >= len(spc.pt) {
		if pi < cap(spc.pt) {
			spc.pt = spc.pt[:pi+1]
		} else {
			spc.pt = append(spc.pt, image.ZP)
		}
	}
	spc.pt[pi] = image.ZP
	spc.qi.Update(pi, image.ZP)
}

func (spc *space) posDestroyed(e ecs.Entity, t ecs.Type) {
	if pi, def := spc.ArrayIndex.Get(e); def {
		spc.pt[pi] = image.ZP // FIXME invalid
	}
	spc.ArrayIndex.EntityDestroyed(e, t)
}

func (spc *space) Get(e ecs.Entity) positioned {
	if pi, def := spc.ArrayIndex.Get(e); def {
		return positioned{spc, pi}
	}
	return positioned{}
}

func (spc *space) GetID(id ecs.ID) positioned {
	if pi, def := spc.ArrayIndex.GetID(id); def {
		return positioned{spc, pi}
	}
	return positioned{}
}

func (spc *space) At(p image.Point) (pq positionQuery) {
	pq.spc = spc
	pq.Cursor = spc.qi.At(p)
	return pq
}

func (spc *space) Within(r image.Rectangle) (pq positionQuery) {
	pq.spc = spc
	pq.Cursor = spc.qi.Within(r)
	return pq
}

type positionQuery struct {
	spc *space
	quadindex.Cursor
}

func (pq *positionQuery) handle() positioned {
	if pi := pq.I(); pi >= 0 {
		return positioned{pq.spc, pi}
	}
	return positioned{}
}

func (posd positioned) zero() bool { return posd.spc == nil }

func (posd positioned) Point() image.Point {
	if posd.spc == nil {
		return image.ZP
	}
	return posd.spc.pt[posd.pi]
}

func (posd positioned) SetPoint(p image.Point) {
	if posd.spc != nil {
		posd.spc.pt[posd.pi] = p
		posd.spc.qi.Update(posd.pi, p)
	}
}

func (posd positioned) Entity() ecs.Entity { return posd.spc.Entity(posd.pi) }
func (posd positioned) ID() ecs.ID         { return posd.spc.ID(posd.pi) }

func (posd positioned) String() string {
	if posd.spc == nil {
		return fmt.Sprintf("no-position")
	}
	return fmt.Sprintf("pt: %v", posd.spc.pt[posd.pi])
}
