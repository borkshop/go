package quadindex

import (
	"fmt"
	"image"
)

// Key is a quadtree key value.
type Key uint64

const (
	keySet Key = 1 << (64 - iota - 1)
	keyInval

	keyBits     = 64 - iota
	keyCompBits = keyBits / 2

	keyMask     = 1<<(keyBits+1) - 1
	keyCompMask = 1<<(keyCompBits+1) - 1

	minInt = -1<<(keyCompBits-1) + 1
	maxInt = 1<<(keyCompBits-1) - 1
)

func (k Key) String() string {
	if !k.Set() {
		return "Key(unset)"
	}
	if k.invalid() {
		return fmt.Sprintf("Key%v*", k.Pt())
	}
	return fmt.Sprintf("Key%v", k.Pt())
}

func (k Key) invalid() bool {
	return k&keyInval != 0
}

// Set returns true only if the key value is marked as "set" or defined; the
// point of an unset key value is meaningless; in practice you should only see
// zero key values that are unset.
func (k Key) Set() bool {
	return k&keySet != 0
}

// Pt reconstructs the (maybe truncated) image point encoded by the key value.
// Returns image.ZP if the key value is not "set".
func (k Key) Pt() (p image.Point) {
	if k&keySet == 0 {
		return image.ZP
	}
	x, y := combine(uint64(k&keyMask)), combine(uint64(k&keyMask)>>1)
	p.X = int(x) + minInt
	p.Y = int(y) + minInt
	return p
}

// MakeKey encodes an image point, truncating it if necessary, returning its
// Corresponding key value.
func MakeKey(p image.Point) Key {
	z := zkey(truncQuadComponent(p.X), truncQuadComponent(p.Y))
	return Key(z) | keySet
}

func zkey(x, y uint32) (z uint64) {
	return split(x) | split(y)<<1
}

func split(value uint32) (z uint64) {
	z = uint64(value & keyCompMask)
	z = (z ^ (z << 32)) & 0x000000007fffffff
	z = (z ^ (z << 16)) & 0x0000ffff0000ffff
	z = (z ^ (z << 8)) & 0x00ff00ff00ff00ff // 11111111000000001111111100000000..
	z = (z ^ (z << 4)) & 0x0f0f0f0f0f0f0f0f // 1111000011110000
	z = (z ^ (z << 2)) & 0x3333333333333333 // 11001100..
	z = (z ^ (z << 1)) & 0x5555555555555555 // 1010...
	return z
}

func combine(z uint64) uint32 {
	z = z & 0x5555555555555555
	z = (z ^ (z >> 1)) & 0x3333333333333333
	z = (z ^ (z >> 2)) & 0x0f0f0f0f0f0f0f0f
	z = (z ^ (z >> 4)) & 0x00ff00ff00ff00ff
	z = (z ^ (z >> 8)) & 0x0000ffff0000ffff
	z = (z ^ (z >> 16)) & 0x000000007fffffff
	return uint32(z)
}

func truncQuadComponent(n int) uint32 {
	if n < minInt {
		n = minInt
	}
	if n > maxInt {
		n = maxInt
	}
	return uint32(n - minInt)
}
