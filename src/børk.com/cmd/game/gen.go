package main

import (
	"image"
	"log"
	"math/rand"
	"time"

	"børk.com/ecs"
)

type roomGenConfig struct {
	Log bool

	Floor  entitySpec
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
	ecs.ArrayIndex
	data  []genRoom
	rooms *rooms
	tick  int

	// scratch space
	builder
	points []image.Point
	ids    []ecs.ID
}

type genRoom struct {
	done     bool
	depth    int
	tick     int
	maxExits int
	enter    image.Point
}

type genRoomHandle struct {
	gen *roomGen
	i   int

	r *image.Rectangle
	*genRoom
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
	const deadline = 3 * time.Millisecond
	t0 := time.Now()
	any := false
	for i := 0; i < len(gen.data); i++ {
		if gen.ArrayIndex.ID(i) != 0 {
			room := gen.load(i)
			if room.r.Overlaps(within) {
				any = true
				if room.tick < gen.tick {
					if !room.done {
						gen.createRoom(room)
						room.done = true
					} else if !gen.elaborateRoom(room) {
						gen.Entity(i).DeleteType(gameGen)
					}
					room.tick = gen.tick
				}
			}
		}
		t1 := time.Now()
		if t1.Sub(t0) > deadline {
			return true
		}
	}
	gen.tick++
	return any && len(gen.data) > 0
}

func (gen *roomGen) create(depth int, enter image.Point, r image.Rectangle) genRoomHandle {
	ent := gen.Scope.Create(gameRoom | gameGen)
	room := gen.GetID(ent.ID)
	if room.gen == nil {
		panic("missing new genRoom data")
	}

	room.depth = depth
	room.enter = enter
	*room.r = r
	gen.logf("gen %v %+v", ent, room)
	return room
}

func (gen *roomGen) createRoom(room genRoomHandle) {
	id := room.ID()
	gen.logf("room id:%v r:%v", id, room.r)
	room.maxExits = room.r.Dx() * room.r.Dy() / gen.ExitDensity

	// create room
	gen.reset()
	gen.spec = gen.Floor
	gen.fill(room.r.Inset(1))
	gen.rooms.parts.InsertMany(roomFloor, id, gen.builder.ids...)

	gen.reset()
	gen.spec = gen.Wall
	gen.rectangle(*room.r)
	gen.rooms.parts.InsertMany(roomWall, id, gen.builder.ids...)

	if room.enter != image.ZP {
		// entrance door
		for _, id := range gen.builder.ids {
			if posd := gen.shard.pos.GetID(id); posd.Point() == room.enter {
				gen.carveDoorway(room, posd.Entity())
				break
			}
		}
	}
}

func (gen *roomGen) createCorridor(pos, dir image.Point, n int) image.Point {
	orth := orthNormal(dir)
	gen.reset()
	for i := 0; i < n; i++ {
		pos = pos.Add(dir)
		gen.spec = gen.Floor
		gen.point(pos)
		gen.spec = gen.Wall
		gen.point(pos.Add(orth))
		gen.point(pos.Sub(orth))
	}
	return pos
}

func (gen *roomGen) elaborateRoom(room genRoomHandle) bool {
	gen.logf("elaborate %v", room.r)

	parts := gen.rooms.parts.LookupA(room.ID())
	if cap(gen.points) < len(parts.IDs) {
		gen.points = make([]image.Point, len(parts.IDs))
	}
	if cap(gen.ids) < len(parts.IDs) {
		gen.ids = make([]ecs.ID, len(parts.IDs))
	}
	gen.points = gen.points[:0]
	gen.ids = gen.ids[:0]
	numDoors := 0

	for i := 0; i < len(parts.IDs); i++ {
		part := parts.Entity(i)
		switch {
		case part.Type().HasAll(roomDoor):
			door := gen.rooms.parts.B(part)
			pt := gen.shard.pos.GetID(door.ID).Point()
			gen.points = append(gen.points, pt)
			numDoors++
		case part.Type().HasAll(roomWall):
			gen.ids = append(gen.ids, part.ID)
		}
	}

	// TODO more nuanced avoidance than "Shares a wall"... e.g.:
	// - elide anything that's within some distance of a door (e.g. 1 or 2
	//   cells away)
	// - weight the random choice so that further away walls are more likely to
	//   be chosen

	walls := gen.rooms.parts.Bs(ecs.Ents(parts.Scope, gen.ids), gen.ids)

	// prune corner walls and walls that share a component with any prior door
	var i int
	for j := 0; j < len(walls.IDs); j++ {
		pt := gen.shard.pos.GetID(walls.IDs[j]).Point()
		if isCorner(pt, *room.r) || sharesPointComponent(pt, gen.points) {
			continue
		}
		walls.IDs[j], walls.IDs[i] = walls.IDs[i], walls.IDs[j]
		i++
	}
	walls.IDs = walls.IDs[:i]

	// TODO hallways with turns

	shuffleIDs(walls.IDs)
	for _, id := range walls.IDs[:gen.PlaceAttempts] {
		// place hallway
		start := gen.shard.pos.GetID(id).Point()
		dir := room.wallNormal(start)
		end, n := gen.placeCorridor(start, dir)
		if n == 0 {
			continue
		}

		// place next room
		r := gen.placeNextRoom(end, dir)
		if r == image.ZR {
			continue
		}

		gen.logf("hallway dir:%v n:%v", dir, n)
		pos := start
		gen.carveDoorway(room, gen.shard.Entity(id))
		pos = gen.createCorridor(pos, dir, n)
		gen.create(room.depth+1, pos.Add(dir), r)
		numDoors++
		return numDoors < room.maxExits
	}
	return false
}

