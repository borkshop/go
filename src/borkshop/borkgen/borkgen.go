package borkgen

import (
	"image"
	"math/rand"

	"borkshop/hilbert"
	"borkshop/modspace"
	"borkshop/xorshiftstar"
)

// Color is a furnishing color.
type Color int

const (
	// Scale is the height and width of the warehouse Hilbert curve.
	Scale = 256
	// Area is the area of the Hilbert curve
	Area = Scale * Scale
	// Mask covers the bits of all numbers in the domain of the Hilbert curve.
	Mask = Area - 1
	// Hilbert is the Hilbert curve of the warehouse.
	Hilbert = hilbert.Scale(Scale)
	// DoorChance is the number of walls of which one is likely to be a door.
	DoorChance = 3
	// Margin is space between rooms including their walls.
	Margin = 3
)

const (
	// White is a furniture color.
	White Color = iota
	// Blond is a furniture color.
	Blond
	// Brown is a furniture color.
	Brown
	// Black is a furniture color.
	Black
)

var (
	// Space is the toroidal space of the warehouse Hilbert curve.
	Space = modspace.Space(image.Point{Scale, Scale})
	// North is the relative position of the northern point.
	North = image.Point{0, -1}
	// South is the relative position of the southern point.
	South = image.Point{0, 1}
	// West is the relative position of the southern point.
	West = image.Point{-1, 0}
	// East is the relative position of the eastern point.
	East = image.Point{1, 0}

	unitPt   = image.Point{1, 1}
	unitRect = image.Rectangle{image.ZP, unitPt}
)

// Room is a room description.
type Room struct {
	HilbertNum                                       int
	Next, Prev                                       image.Point
	Pt, HilbertPt, Size                              image.Point
	Floor                                            image.Rectangle
	NorthMargin, SouthMargin, WestMargin, EastMargin int
	NorthWall, SouthWall, WestWall, EastWall         bool
	NorthDoor, SouthDoor, WestDoor, EastDoor         bool
	IsWarehouse                                      bool
	WarehouseNum                                     int
}

// DescribeRoom describes a room at a particular coördinate on a Hilbert space.
func DescribeRoom(hpt image.Point) *Room {
	room := &Room{}

	room.HilbertPt = hpt
	room.HilbertNum = Hilbert.Encode(image.Pt(hpt.X&(Scale-1), hpt.Y&(Scale-1)))
	room.Next = Hilbert.Decode((room.HilbertNum + 1) & Mask)
	room.Prev = Hilbert.Decode((Area + room.HilbertNum - 1) & Mask)

	topMarginRand := xorshiftstar.New(hpt.Y * 2)
	bottomMarginRand := xorshiftstar.New(hpt.Y*2 + 1)
	leftMarginRand := xorshiftstar.New(hpt.X * 2)
	rightMarginRand := xorshiftstar.New(hpt.X*2 + 1)

	room.NorthMargin = int(1 + topMarginRand.Uint64()%3 + topMarginRand.Uint64()%3)
	room.SouthMargin = int(1 + bottomMarginRand.Uint64()%3 + bottomMarginRand.Uint64()%3)
	room.WestMargin = int(1 + leftMarginRand.Uint64()%3 + leftMarginRand.Uint64()%3)
	room.EastMargin = int(1 + rightMarginRand.Uint64()%3 + rightMarginRand.Uint64()%3)

	width := int(1 + room.WestMargin + room.EastMargin)
	height := int(1 + room.NorthMargin + room.SouthMargin)
	room.Size = image.Point{width, height}

	room.Floor = image.Rectangle{
		image.Pt(-room.WestMargin-1, -room.NorthMargin-1),
		image.Pt(room.EastMargin+2, room.SouthMargin+2),
	}

	north := Space.Add(hpt, North)
	south := Space.Add(hpt, South)
	west := Space.Add(hpt, West)
	east := Space.Add(hpt, East)

	hilbertNorth := Hilbert.Encode(north)
	hilbertSouth := Hilbert.Encode(south)
	hilbertWest := Hilbert.Encode(west)
	hilbertEast := Hilbert.Encode(east)

	room.IsWarehouse = isWarehouse(room.HilbertNum)
	room.WarehouseNum = warehouseNum(room.HilbertNum)

	if north != room.Next && north != room.Prev && !(room.IsWarehouse && isWarehouse(hilbertNorth) && room.WarehouseNum == warehouseNum(hilbertNorth)) {
		room.NorthWall = true
		room.NorthDoor = isDoor(room.HilbertNum, hilbertNorth)
	}

	if south != room.Next && south != room.Prev && !(room.IsWarehouse && isWarehouse(hilbertSouth) && room.WarehouseNum == warehouseNum(hilbertSouth)) {
		room.SouthWall = true
		room.SouthDoor = isDoor(room.HilbertNum, hilbertSouth)
	}

	if west != room.Next && west != room.Prev && !(room.IsWarehouse && isWarehouse(hilbertWest) && room.WarehouseNum == warehouseNum(hilbertWest)) {
		room.WestWall = true
		room.WestDoor = isDoor(room.HilbertNum, hilbertWest)
	}

	if east != room.Next && east != room.Prev && !(room.IsWarehouse && isWarehouse(hilbertEast) && room.WarehouseNum == warehouseNum(hilbertEast)) {
		room.EastWall = true
		room.EastDoor = isDoor(room.HilbertNum, hilbertEast)
	}

	return room
}

