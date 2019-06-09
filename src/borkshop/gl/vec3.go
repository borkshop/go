package gl

import (
	"github.com/chewxy/math32"
)

// Vec3 is a 3-dimensional vector.
type Vec3 struct {
	X, Y, Z float32
}

func V3(x, y, z float32) Vec3 {
	return Vec3{x, y, z}
}

// Add the argument to a copy of the receiver, returning the sum.
func (v Vec3) Add(b Vec3) Vec3 {
	v.X += b.X
	v.Y += b.Y
	v.Z += b.Z
	return v
}

// Subtract the argument from a copy of the receiver, returning the difference.
func (v Vec3) Subtract(b Vec3) Vec3 {
	v.X -= b.X
	v.Y -= b.Y
	v.Z -= b.Z
	return v
}

// Multiply the argument element-wise into a copy of the receiver, returning
// the product.
func (v Vec3) Multiply(b Vec3) Vec3 {
	v.X *= b.X
	v.Y *= b.Y
	v.Z *= b.Z
	return v
}

// Divide the argument element-wise into a copy of the receiver, returning
// the product.
func (v Vec3) Divide(b Vec3) Vec3 {
	v.X /= b.X
	v.Y /= b.Y
	v.Z /= b.Z
	return v
}

// Ceil returns the element-wise ceiling of the receiver.
func (v Vec3) Ceil() Vec3 {
	return Vec3{
		math32.Ceil(v.X),
		math32.Ceil(v.Y),
		math32.Ceil(v.Z),
	}
}

// Floor returns the element-wise floor of the receiver.
func (v Vec3) Floor() Vec3 {
	return Vec3{
		math32.Floor(v.X),
		math32.Floor(v.Y),
		math32.Floor(v.Z),
	}
}

// Abs returns the element-wise absolute value of the receiver.
func (v Vec3) Abs() Vec3 {
	return Vec3{
		math32.Abs(v.X),
		math32.Abs(v.Y),
		math32.Abs(v.Z),
	}
}

// Min returns the element-wise minimum of the receiver and the argument.
func (v Vec3) Min(b Vec3) Vec3 {
	return Vec3{
		math32.Min(v.X, b.X),
		math32.Min(v.Y, b.Y),
		math32.Min(v.Z, b.Z),
	}
}

// Max returns the element-wise minimum of the receiver and the argument.
func (v Vec3) Max(b Vec3) Vec3 {
	return Vec3{
		math32.Max(v.X, b.X),
		math32.Max(v.Y, b.Y),
		math32.Max(v.Z, b.Z),
	}
}

// Scale returns a copy of the receiver scaled by the given factor.
func (v Vec3) Scale(s float32) Vec3 {
	return Vec3{
		v.X * s,
		v.Y * s,
		v.Z * s,
	}
}

// Negate returns a copy of the receiver with each element negated.
func (v Vec3) Negate() Vec3 {
	return Vec3{
		-v.X,
		-v.Y,
		-v.Z,
	}
}

// Inverse returns a copy of the receiver with each element inverted.
func (v Vec3) Inverse() Vec3 {
	return Vec3{
		1.0 / v.X,
		1.0 / v.Y,
		1.0 / v.Z,
	}
}

// SquaredDistance computes the squared euclidian distance between the receiver
// and argument vectors.
func (v Vec3) SquaredDistance(b Vec3) float32 {
	dx := b.X - v.X
	dy := b.Y - v.Y
	dz := b.Z - v.Z
	return dx*dx + dy*dy + dz*dz
}

// Distance computes the euclidian distance between the receiver and argument
// vectors.
func (v Vec3) Distance(b Vec3) float32 {
	return math32.Sqrt(v.SquaredDistance(b))
}

// SquaredLength computes the squared length of the receiver.
func (v Vec3) SquaredLength() float32 {
	x := v.X
	y := v.Y
	z := v.Z
	return x*x + y*y + z*z
}

// Length computes the squared length of the receiver.
func (v Vec3) Length() float32 {
	return math32.Sqrt(v.SquaredLength())
}

// Normalize returns a normalized copy of the receiver.
func (v Vec3) Normalize() Vec3 {
	if n := v.SquaredLength(); n > 0 {
		n = 1 / math32.Sqrt(n)
		v.X *= n
		v.Y *= n
		v.Z *= n
	}
	return v
}

// Sum returns the total of the receiver's elements.
func (v Vec3) Sum() float32 {
	return v.X + v.Y + v.Z
}

// Dot returns the dot product of the receiver and argument vectors.
func (v Vec3) Dot(b Vec3) float32 {
	return v.Multiply(b).Sum()
}

// Cross returns the cross product of the receiver and argument vectors.
func (v Vec3) Cross(b Vec3) Vec3 {
	ax, ay, az := v.X, v.Y, v.Z
	bx, by, bz := b.X, b.Y, b.Z
	return Vec3{
		ay*bz - az*by,
		az*bx - ax*bz,
		ax*by - ay*bx,
	}
}

// Angle returns the angle between the receiver and argument vectors.
func (v Vec3) Angle(b Vec3) float32 {
	if cosine := v.Normalize().Dot(b.Normalize()); cosine <= 1.0 {
		return math32.Acos(cosine)
	}
	return 0
}

// Equals returns true only if all elements of the given vector are exactly
// equal to the corresponding elements in the receiver vector.
func (v Vec3) Equals(b Vec3) bool {
	return v.X == b.X && v.Y == b.Y && v.Z == b.Z
}

// Within returns true only if all of the given argument's elements are within
// epsilon of the receiver's elements.
func (v Vec3) Within(b Vec3, epsilon float32) bool {
	d := v.Subtract(b).Abs()
	v = v.Abs()
	b = b.Abs()
	return (d.X <= epsilon*math32.Max(1, math32.Max(v.X, b.X)) &&
		d.Y <= epsilon*math32.Max(1, math32.Max(v.Y, b.Y)) &&
		d.Z <= epsilon*math32.Max(1, math32.Max(v.Z, b.Z)))
}
