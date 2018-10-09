package borkgen

import (
	"image"

	"børk.com/hilbert"
	"børk.com/modspace"
	"børk.com/xorshiftstar"
)

const (
	// Scale is the height and width of the warehouse Hilbert curve.
	Scale = 16
	// Hilbert is the Hilbert curve of the warehouse.
	Hilbert = hilbert.Scale(Scale)
	// DoorChance is the number of walls of which one is likely to be a door.
	DoorChance = 6
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
	room.HilbertNum = Hilbert.Encode(hpt)
	room.Next = Hilbert.Decode(room.HilbertNum + 1)
	room.Prev = Hilbert.Decode(room.HilbertNum - 1)

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
		rng := xorshiftstar.New(room.HilbertNum ^ hilbertNorth)
		if rng.Uint64()%DoorChance == 0 {
			room.NorthDoor = true
		}
	}

	switch south {
	case room.Next, room.Prev:
	default:
		room.SouthWall = true
		rng := xorshiftstar.New(room.HilbertNum ^ hilbertSouth)
		if rng.Uint64()%DoorChance == 0 {
			room.SouthDoor = true
		}
	}

	switch west {
	case room.Next, room.Prev:
	default:
		room.WestWall = true
		rng := xorshiftstar.New(room.HilbertNum ^ hilbertWest)
		if rng.Uint64()%DoorChance == 0 {
			room.WestDoor = true
		}
	}

	switch east {
	case room.Next, room.Prev:
	default:
		room.EastWall = true
		rng := xorshiftstar.New(room.HilbertNum ^ hilbertEast)
		if rng.Uint64()%DoorChance == 0 {
			room.EastDoor = true
		}
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
