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
	hilbert    int
	origin     image.Point
	drawnRooms map[int]struct{}
	cursor     *borkgen.Room

	// scratch space
	builder
}

func (gen *roomGen) Init(s *shard, t ecs.Type) {
	gen.drawnRooms = make(map[int]struct{})
	gen.shard = s
}

func (gen *roomGen) logf(mess string, args ...interface{}) {
	log.Printf(mess, args...)
}

func (gen *roomGen) run(within image.Rectangle) bool {
	if gen.cursor == nil {
		gen.cursor = borkgen.DescribeRoom(image.ZP)
	}
	gen.cursor = borkgen.Draw(gen, gen, gen.cursor, within)
	return false
}

func (gen *roomGen) SetRoomDrawn(i int) {
	gen.logf("room drawn %d\n", i)
	gen.drawnRooms[i] = struct{}{}
}

func (gen *roomGen) IsRoomDrawn(i int) bool {
	_, ok := gen.drawnRooms[i]
	return ok
}

func (gen *roomGen) FillAisle(rect image.Rectangle) {
	gen.builder.spec = gen.Aisle
	gen.builder.fill(rect)
}

func (gen *roomGen) FillWall(rect image.Rectangle) {
	gen.builder.spec = gen.Wall
	gen.builder.fill(rect)
}

func (gen *roomGen) FillFloor(rect image.Rectangle) {
	gen.builder.spec = gen.Floor
	gen.builder.fill(rect)
}
