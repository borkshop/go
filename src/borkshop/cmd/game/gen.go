package main

import (
	"image"
	"log"

	"borkshop/borkgen"
	"borkshop/ecs"
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
	hilbert   int
	origin    image.Point
	generated map[image.Point]struct{}
	cursor    *borkgen.Room

	// scratch space
	builder
}

func (gen *roomGen) Init(s *shard, t ecs.Type) {
	gen.generated = make(map[image.Point]struct{})
	gen.shard = s
}

func (gen *roomGen) logf(mess string, args ...interface{}) {
	if gen.Log {
		log.Printf(mess, args...)
	}
}

func (gen *roomGen) run(within image.Rectangle) bool {
	if gen.cursor == nil {
		gen.cursor = borkgen.DescribeRoom(image.ZP)
	}
	room := gen.cursor

	gen.logf("generating %v at %v\n", room.HilbertPt, room.Pt)

	// find the northwest corner of the visible region
	for room.Pt.X > within.Min.X {
		room = room.At(image.Pt(room.HilbertPt.X-1, room.HilbertPt.Y))
	}
	for room.Pt.Y > within.Min.Y {
		room = room.At(image.Pt(room.HilbertPt.X, room.HilbertPt.Y-1))
	}
	// draw rooms in boustrophedon
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
	gen.cursor = room

	return false
}

func (gen *roomGen) genRoom(room *borkgen.Room) {
	if _, ok := gen.generated[room.HilbertPt]; ok {
		return
	}
	gen.generated[room.HilbertPt] = struct{}{}

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
