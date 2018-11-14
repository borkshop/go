package borkgen

import (
	"image"
	"math/rand"
)

type alignment int

const (
	horizontal alignment = 1 << iota
	vertical
)

func measureFloor(room *Room, rng rand.Source64, alignment alignment) image.Rectangle {
	if room.HilbertNum&1 == 0 {
		return measureSmallFloor(room, rng, alignment)
	}
	return measureLargeFloor(room, rng, alignment)
}

func measureSmallFloor(room *Room, rng rand.Source64, alignment alignment) image.Rectangle {
	floor := room.Floor.Add(room.Pt).Inset(2)
	if alignment&vertical != 0 && room.WestMargin&1 != 0 {
		floor.Min.X++
	}
	if alignment&horizontal != 0 && room.NorthMargin&1 != 0 {
		floor.Min.Y++
	}
	return floor
}

func measureLargeFloor(room *Room, rng rand.Source64, alignment alignment) image.Rectangle {
	floor := room.Floor.Add(room.Pt).Inset(1)

	// Depending on stride and whether there are walls, a room might be able to
	// grow into the cells where there would have been walls.
	// We take the opportunity, one direction or the other, if it arises, but
	// if there are opportunities in both directions, we choose a direction
	// randomly.
	var canGrowLat, canGrowLong int
	var canGrowSouth, canGrowNorth, canGrowEast, canGrowWest int

	// For west and north, we are also obliged to ensure that that the aisles
	// pass between the display items, so we offset the position of the first
	// item by one in each dimension, depending on whether the margin is even
	// or odd.
	// Only if we have bumped the furniture south or east by one is there a
	// possibility that we can shift north or west by two to fill the floor
	// where there would have been a wall.
	if alignment&vertical != 0 {
		if room.WestMargin&1 == 0 {
			floor.Min.X++
			if !room.WestWall {
				canGrowWest += 2
				canGrowLat += 2
			}
		}
	} else if !room.WestWall {
		canGrowWest++
		canGrowLat++
	}

	if alignment&horizontal != 0 {
		if room.NorthMargin&1 == 0 {
			floor.Min.Y++
			if !room.NorthWall {
				canGrowNorth += 2
				canGrowLong += 2
			}
		}
	} else if !room.NorthWall {
		canGrowNorth++
		canGrowLong++
	}

	if (alignment&horizontal == 0 || room.EastMargin&1 == 0) && !room.EastWall {
		canGrowEast++
		canGrowLat++
	}

	if (alignment&vertical == 0 || room.SouthMargin&1 == 0) && !room.SouthWall {
		canGrowSouth++
		canGrowLong++
	}

	// All things being equal, we do not favor latitudinal or logintudinal
	// growth over the other.
	var favorLong bool
	if canGrowLong*room.Size.Y == canGrowLat*room.Size.X {
		favorLong = rng.Uint64()&1 == 0
	} else {
		favorLong = canGrowLong > canGrowLat
	}

	if favorLong {
		floor.Max.Y += canGrowSouth
		floor.Min.Y -= canGrowNorth
	} else {
		floor.Max.X += canGrowEast
		floor.Min.X -= canGrowWest
	}
	return floor
}
