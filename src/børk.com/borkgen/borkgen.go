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
	Hilbert                                          int
	Next, Prev                                       image.Point
	At, Size                                         image.Point
	NorthMargin, SouthMargin, WestMargin, EastMargin int
	NorthWall, SouthWall, WestWall, EastWall         bool
	NorthDoor, SouthDoor, WestDoor, EastDoor         bool
}

// DescribeRoom describes a room at a particular coördinate on a Hilbert space.
func DescribeRoom(at image.Point) Room {
	room := Room{}

	room.At = at
	room.Hilbert = Hilbert.Encode(at)
	room.Next = Hilbert.Decode(room.Hilbert + 1)
	room.Prev = Hilbert.Decode(room.Hilbert - 1)

	topMarginRand := xorshiftstar.New(at.Y * 2)
	bottomMarginRand := xorshiftstar.New(at.Y*2 + 1)
	leftMarginRand := xorshiftstar.New(at.X * 2)
	rightMarginRand := xorshiftstar.New(at.X*2 + 1)

	room.NorthMargin = int(2 + topMarginRand.Uint64()%5)
	room.SouthMargin = int(2 + bottomMarginRand.Uint64()%5)
	room.WestMargin = int(2 + leftMarginRand.Uint64()%5)
	room.EastMargin = int(2 + rightMarginRand.Uint64()%5)

	width := int(1 + room.WestMargin + room.EastMargin)
	height := int(1 + room.NorthMargin + room.SouthMargin)
	room.Size = image.Point{width, height}

	north := Space.Add(at, North)
	south := Space.Add(at, South)
	west := Space.Add(at, West)
	east := Space.Add(at, East)

	hilbertNorth := Hilbert.Encode(north)
	hilbertSouth := Hilbert.Encode(south)
	hilbertWest := Hilbert.Encode(west)
	hilbertEast := Hilbert.Encode(east)

	switch north {
	case room.Next, room.Prev:
	default:
		room.NorthWall = true
		rng := xorshiftstar.New(room.Hilbert ^ hilbertNorth)
		if rng.Uint64()%4 == 0 {
			room.NorthDoor = true
		}
	}

	switch south {
	case room.Next, room.Prev:
	default:
		room.SouthWall = true
		rng := xorshiftstar.New(room.Hilbert ^ hilbertSouth)
		if rng.Uint64()%4 == 0 {
			room.SouthDoor = true
		}
	}

	switch west {
	case room.Next, room.Prev:
	default:
		room.WestWall = true
		rng := xorshiftstar.New(room.Hilbert ^ hilbertWest)
		if rng.Uint64()%4 == 0 {
			room.WestDoor = true
		}
	}

	switch east {
	case room.Next, room.Prev:
	default:
		room.EastWall = true
		rng := xorshiftstar.New(room.Hilbert ^ hilbertEast)
		if rng.Uint64()%4 == 0 {
			room.EastDoor = true
		}
	}

	return room
}
