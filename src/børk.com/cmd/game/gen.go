package main

import (
	"image"
	"log"
	"math/rand"

	"børk.com/borkgen"
	"børk.com/ecs"
)

type roomGenConfig struct {
	Log bool

	Floor  entitySpec
	Aisle  entitySpec
	Wall   entitySpec
	Door   entitySpec
	Player entitySpec

	PlaceAttempts int

	RoomSize    image.Rectangle
	MinHallSize int
	MaxHallSize int
	ExitDensity int
}

type roomGen struct {
	roomGenConfig

	// generation state
	first   bool
	hilbert int
	origin  image.Point
	done    bool

	// generation state
	ecs.ArrayIndex
	data  []genRoom
	rooms *rooms
	tick  int

	// scratch space
	builder
	points []image.Point
	ids    []ecs.ID
}

func (gen *roomGen) Init(s *shard, t ecs.Type) {
	gen.shard = s
	gen.rooms = &s.rooms
	gen.ArrayIndex.Init(&s.Scope)
	s.Scope.Watch(t, 0, gen)
}

func (gen *roomGen) logf(mess string, args ...interface{}) {
	if gen.Log {
		log.Printf(mess, args...)
	}
}

func (gen *roomGen) EntityCreated(ent ecs.Entity, _ ecs.Type) {
	i := gen.ArrayIndex.Insert(ent)
	for i >= len(gen.data) {
		if i < cap(gen.data) {
			gen.data = gen.data[:i+1]
		} else {
			gen.data = append(gen.data, genRoom{})
		}
	}
	gen.data[i] = genRoom{}
}

func (gen *roomGen) Get(ent ecs.Entity) genRoomHandle {
	if i, def := gen.ArrayIndex.Get(ent); def {
		return gen.load(i)
	}
	return genRoomHandle{}
}

func (gen *roomGen) GetID(id ecs.ID) genRoomHandle {
	i, def := gen.ArrayIndex.GetID(id)
	if def {
		return gen.load(i)
	}
	return genRoomHandle{}
}

func (gen *roomGen) expandSimRegion(r image.Rectangle) image.Rectangle {
	// TODO evaluate transitive vs first-level-only expansion
	maxImpact := gen.RoomSize.Max.Add(image.Pt(gen.MaxHallSize, gen.MaxHallSize))
	r.Min = r.Min.Sub(maxImpact.Mul(gen.PlaceAttempts / 2))
	r.Max = r.Max.Add(maxImpact.Mul(gen.PlaceAttempts / 2))
	res := r
	for i := 0; i < len(gen.data); i++ {
		if gen.ArrayIndex.ID(i) != 0 {
			room := gen.load(i)
			impact := image.Rectangle{
				room.r.Min.Sub(maxImpact),
				room.r.Max.Add(maxImpact)}
			if impact.Overlaps(r) {
				res = expandTo(res, impact.Min)
				res = expandTo(res, impact.Max)
			}
		}
	}
	return res
}

func (gen *roomGen) run(within image.Rectangle) bool {
	if !gen.done {
		room := borkgen.DescribeRoom(image.ZP)

		// find the northwest corner of the visible region
		for room.Pt.X > within.Min.X {
			room = room.At(image.Pt(room.HilbertPt.X-1, room.HilbertPt.Y))
		}
		for room.Pt.Y > within.Min.Y {
			room = room.At(image.Pt(room.HilbertPt.X, room.HilbertPt.Y-1))
		}
		// draw rooms in boustrophedron
		for room.Pt.Y < within.Max.Y {
			for room.Pt.X < within.Max.X {
				gen.genRoom(room)
				room = room.At(image.Pt(room.HilbertPt.X+1, room.HilbertPt.Y))
			}
			room = room.At(image.Pt(room.HilbertPt.X, room.HilbertPt.Y+1))
			for room.Pt.X > within.Min.X {
				gen.genRoom(room)
				room = room.At(image.Pt(room.HilbertPt.X-1, room.HilbertPt.Y))
			}
			room = room.At(image.Pt(room.HilbertPt.X, room.HilbertPt.Y+1))
		}
		gen.done = true
	}
	return false
}

