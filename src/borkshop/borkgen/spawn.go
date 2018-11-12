package borkgen

import (
	"image"
	"math/rand"
)

// Spawn returns a random starting location within the labyrinth.
func Spawn() image.Point {
	hilbert := rand.Intn(WarehouseCount)<<4*5 - 1
	return Hilbert.Decode(hilbert & Mask)
}
