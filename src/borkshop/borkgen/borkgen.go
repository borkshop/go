package borkgen

import (
	"image"

	"borkshop/hilbert"
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
	// WarehouseCount is the number of warehouses in the world.
	WarehouseCount = (Area >> 4 / 5)
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
	// Region is the bounding box for the area, suitable for modulo.
	Region = image.Rectangle{image.ZP, image.Point{Scale, Scale}}
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

	room.NorthMargin = int(2 + topMarginRand.Uint64()%3 + topMarginRand.Uint64()%3)
	room.SouthMargin = int(2 + bottomMarginRand.Uint64()%3 + bottomMarginRand.Uint64()%3)
	room.WestMargin = int(2 + leftMarginRand.Uint64()%3 + leftMarginRand.Uint64()%3)
	room.EastMargin = int(2 + rightMarginRand.Uint64()%3 + rightMarginRand.Uint64()%3)

	width := int(1 + room.WestMargin + room.EastMargin)
	height := int(1 + room.NorthMargin + room.SouthMargin)
	room.Size = image.Point{width, height}

	room.Floor = image.Rectangle{
		image.Pt(-room.WestMargin-1, -room.NorthMargin-1),
		image.Pt(room.EastMargin+2, room.SouthMargin+2),
	}

	north := hpt.Add(North).Mod(Region)
	south := hpt.Add(South).Mod(Region)
	west := hpt.Add(West).Mod(Region)
	east := hpt.Add(East).Mod(Region)

	hilbertNorth := Hilbert.Encode(north)
	hilbertSouth := Hilbert.Encode(south)
	hilbertWest := Hilbert.Encode(west)
	hilbertEast := Hilbert.Encode(east)

	room.IsWarehouse = isWarehouse(room.HilbertNum)
	room.WarehouseNum = warehouseNum(room.HilbertNum)

	if isWall(room, north, hilbertNorth) {
		room.NorthWall = true
		room.NorthDoor = isDoor(room.HilbertNum, hilbertNorth)
	}

	if isWall(room, south, hilbertSouth) {
		room.SouthWall = true
		room.SouthDoor = isDoor(room.HilbertNum, hilbertSouth)
	}

	if isWall(room, west, hilbertWest) {
		room.WestWall = true
		room.WestDoor = isDoor(room.HilbertNum, hilbertWest)
	}

	if isWall(room, east, hilbertEast) {
		room.EastWall = true
		room.EastDoor = isDoor(room.HilbertNum, hilbertEast)
	}

	return room
}

func isWall(room *Room, otherPt image.Point, otherHilbertNum int) bool {
	if otherPt == room.Next || otherPt == room.Prev {
		// The only pair of adjacent rooms that we block is the boundary
		// between a warehouse and the next.
		return room.WarehouseNum != warehouseNum(otherHilbertNum)
	}
	// Otherwise, the only walls we do not build are the internal walls of a
	// warehouse.
	return !(room.IsWarehouse &&
		isWarehouse(otherHilbertNum) &&
		room.WarehouseNum == warehouseNum(otherHilbertNum))
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

// Canvas is a surface on which to draw a showroom.
type Canvas interface {
	FillFloor(image.Rectangle)
	FillWall(image.Rectangle)
	FillAisle(image.Rectangle)
	FillDisplay(image.Rectangle, string, Color)
	FillStack(image.Rectangle)
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
		drawShowroom(canvas, room)
	}
}