func (gen *roomGen) genRoom(room *borkgen.Room) {
	// center
	gen.builder.spec = gen.Aisle
	gen.builder.moveTo(room.Pt)
	gen.builder.create()

	// corners
	// gen.wallOrFloor(room.NorthWall || room.WestWall)
	gen.builder.spec = gen.Wall
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, -room.NorthMargin-1)).
		Add(room.Pt))
	// gen.wallOrFloor(room.NorthWall || room.EastWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, -room.NorthMargin-1)).
		Add(room.Pt))
	// gen.wallOrFloor(room.SouthWall || room.WestWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, room.SouthMargin+1)).
		Add(room.Pt))
	// gen.wallOrFloor(room.SouthWall || room.EastWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, room.SouthMargin+1)).
		Add(room.Pt))

	// aisles
	gen.floorOrAisle(!room.NorthWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(0, -room.NorthMargin)).
		Add(room.Pt))
	gen.floorOrAisle(!room.SouthWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(0, 1)).
		Add(room.Pt))
	gen.floorOrAisle(!room.WestWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, 0)).
		Add(room.Pt))
	gen.floorOrAisle(!room.EastWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, 0)).
		Add(room.Pt))

	// floor quadrants
	gen.builder.spec = gen.Floor
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.WestMargin, room.NorthMargin)}.
		Add(image.Pt(-room.WestMargin, -room.NorthMargin)).
		Add(room.Pt))
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.WestMargin, room.SouthMargin)}.
		Add(image.Pt(-room.WestMargin, 1)).
		Add(room.Pt))
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.EastMargin, room.NorthMargin)}.
		Add(image.Pt(1, -room.NorthMargin)).
		Add(room.Pt))
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.EastMargin, room.SouthMargin)}.
		Add(image.Pt(1, 1)).
		Add(room.Pt))

	// north wall segments
	gen.wallOrFloor(room.NorthWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, -room.NorthMargin-1)).
		Add(room.Pt))
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, -room.NorthMargin-1)).
		Add(room.Pt))
	gen.maybeDoor(room.NorthWall, room.NorthDoor)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(0, -room.NorthMargin-1)).
		Add(room.Pt))

	// south wall segments
	gen.wallOrFloor(room.SouthWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, room.SouthMargin+1)).
		Add(room.Pt))
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, room.SouthMargin+1)).
		Add(room.Pt))
	gen.maybeDoor(room.SouthWall, room.SouthDoor)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(0, room.SouthMargin+1)).
		Add(room.Pt))

	// west wall segments
	gen.wallOrFloor(room.WestWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(-room.WestMargin-1, -room.NorthMargin)).
		Add(room.Pt))
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(-room.WestMargin-1, 1)).
		Add(room.Pt))
	gen.maybeDoor(room.WestWall, room.WestDoor)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, 0)).
		Add(room.Pt))

	// east wall segments
	gen.wallOrFloor(room.EastWall)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(room.EastMargin+1, -room.NorthMargin)).
		Add(room.Pt))
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(room.EastMargin+1, 1)).
		Add(room.Pt))
	gen.maybeDoor(room.EastWall, room.EastDoor)
	gen.builder.fill(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, 0)).
		Add(room.Pt))
}

func (gen *roomGen) wallOrFloor(wall bool) {
	if wall {
		gen.builder.spec = gen.Wall
	} else {
		gen.builder.spec = gen.Floor
	}
}

func (gen *roomGen) wallOrAisle(aisle bool) {
	if aisle {
		gen.builder.spec = gen.Aisle
	} else {
		gen.builder.spec = gen.Wall
	}
}

func (gen *roomGen) floorOrAisle(aisle bool) {
	if aisle {
		gen.builder.spec = gen.Aisle
	} else {
		gen.builder.spec = gen.Floor
	}
}

func (gen *roomGen) maybeDoor(wall bool, door bool) {
	if wall {
		if door {
			gen.builder.spec = gen.Floor
		} else {
			gen.builder.spec = gen.Wall
		}
	} else {
		gen.builder.spec = gen.Aisle
	}
}

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

func sharesPointComponent(pt image.Point, pts []image.Point) bool {
	for _, pti := range pts {
		if pti.X == pt.X || pti.Y == pt.Y {
			return true
		}
	}
	return false
}

func shuffleIDs(ids []ecs.ID) {
	for i := 1; i < len(ids); i++ {
		if j := rand.Intn(i + 1); j != i {
			ids[i], ids[j] = ids[j], ids[i]
		}
	}
}

func isCorner(p image.Point, r image.Rectangle) bool {
	return (p.X == r.Min.X && p.Y == r.Min.Y) ||
		(p.X == r.Min.X && p.Y == r.Max.Y-1) ||
		(p.X == r.Max.X-1 && p.Y == r.Min.Y) ||
		(p.X == r.Max.X-1 && p.Y == r.Max.Y-1)
}

func orthNormal(p image.Point) image.Point {
	if p.X == 0 {
		return image.Pt(1, 0)
	}
	if p.Y == 0 {
		return image.Pt(0, 1)
	}
	return image.ZP
}

func expandTo(r image.Rectangle, p image.Point) image.Rectangle {
	if p.X < r.Min.X {
		r.Min.X = p.X
	} else if p.X >= r.Max.X {
		r.Max.X = p.X + 1
	}
	if p.Y < r.Min.Y {
		r.Min.Y = p.Y
	} else if p.Y >= r.Max.Y {
		r.Max.Y = p.Y + 1
	}
	return r
}
