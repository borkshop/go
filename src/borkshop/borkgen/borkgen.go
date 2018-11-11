package borkgen

import (
	"image"

	"borkshop/hilbert"
	"borkshop/modspace"
	"borkshop/xorshiftstar"
)

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
	NorthMargin, SouthMargin, WestMargin, EastMargin int
	NorthWall, SouthWall, WestWall, EastWall         bool
	NorthDoor, SouthDoor, WestDoor, EastDoor         bool
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

	north := Space.Add(hpt, North)
	south := Space.Add(hpt, South)
	west := Space.Add(hpt, West)
	east := Space.Add(hpt, East)

	hilbertNorth := Hilbert.Encode(north)
	hilbertSouth := Hilbert.Encode(south)
	hilbertWest := Hilbert.Encode(west)
	hilbertEast := Hilbert.Encode(east)

	switch north {
	case room.Next, room.Prev:
	default:
		room.NorthWall = true
		room.NorthDoor = isDoor(room.HilbertNum, hilbertNorth)
	}

	switch south {
	case room.Next, room.Prev:
	default:
		room.SouthWall = true
		room.SouthDoor = isDoor(room.HilbertNum, hilbertSouth)
	}

	switch west {
	case room.Next, room.Prev:
	default:
		room.WestWall = true
		room.WestDoor = isDoor(room.HilbertNum, hilbertWest)
	}

	switch east {
	case room.Next, room.Prev:
	default:
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
	m := a
	if b < a {
		m = b
	}
	rng := xorshiftstar.New(a ^ b)
	return m&1 == 0 && rng.Uint64()%DoorChance == 0
}

// Canvas is a surface on which to draw a showroom.
type Canvas interface {
	FillFloor(image.Rectangle)
	FillWall(image.Rectangle)
	FillAisle(image.Rectangle)
}

// Memo tracks whether a room has been drawn for the given hilbert point.
type Memo interface {
	SetRoomDrawn(num int)
	IsRoomDrawn(num int) bool
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
	for room.Pt.Y < within.Max.Y {
		for room.Pt.X < within.Max.X {
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
	if memo.IsRoomDrawn(room.HilbertNum) {
		return
	}
	memo.SetRoomDrawn(room.HilbertNum)

	// center
	canvas.FillAisle(unitRect.Add(room.Pt))

	// corners
	canvas.FillWall(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, -room.NorthMargin-1)).
		Add(room.Pt))
	canvas.FillWall(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, -room.NorthMargin-1)).
		Add(room.Pt))
	canvas.FillWall(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, room.SouthMargin+1)).
		Add(room.Pt))
	canvas.FillWall(image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, room.SouthMargin+1)).
		Add(room.Pt))

	// aisles
	fillFloorOrAisle(canvas, room.NorthWall, image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(0, -room.NorthMargin)).
		Add(room.Pt))
	fillFloorOrAisle(canvas, room.SouthWall, image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(0, 1)).
		Add(room.Pt))
	fillFloorOrAisle(canvas, room.WestWall, image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, 0)).
		Add(room.Pt))
	fillFloorOrAisle(canvas, room.EastWall, image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, 0)).
		Add(room.Pt))

	// floor quadrants
	canvas.FillFloor(image.Rectangle{image.ZP, image.Pt(room.WestMargin, room.NorthMargin)}.
		Add(image.Pt(-room.WestMargin, -room.NorthMargin)).
		Add(room.Pt))
	canvas.FillFloor(image.Rectangle{image.ZP, image.Pt(room.WestMargin, room.SouthMargin)}.
		Add(image.Pt(-room.WestMargin, 1)).
		Add(room.Pt))
	canvas.FillFloor(image.Rectangle{image.ZP, image.Pt(room.EastMargin, room.NorthMargin)}.
		Add(image.Pt(1, -room.NorthMargin)).
		Add(room.Pt))
	canvas.FillFloor(image.Rectangle{image.ZP, image.Pt(room.EastMargin, room.SouthMargin)}.
		Add(image.Pt(1, 1)).
		Add(room.Pt))

	// north wall segments
	fillWallOrFloor(canvas, room.NorthWall, image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, -room.NorthMargin-1)).
		Add(room.Pt))
	fillWallOrFloor(canvas, room.NorthWall, image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, -room.NorthMargin-1)).
		Add(room.Pt))
	fillDoorMaybe(canvas, room.NorthWall, room.NorthDoor, image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(0, -room.NorthMargin-1)).
		Add(room.Pt))

	// south wall segments
	fillWallOrFloor(canvas, room.SouthWall, image.Rectangle{image.ZP, image.Pt(room.WestMargin, 1)}.
		Add(image.Pt(-room.WestMargin, room.SouthMargin+1)).
		Add(room.Pt))
	fillWallOrFloor(canvas, room.SouthWall, image.Rectangle{image.ZP, image.Pt(room.EastMargin, 1)}.
		Add(image.Pt(1, room.SouthMargin+1)).
		Add(room.Pt))
	fillDoorMaybe(canvas, room.SouthWall, room.SouthDoor, image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(0, room.SouthMargin+1)).
		Add(room.Pt))

	// west wall segments
	fillWallOrFloor(canvas, room.WestWall, image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(-room.WestMargin-1, -room.NorthMargin)).
		Add(room.Pt))
	fillWallOrFloor(canvas, room.WestWall, image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(-room.WestMargin-1, 1)).
		Add(room.Pt))
	fillDoorMaybe(canvas, room.WestWall, room.WestDoor, image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(-room.WestMargin-1, 0)).
		Add(room.Pt))

	// east wall segments
	fillWallOrFloor(canvas, room.EastWall, image.Rectangle{image.ZP, image.Pt(1, room.NorthMargin)}.
		Add(image.Pt(room.EastMargin+1, -room.NorthMargin)).
		Add(room.Pt))
	fillWallOrFloor(canvas, room.EastWall, image.Rectangle{image.ZP, image.Pt(1, room.SouthMargin)}.
		Add(image.Pt(room.EastMargin+1, 1)).
		Add(room.Pt))
	fillDoorMaybe(canvas, room.EastWall, room.EastDoor, image.Rectangle{image.ZP, image.Pt(1, 1)}.
		Add(image.Pt(room.EastMargin+1, 0)).
		Add(room.Pt))
}

func fillWallOrFloor(canvas Canvas, wall bool, rect image.Rectangle) {
	if wall {
		canvas.FillWall(rect)
	} else {
		canvas.FillFloor(rect)
	}
}

func fillWallOrAisle(canvas Canvas, wall bool, rect image.Rectangle) {
	if wall {
		canvas.FillWall(rect)
	} else {
		canvas.FillAisle(rect)
	}
}

func fillFloorOrAisle(canvas Canvas, floor bool, rect image.Rectangle) {
	if floor {
		canvas.FillFloor(rect)
	} else {
		canvas.FillAisle(rect)
	}
}

func fillDoorMaybe(canvas Canvas, wall bool, door bool, rect image.Rectangle) {
	if wall {
		if door {
			canvas.FillFloor(rect)
		} else {
			canvas.FillWall(rect)
		}
	} else {
		canvas.FillAisle(rect)
	}
}
