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
	NorthDoor, SouthDoor, WestDoor, EastDoor         int
}

// DescribeRoom describes a room at a particular position.
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

	room.NorthMargin = int(1 + topMarginRand.Uint64()%7)
	room.SouthMargin = int(1 + bottomMarginRand.Uint64()%7)
	room.WestMargin = int(1 + leftMarginRand.Uint64()%7)
	room.EastMargin = int(1 + rightMarginRand.Uint64()%7)

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
		if rng.Uint64()%3 == 1 {
			room.NorthDoor = int(1+rng.Uint64()) % (width - 2)
		}
	}

	switch south {
	case room.Next, room.Prev:
	default:
		room.SouthWall = true
		rng := xorshiftstar.New(room.Hilbert ^ hilbertSouth)
		if rng.Uint64()%3 == 1 {
			room.SouthDoor = int(1+rng.Uint64()) % (width - 2)
		}
	}

	switch west {
	case room.Next, room.Prev:
	default:
		room.WestWall = true
		rng := xorshiftstar.New(room.Hilbert ^ hilbertWest)
		if rng.Uint64()%3 == 1 {
			room.WestDoor = int(1+rng.Uint64()) % (height - 2)
		}
	}

	switch east {
	case room.Next, room.Prev:
	default:
		room.EastWall = true
		rng := xorshiftstar.New(room.Hilbert ^ hilbertEast)
		if rng.Uint64()%3 == 1 {
			room.EastDoor = int(1+rng.Uint64()) % (height - 2)
		}
	}

	return room
}
