package borkgen

import (
	"borkshop/xorshiftstar"
	"image"
	"math/rand"
)

func drawShowroom(canvas Canvas, room *Room) {
	drawWalls(canvas, room, image.ZR)

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