func (gen *roomGen) placeCorridor(pos, dir image.Point) (image.Point, int) {
	n := rand.Intn(gen.MaxHallSize-gen.MinHallSize) + gen.MinHallSize
	end := pos.Add(dir.Mul(n + 1)) // +1 to include landing
	r := image.Rectangle{pos, end.Add(dir)}.Canon()
	// TODO care about checking for wall cells too?
	if gen.anyWithin(r) {
		return pos, 0
	}
	return end, n
}

func (gen *roomGen) placeNextRoom(enter, dir image.Point) image.Rectangle {
	r := gen.placeRoom(enter, dir, gen.chooseRoomSize())
	if gen.anyWithin(r) {
		return image.ZR
	}
	return r
}

func (gen *roomGen) carveDoorway(room genRoomHandle, wall ecs.Entity) ecs.Entity {
	pt := gen.shard.pos.Get(wall).Point()
	gen.logf("doorway @%v", pt)
	gen.Floor.apply(gen.shard, wall)
	door := gen.Door.create(gen.shard, pt)
	// TODO set door behavior
	gen.rooms.parts.Insert(roomDoor, room.ID(), door.ID)
	return door
}

func (gen *roomGen) anyWithin(r image.Rectangle) bool {
	for q := gen.shard.pos.Within(r); q.Next(); {
		ent := q.handle().Entity()
		switch ent.Type() {
		case gen.Floor.t, gen.Wall.t, gen.Door.t:
			return true
		}
	}
	return false
}

func (gen *roomGen) chooseRoomSize() image.Point {
	return gen.RoomSize.Min.Add(image.Pt(
		rand.Intn(gen.RoomSize.Dx()),
		rand.Intn(gen.RoomSize.Dy()),
	))
}

func (gen *roomGen) placeRoom(enter, dir, sz image.Point) (r image.Rectangle) {
	// TODO better placement
	r.Min = enter
	if dir.Y == 0 {
		if dir.X == -1 {
			r.Min.X -= sz.X - 1
		}
		r.Min.Y -= rand.Intn(sz.Y-2) + 1
	} else { // dir.X == 0
		r.Min.X -= rand.Intn(sz.X-2) + 1
		if dir.Y == -1 {
			r.Min.Y -= sz.Y - 1
		}
	}
	r.Max = r.Min.Add(sz)
	return r
}

func (gen *roomGen) load(i int) (room genRoomHandle) {
	room.gen = gen
	room.i = i
	room.genRoom = &gen.data[i]
	room.r = gen.rooms.GetID(gen.ID(i))
	return room
}

func (room genRoomHandle) ID() ecs.ID {
	return room.gen.ArrayIndex.ID(room.i)
}

func (room genRoomHandle) wallNormal(p image.Point) (dir image.Point) {
	if p.X == room.r.Min.X {
		dir.X = -1
	} else if p.Y == room.r.Min.Y {
		dir.Y = -1
	} else if p.X == room.r.Max.X-1 {
		dir.X = 1
	} else if p.Y == room.r.Max.Y-1 {
		dir.Y = 1
	}
	return dir
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