// At returns a room by walking to it from the selected room.
func (r *Room) At(hpt image.Point) *Room {
	for r.HilbertPt.X > hpt.X {
		s := DescribeRoom(image.Pt(r.HilbertPt.X-1, r.HilbertPt.Y))
		s.Pt = r.Pt
		s.Pt.X -= r.WestMargin + Margin + s.EastMargin
		r = s
	}
	for r.HilbertPt.X < hpt.X {
		s := DescribeRoom(image.Pt(r.HilbertPt.X+1, r.HilbertPt.Y))
		s.Pt = r.Pt
		s.Pt.X += r.EastMargin + Margin + s.WestMargin
		r = s
	}
	for r.HilbertPt.Y > hpt.Y {
		s := DescribeRoom(image.Pt(r.HilbertPt.X, r.HilbertPt.Y-1))
		s.Pt = r.Pt
		s.Pt.Y -= r.NorthMargin + Margin + s.SouthMargin
		r = s
	}
	for r.HilbertPt.Y < hpt.Y {
		s := DescribeRoom(image.Pt(r.HilbertPt.X, r.HilbertPt.Y+1))
		s.Pt = r.Pt
		s.Pt.Y += r.SouthMargin + Margin + s.NorthMargin
		r = s
	}
	return r
}

func isDoor(a, b int) bool {
	if isWarehouse(a) || isWarehouse(b) {
		return false
	}
	if b < a {
		a, b = b, a
	}
	if b-a < 8 {
		return false
	}
	rng := xorshiftstar.New(a)
	return a&1 == 0 && int(rng.Uint64())&1 == 0
	// return a&1 == 0
}

func containingRoom(room *Room) *Room {
	if isWarehouse(room.HilbertNum) {
		start := room.HilbertNum & ^0xf
		at := Hilbert.Decode(start)
		return room.At(at)
	}
	return room
}

func warehouseNum(n int) int {
	return (n >> 4) / 5
}

func isWarehouse(n int) bool {
	return (n>>4)%5 == 0
}

func isWarehouseStart(n int) bool {
	return isWarehouse(n) && n&0xf == 0
}

// Canvas is a surface on which to draw a showroom.
type Canvas interface {
	FillFloor(image.Rectangle)
	FillWall(image.Rectangle)
	FillAisle(image.Rectangle)
	FillDisplay(image.Rectangle, string, Color)
}

// Memo tracks whether a room has been drawn for the given hilbert point.
type Memo interface {
	SetRoomDrawn(*Room)
	IsRoomDrawn(*Room) bool
}

