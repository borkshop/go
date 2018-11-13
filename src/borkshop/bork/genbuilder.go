package main

import (
	"image"

	"borkshop/ecs"
)

type builder struct {
	shard *shard // FIXME rename
	pos   image.Point
	ids   []ecs.ID

	spec entitySpec
}

func (bld *builder) reset() {
	bld.ids = bld.ids[:0]
}

func (bld *builder) moveTo(pos image.Point) {
	bld.pos = pos
}

func (bld *builder) rectangle(box image.Rectangle) {
	bld.moveTo(box.Min)
	bld.lineTo(image.Pt(0, 1), box.Dy()-1)
	bld.lineTo(image.Pt(1, 0), box.Dx()-1)
	bld.lineTo(image.Pt(0, -1), box.Dy()-1)
	bld.lineTo(image.Pt(-1, 0), box.Dx()-1)
}

func (bld *builder) point(p image.Point) ecs.Entity {
	bld.pos = p
	return bld.create()
}

func (bld *builder) fill(r image.Rectangle) {
	for bld.moveTo(r.Min); bld.pos.Y < r.Max.Y; bld.pos.Y++ {
		for bld.pos.X = r.Min.X; bld.pos.X < r.Max.X; bld.pos.X++ {
			bld.create()
		}
	}
}

func (bld *builder) lineTo(p image.Point, n int) {
	for i := 0; i < n; i++ {
		bld.create()
		bld.pos = bld.pos.Add(p)
	}
}

func (bld *builder) create() ecs.Entity {
	ent := bld.spec.create(bld.shard, bld.pos)
	bld.ids = append(bld.ids, ent.ID)
	return ent
}
