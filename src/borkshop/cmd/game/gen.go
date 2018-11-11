package main

import (
	"image"
	"log"
	"math/rand"

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
	lastDrawnRoom *borkgen.Room
	drawnRooms    map[int]struct{}

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
	if gen.lastDrawnRoom == nil {
		spawn := image.Pt(rand.Intn(borkgen.Scale), rand.Intn(borkgen.Scale))
		gen.logf("player spawns at %v\n", spawn)
		gen.lastDrawnRoom = borkgen.DescribeRoom(spawn)
	}
	gen.lastDrawnRoom = borkgen.Draw(gen, gen, gen.lastDrawnRoom, within)
	return false
}

func (gen *roomGen) SetRoomDrawn(room *borkgen.Room) {
	gen.logf("room drawn %v\n", room.HilbertPt)
	gen.drawnRooms[room.HilbertNum] = struct{}{}
}

func (gen *roomGen) IsRoomDrawn(room *borkgen.Room) bool {
	_, ok := gen.drawnRooms[room.HilbertNum]
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
