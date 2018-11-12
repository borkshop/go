package borkgen

import "image"

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

	// rng := xorshiftstar.New(start.HilbertNum)
	floor = floor.Inset(3)
	for y := floor.Min.Y; y+2 <= floor.Max.Y; y += 4 {
		for x := floor.Min.X; x < floor.Max.X; x++ {
			canvas.FillStack(image.Rectangle{
				image.Point{x, y},
				image.Point{x + 1, y + 1},
			})
			canvas.FillStack(image.Rectangle{
				image.Point{x, y + 1},
				image.Point{x + 1, y + 2},
			})
		}
	}
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