// Draw paints a canvas within the given bounds.
func Draw(canvas Canvas, memo Memo, room *Room, within image.Rectangle) *Room {
	// find the northwest corner of the visible region
	for room.Pt.X > within.Min.X {
		room = room.At(image.Pt(room.HilbertPt.X-1, room.HilbertPt.Y))
	}
	for room.Pt.Y > within.Min.Y {
		room = room.At(image.Pt(room.HilbertPt.X, room.HilbertPt.Y-1))
	}
	// draw rooms in boustrophedon
	for room.Pt.Y < within.Max.Y+room.Size.Y {
		for room.Pt.X < within.Max.X+room.Size.X {
			drawRoom(canvas, memo, room)
			room = room.At(image.Pt(room.HilbertPt.X+1, room.HilbertPt.Y))
		}
		room = room.At(image.Pt(room.HilbertPt.X, room.HilbertPt.Y+1))
		for room.Pt.X+room.Size.X > within.Min.X {
			drawRoom(canvas, memo, room)
			room = room.At(image.Pt(room.HilbertPt.X-1, room.HilbertPt.Y))
		}
		room = room.At(image.Pt(room.HilbertPt.X, room.HilbertPt.Y+1))
	}
	return room
}

func drawRoom(canvas Canvas, memo Memo, room *Room) {
	room = containingRoom(room)

	if memo.IsRoomDrawn(room) {
		return
	}
	memo.SetRoomDrawn(room)

	if room.IsWarehouse {
		drawWarehouse(canvas, room)
	} else {
		drawWalls(canvas, room, image.ZR)
		drawShowroom(canvas, room)
	}
}

func drawWarehouse(canvas Canvas, start *Room) {
	floor := start.Floor.Add(start.Pt)
	room := start
	for i := 0; i < 16; i++ {
		floor = floor.Union(room.Floor.Add(room.Pt))
		room = room.At(room.Next)
	}
	room = start
	for i := 0; i < 16; i++ {
		drawWalls(canvas, room, floor.Inset(1))
		room = room.At(room.Next)
	}
	canvas.FillAisle(floor)
}

func drawShowroom(canvas Canvas, room *Room) {

	// Floor
	floor := room.Floor.Add(room.Pt)
	canvas.FillFloor(floor)

	// Center
	canvas.FillAisle(unitRect.Add(room.Pt))

	// Aisles
	fillAisle(canvas, !room.NorthWall, image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(0, -room.NorthMargin)).
		Add(room.Pt))
	fillAisle(canvas, !room.SouthWall, image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(0, 1)).
		Add(room.Pt))
	fillAisle(canvas, !room.WestWall, image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, 0)).
		Add(room.Pt))
	fillAisle(canvas, !room.EastWall, image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, 0)).
		Add(room.Pt))

	// Display items
	rng := xorshiftstar.New(room.HilbertNum)

	fillDisplaysUniformly(canvas, room, rng)
}

func drawWalls(canvas Canvas, room *Room, mask image.Rectangle) {
	// North wall segments
	fillWall(canvas, room.NorthWall, image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, -room.NorthMargin-1)).
		Add(room.Pt))
	fillWall(canvas, room.NorthWall, image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, -room.NorthMargin-1)).
		Add(room.Pt))
	fillDoor(canvas, room.NorthWall, room.NorthDoor, room.IsWarehouse, image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(0, -room.NorthMargin-1)).
		Add(room.Pt))

	// South wall segments
	fillWall(canvas, room.SouthWall, image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, room.SouthMargin+1)).
		Add(room.Pt))
	fillWall(canvas, room.SouthWall, image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, room.SouthMargin+1)).
		Add(room.Pt))
	fillDoor(canvas, room.SouthWall, room.SouthDoor, room.IsWarehouse, image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(0, room.SouthMargin+1)).
		Add(room.Pt))

	// West wall segments
	fillWall(canvas, room.WestWall, image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(-room.WestMargin-1, -room.NorthMargin)).
		Add(room.Pt))
	fillWall(canvas, room.WestWall, image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(-room.WestMargin-1, 1)).
		Add(room.Pt))
	fillDoor(canvas, room.WestWall, room.WestDoor, room.IsWarehouse, image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, 0)).
		Add(room.Pt))

	// East wall segments
	fillWall(canvas, room.EastWall, image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(room.EastMargin+1, -room.NorthMargin)).
		Add(room.Pt))
	fillWall(canvas, room.EastWall, image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(room.EastMargin+1, 1)).
		Add(room.Pt))
	fillDoor(canvas, room.EastWall, room.EastDoor, room.IsWarehouse, image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, 0)).
		Add(room.Pt))

	// Corners
	nw := image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, -room.NorthMargin-1)).
		Add(room.Pt)
	ne := image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, -room.NorthMargin-1)).
		Add(room.Pt)
	sw := image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, room.SouthMargin+1)).
		Add(room.Pt)
	se := image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, room.SouthMargin+1)).
		Add(room.Pt)
	// Except, to avoid drawing columns inside a warehouse, elide those columns
	// that overlap the mask.
	if !nw.Overlaps(mask) {
		canvas.FillWall(nw)
	}
	if !ne.Overlaps(mask) {
		canvas.FillWall(ne)
	}
	if !sw.Overlaps(mask) {
		canvas.FillWall(sw)
	}
	if !se.Overlaps(mask) {
		canvas.FillWall(se)
	}
}

