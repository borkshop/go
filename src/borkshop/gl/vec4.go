package gl

import "github.com/chewxy/math32"

// Vec4 is a 4-dimensional vector.
type Vec4 struct {
	X, Y, Z, W float32
}

func V4(x, y, z, w float32) Vec4 {
	return Vec4{x, y, z, w}
}

// Add the argument to a copy of the receiver, returning the sum.
func (v Vec4) Add(b Vec4) Vec4 {
	v.X += b.X
	v.Y += b.Y
	v.Z += b.Z
	v.W += b.W
	return v
}

// Subtract the argument from a copy of the receiver, returning the difference.
func (v Vec4) Subtract(b Vec4) Vec4 {
	v.X -= b.X
	v.Y -= b.Y
	v.Z -= b.Z
	v.W -= b.W
	return v
}

// Multiply the argument element-wise into a copy of the receiver, returning
// the product.
func (v Vec4) Multiply(b Vec4) Vec4 {
	v.X *= b.X
	v.Y *= b.Y
	v.Z *= b.Z
	v.W *= b.W
	return v
}

// Divide the argument element-wise into a copy of the receiver, returning
// the product.
func (v Vec4) Divide(b Vec4) Vec4 {
	v.X /= b.X
	v.Y /= b.Y
	v.Z /= b.Z
	v.W /= b.W
	return v
}

// Ceil returns the element-wise ceiling of the receiver.
func (v Vec4) Ceil() Vec4 {
	return Vec4{
		math32.Ceil(v.X),
		math32.Ceil(v.Y),
		math32.Ceil(v.Z),
		math32.Ceil(v.W),
	}
}

// Floor returns the element-wise floor of the receiver.
func (v Vec4) Floor() Vec4 {
	return Vec4{
		math32.Floor(v.X),
		math32.Floor(v.Y),
		math32.Floor(v.Z),
		math32.Floor(v.W),
	}
}

// Abs returns the element-wise absolute value of the receiver.
func (v Vec4) Abs() Vec4 {
	return Vec4{
		math32.Abs(v.X),
		math32.Abs(v.Y),
		math32.Abs(v.Z),
		math32.Abs(v.W),
	}
}

// Min returns the element-wise minimum of the receiver and the argument.
func (v Vec4) Min(b Vec4) Vec4 {
	return Vec4{
		math32.Min(v.X, b.X),
		math32.Min(v.Y, b.Y),
		math32.Min(v.Z, b.Z),
		math32.Min(v.W, b.W),
	}
}

// Max returns the element-wise minimum of the receiver and the argument.
func (v Vec4) Max(b Vec4) Vec4 {
	return Vec4{
		math32.Max(v.X, b.X),
		math32.Max(v.Y, b.Y),
		math32.Max(v.Z, b.Z),
		math32.Max(v.W, b.W),
	}
}

// Scale returns a copy of the receiver scaled by the given factor.
func (v Vec4) Scale(s float32) Vec4 {
	return Vec4{
		v.X * s,
		v.Y * s,
		v.Z * s,
		v.W * s,
	}
}

// Negate returns a copy of the receiver with each element negated.
func (v Vec4) Negate() Vec4 {
	return Vec4{
		-v.X,
		-v.Y,
		-v.Z,
		-v.W,
	}
}

// Inverse returns a copy of the receiver with each element inverted.
func (v Vec4) Inverse() Vec4 {
	return Vec4{
		1.0 / v.X,
		1.0 / v.Y,
		1.0 / v.Z,
		1.0 / v.W,
	}
}

// SquaredDistance computes the squared euclidian distance between the receiver
// and argument vectors.
func (v Vec4) SquaredDistance(b Vec4) float32 {
	dx := b.X - v.X
	dy := b.Y - v.Y
	dz := b.Z - v.Z
	dw := b.W - v.W
	return dx*dx + dy*dy + dz*dz + dw*dw
}

// Distance computes the euclidian distance between the receiver and argument
// vectors.
func (v Vec4) Distance(b Vec4) float32 {
	return math32.Sqrt(v.SquaredDistance(b))
}

// SquaredLength computes the squared length of the receiver.
func (v Vec4) SquaredLength() float32 {
	x := v.X
	y := v.Y
	z := v.Z
	w := v.W
	return x*x + y*y + z*z + w*w
}

// Length computes the squared length of the receiver.
func (v Vec4) Length() float32 {
	return math32.Sqrt(v.SquaredLength())
}

// Normalize returns a normalized copy of the receiver.
func (v Vec4) Normalize() Vec4 {
	if n := v.SquaredLength(); n > 0 {
		n = 1 / math32.Sqrt(n)
		v.X *= n
		v.Y *= n
		v.Z *= n
		v.W *= n
	}
	return v
}

// Sum returns the total of the receiver's elements.
func (v Vec4) Sum() float32 {
	return v.X + v.Y + v.Z + v.W
}

// Dot returns the dot product of the receiver and argument vectors.
func (v Vec4) Dot(b Vec4) float32 {
	return v.Multiply(b).Sum()
}

// Equals returns true only if all elements of the given vector are exactly
// equal to the corresponding elements in the receiver vector.
func (v Vec4) Equals(b Vec4) bool {
	return v.X == b.X && v.Y == b.Y && v.Z == b.Z && v.W == b.W
}

// Within returns true only if all of the given argument's elements are within
// epsilon of the receiver's elements.
func (v Vec4) Within(b Vec4, epsilon float32) bool {
	d := v.Subtract(b).Abs()
	v = v.Abs()
	b = b.Abs()
	return (d.X <= epsilon*math32.Max(1, math32.Max(v.X, b.X)) &&
		d.Y <= epsilon*math32.Max(1, math32.Max(v.Y, b.Y)) &&
		d.Z <= epsilon*math32.Max(1, math32.Max(v.Z, b.Z)) &&
		d.W <= epsilon*math32.Max(1, math32.Max(v.W, b.W)))
}