func fillDisplaysUniformly(canvas Canvas, room *Room, rng rand.Source64) {
	floor := room.Floor.Add(room.Pt)
	itemFloor := image.Rectangle{
		floor.Min.Add(unitPt),
		floor.Max.Sub(unitPt),
	}

	// Depending on stride and whether there are walls, a room might be able to
	// grow into the cells where there would have been walls.
	// We take the opportunity, one direction or the other, if it arises, but
	// if there are opportunities in both directions, we choose a direction
	// randomly.
	var canGrowLat, canGrowLong int
	var canGrowSouth, canGrowNorth, canGrowEast, canGrowWest bool

	// For west and north, we are also obliged to ensure that that the aisles
	// pass between the display items, so we offset the position of the first
	// item by one in each dimension, depending on whether the margin is even
	// or odd.
	// Only if we have bumped the furniture south or east by one is there a
	// possibility that we can shift north or west by two to fill the floor
	// where there would have been a wall.
	if room.WestMargin&1 == 0 {
		itemFloor.Min.X++
		if !room.WestWall {
			canGrowWest = true
			canGrowLat++
		}
	}
	if room.NorthMargin&1 == 0 {
		itemFloor.Min.Y++
		if !room.NorthWall {
			canGrowNorth = true
			canGrowLong++
		}
	}
	if room.EastMargin&1 == 0 && !room.EastWall {
		canGrowEast = true
		canGrowLat++
	}
	if room.SouthMargin&1 == 0 && !room.SouthWall {
		canGrowSouth = true
		canGrowLong++
	}

	// All things being equal, we do not favor latitudinal or logintudinal
	// growth over the other.
	var favorLong bool
	if canGrowLong == canGrowLat {
		favorLong = rng.Uint64()&1 == 0
	} else {
		favorLong = canGrowLong > canGrowLat
	}

	if favorLong {
		if canGrowSouth {
			itemFloor.Max.Y++
		}
		if canGrowNorth {
			itemFloor.Min.Y -= 2
		}
	} else {
		if canGrowEast {
			itemFloor.Max.X++
		}
		if canGrowWest {
			itemFloor.Min.X -= 2
		}
	}

	for y := itemFloor.Min.Y; y < itemFloor.Max.Y; y += 2 {
		for x := itemFloor.Min.X; x < itemFloor.Max.X; x += 2 {
			i := int(rng.Uint64()>>1) % len(catalog)
			c := int(rng.Uint64()>>1) % 4
			canvas.FillDisplay(unitRect.Add(image.Pt(x, y)), catalog[i], Color(c))
		}
	}
}

func fillWall(canvas Canvas, wall bool, rect image.Rectangle) {
	if wall {
		canvas.FillWall(rect)
	}
}

func fillAisle(canvas Canvas, aisle bool, rect image.Rectangle) {
	if aisle {
		canvas.FillAisle(rect)
	}
}

func fillDoor(canvas Canvas, wall bool, door bool, warehouse bool, rect image.Rectangle) {
	if wall {
		if !door {
			canvas.FillWall(rect)
		}
	} else if !warehouse {
		canvas.FillAisle(rect)
	}
}
