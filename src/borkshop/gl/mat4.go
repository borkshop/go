package gl

import "github.com/chewxy/math32"

// Mat4 is a 4x4 matrix.
type Mat4 struct {
	X, Y, Z, W Vec4
}

func M4(
	a00, a01, a02, a03 float32,
	a10, a11, a12, a13 float32,
	a20, a21, a22, a23 float32,
	a30, a31, a32, a33 float32,
) Mat4 {
	return Mat4{
		V4(a00, a01, a02, a03),
		V4(a10, a11, a12, a13),
		V4(a20, a21, a22, a23),
		V4(a30, a31, a32, a33),
	}
}

// I4 creates a new identity Mat4.
func I4() Mat4 {
	return M4(
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	)
}

// Transpose returns a transposed copy of the receiver.
func (m Mat4) Transpose() Mat4 {
	m.X.Y, m.Y.X = m.Y.X, m.X.Y
	m.X.Z, m.Z.X = m.Z.X, m.X.Z
	m.X.W, m.W.X = m.W.X, m.X.W
	m.Y.Z, m.Z.Y = m.Z.Y, m.Y.Z
	m.Y.W, m.W.Y = m.W.Y, m.Y.W
	m.Z.W, m.W.Z = m.W.Z, m.Z.W
	return m
}

// Invert returns an copy of the receiver matrix, inverted if possible, and
// true only if so.
func (m Mat4) Invert() (Mat4, bool) {
	b00 := m.X.X*m.Y.Y - m.X.Y*m.Y.X
	b01 := m.X.X*m.Y.Z - m.X.Z*m.Y.X
	b02 := m.X.X*m.Y.W - m.X.W*m.Y.X
	b03 := m.X.Y*m.Y.Z - m.X.Z*m.Y.Y
	b04 := m.X.Y*m.Y.W - m.X.W*m.Y.Y
	b05 := m.X.Z*m.Y.W - m.X.W*m.Y.Z
	b06 := m.Z.X*m.W.Y - m.Z.Y*m.W.X
	b07 := m.Z.X*m.W.Z - m.Z.Z*m.W.X
	b08 := m.Z.X*m.W.W - m.Z.W*m.W.X
	b09 := m.Z.Y*m.W.Z - m.Z.Z*m.W.Y
	b10 := m.Z.Y*m.W.W - m.Z.W*m.W.Y
	b11 := m.Z.Z*m.W.W - m.Z.W*m.W.Z

	if det := b00*b11 - b01*b10 + b02*b09 + b03*b08 - b04*b07 + b05*b06; det != 0 {
		det = 1.0 / det
		return M4(
			(m.Y.Y*b11-m.Y.Z*b10+m.Y.W*b09)*det,
			(m.X.Z*b10-m.X.Y*b11-m.X.W*b09)*det,
			(m.W.Y*b05-m.W.Z*b04+m.W.W*b03)*det,
			(m.Z.Z*b04-m.Z.Y*b05-m.Z.W*b03)*det,
			(m.Y.Z*b08-m.Y.X*b11-m.Y.W*b07)*det,
			(m.X.X*b11-m.X.Z*b08+m.X.W*b07)*det,
			(m.W.Z*b02-m.W.X*b05-m.W.W*b01)*det,
			(m.Z.X*b05-m.Z.Z*b02+m.Z.W*b01)*det,
			(m.Y.X*b10-m.Y.Y*b08+m.Y.W*b06)*det,
			(m.X.Y*b08-m.X.X*b10-m.X.W*b06)*det,
			(m.W.X*b04-m.W.Y*b02+m.W.W*b00)*det,
			(m.Z.Y*b02-m.Z.X*b04-m.Z.W*b00)*det,
			(m.Y.Y*b07-m.Y.X*b09-m.Y.Z*b06)*det,
			(m.X.X*b09-m.X.Y*b07+m.X.Z*b06)*det,
			(m.W.Y*b01-m.W.X*b03-m.W.Z*b00)*det,
			(m.Z.X*b03-m.Z.Y*b01+m.Z.Z*b00)*det,
		), true
	}

	return m, false
}

// Adjoint returns the adjugate of the receiver matrix.
func (m Mat4) Adjoint() Mat4 {
	a00, a01, a02, a03 := m.X.X, m.X.Y, m.X.Z, m.X.W
	a10, a11, a12, a13 := m.Y.X, m.Y.Y, m.Y.Z, m.Y.W
	a20, a21, a22, a23 := m.Z.X, m.Z.Y, m.Z.Z, m.Z.W
	a30, a31, a32, a33 := m.W.X, m.W.Y, m.W.Z, m.W.W
	return M4(
		(a11*(a22*a33-a23*a32) - a21*(a12*a33-a13*a32) + a31*(a12*a23-a13*a22)),
		-(a01*(a22*a33-a23*a32) - a21*(a02*a33-a03*a32) + a31*(a02*a23-a03*a22)),
		(a01*(a12*a33-a13*a32) - a11*(a02*a33-a03*a32) + a31*(a02*a13-a03*a12)),
		-(a01*(a12*a23-a13*a22) - a11*(a02*a23-a03*a22) + a21*(a02*a13-a03*a12)),
		-(a10*(a22*a33-a23*a32) - a20*(a12*a33-a13*a32) + a30*(a12*a23-a13*a22)),
		(a00*(a22*a33-a23*a32) - a20*(a02*a33-a03*a32) + a30*(a02*a23-a03*a22)),
		-(a00*(a12*a33-a13*a32) - a10*(a02*a33-a03*a32) + a30*(a02*a13-a03*a12)),
		(a00*(a12*a23-a13*a22) - a10*(a02*a23-a03*a22) + a20*(a02*a13-a03*a12)),
		(a10*(a21*a33-a23*a31) - a20*(a11*a33-a13*a31) + a30*(a11*a23-a13*a21)),
		-(a00*(a21*a33-a23*a31) - a20*(a01*a33-a03*a31) + a30*(a01*a23-a03*a21)),
		(a00*(a11*a33-a13*a31) - a10*(a01*a33-a03*a31) + a30*(a01*a13-a03*a11)),
		-(a00*(a11*a23-a13*a21) - a10*(a01*a23-a03*a21) + a20*(a01*a13-a03*a11)),
		-(a10*(a21*a32-a22*a31) - a20*(a11*a32-a12*a31) + a30*(a11*a22-a12*a21)),
		(a00*(a21*a32-a22*a31) - a20*(a01*a32-a02*a31) + a30*(a01*a22-a02*a21)),
		-(a00*(a11*a32-a12*a31) - a10*(a01*a32-a02*a31) + a30*(a01*a12-a02*a11)),
		(a00*(a11*a22-a12*a21) - a10*(a01*a22-a02*a21) + a20*(a01*a12-a02*a11)),
	)
}

// Determinant returns the determinant of the receiver matrix.
func (m Mat4) Determinant() float32 {
	a00, a01, a02, a03 := m.X.X, m.X.Y, m.X.Z, m.X.W
	a10, a11, a12, a13 := m.Y.X, m.Y.Y, m.Y.Z, m.Y.W
	a20, a21, a22, a23 := m.Z.X, m.Z.Y, m.Z.Z, m.Z.W
	a30, a31, a32, a33 := m.W.X, m.W.Y, m.W.Z, m.W.W

	b00 := a00*a11 - a01*a10
	b01 := a00*a12 - a02*a10
	b02 := a00*a13 - a03*a10
	b03 := a01*a12 - a02*a11
	b04 := a01*a13 - a03*a11
	b05 := a02*a13 - a03*a12
	b06 := a20*a31 - a21*a30
	b07 := a20*a32 - a22*a30
	b08 := a20*a33 - a23*a30
	b09 := a21*a32 - a22*a31
	b10 := a21*a33 - a23*a31
	b11 := a22*a33 - a23*a32

	return b00*b11 - b01*b10 + b02*b09 + b03*b08 - b04*b07 + b05*b06
}

// Multiply the argument matrix into a copy of the receiver, returning the
// product matrix.
func (m Mat4) Multiply(b Mat4) Mat4 {
	xco := V4(m.X.X, m.Y.X, m.Z.X, m.W.X)
	yco := V4(m.X.Y, m.Y.Y, m.Z.Y, m.W.Y)
	zco := V4(m.X.Z, m.Y.Z, m.Z.Z, m.W.Z)
	wco := V4(m.X.W, m.Y.W, m.Z.W, m.W.W)
	m.X = V4(b.X.Dot(xco), b.X.Dot(yco), b.X.Dot(zco), b.X.Dot(wco))
	m.Y = V4(b.Y.Dot(xco), b.Y.Dot(yco), b.Y.Dot(zco), b.Y.Dot(wco))
	m.Z = V4(b.Z.Dot(xco), b.Z.Dot(yco), b.Z.Dot(zco), b.Z.Dot(wco))
	m.W = V4(b.W.Dot(xco), b.W.Dot(yco), b.W.Dot(zco), b.W.Dot(wco))
	return m
}

// Translate returns a copy of the receiver matrix translated by the given vector.
func (m Mat4) Translate(v Vec3) Mat4 {
	x, y, z := v.X, v.Y, v.Z
	m.W.X += m.X.X*x + m.Y.X*y + m.Z.X*z
	m.W.Y += m.X.Y*x + m.Y.Y*y + m.Z.Y*z
	m.W.Z += m.X.Z*x + m.Y.Z*y + m.Z.Z*z
	m.W.W += m.X.W*x + m.Y.W*y + m.Z.W*z
	return m
}

// Translate creates a translation matrix from a vector translation.
func Translate(v Vec3) Mat4 {
	return M4(
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		v.X, v.Y, v.Z, 1,
	)
}

// Translation returns the translation vector component within the receiver
// matrix.
func (m Mat4) Translation() Vec3 {
	return V3(m.W.X, m.W.Y, m.W.Z)
}

// Scale returns a copy of the receiver matrix scaled by the dimensions in the
// given vector.
func (m Mat4) Scale(v Vec3) Mat4 {
	m.X = m.X.Scale(v.X)
	m.Y = m.Y.Scale(v.Y)
	m.Z = m.Z.Scale(v.Z)
	return m
}

// Scale creates a scaling matrix from a vector of dimensional scaling
// factors.
func Scale(v Vec3) Mat4 {
	return M4(
		v.X, 0, 0, 0,
		0, v.Y, 0, 0,
		0, 0, v.Z, 0,
		0, 0, 0, 1,
	)
}

// Rotate returns a copy of the receiver matrix that's been rotated the given
// angle around the given axis.
func (m Mat4) Rotate(rad float32, axis Vec3) Mat4 {
	length := axis.SquaredLength()
	if math32.Abs(length) < _epsilon {
		return m
	}
	axis = axis.Scale(1 / length)

	s := math32.Sin(rad)
	c := math32.Cos(rad)
	t := 1 - c

	// Perform rotation-specific matrix multiplication
	xco := V3(m.X.X, m.Y.X, m.Z.X)
	yco := V3(m.X.Y, m.Y.Y, m.Z.Y)
	zco := V3(m.X.Z, m.Y.Z, m.Z.Z)
	wco := V3(m.X.W, m.Y.W, m.Z.W)

	x, y, z := axis.X, axis.Y, axis.Z
	r := axis.Multiply(V3(x*t+c, x*t+z*s, x*t-y*s))
	m.X = V4(r.Dot(xco), r.Dot(yco), r.Dot(zco), r.Dot(wco))

	r = axis.Multiply(V3(y*t-z*s, y*t+c, y*t+x*s))
	m.Y = V4(r.Dot(xco), r.Dot(yco), r.Dot(zco), r.Dot(wco))

	r = axis.Multiply(V3(z*t+y*s, z*t-x*s, z*t+c))
	m.Z = V4(r.Dot(xco), r.Dot(yco), r.Dot(zco), r.Dot(wco))

	return m
}

// Rotate creates a rotation matrix of a given angle around a given axis.
func Rotate(rad float32, axis Vec3) Mat4 {
	length := axis.SquaredLength()
	if math32.Abs(length) < _epsilon {
		return I4()
	}
	axis = axis.Scale(1 / length)

	s := math32.Sin(rad)
	c := math32.Cos(rad)
	t := 1 - c

	x, y, z := axis.X, axis.Y, axis.Z
	return M4(
		x*x*t+c, y*x*t+z*s, z*x*t-y*s, 0,
		x*y*t-z*s, y*y*t+c, z*y*t+x*s, 0,
		x*z*t+y*s, y*z*t-x*s, z*z*t+c, 0,
		0, 0, 0, 1,
	)
}

// RotateX returns a copy of the receiver matrix rotated around the X axis by the given angle.
func (m Mat4) RotateX(rad float32) Mat4 {
	if math32.Abs(rad) < _epsilon {
		return m
	}
	s := math32.Sin(rad)
	c := math32.Cos(rad)
	a10, a11, a12, a13 := m.Y.X, m.Y.Y, m.Y.Z, m.Y.W
	a20, a21, a22, a23 := m.Z.X, m.Z.Y, m.Z.Z, m.Z.W
	m.Y = V4(
		a10*c+a20*s,
		a11*c+a21*s,
		a12*c+a22*s,
		a13*c+a23*s,
	)
	m.Z = V4(
		a20*c-a10*s,
		a21*c-a11*s,
		a22*c-a12*s,
		a23*c-a13*s,
	)
	return m
}

// RotateX creates an X rotation matrix by the given angle.
func RotateX(ang float32) Mat4 {
	s := math32.Sin(ang)
	c := math32.Cos(ang)
	return M4(
		1, 0, 0, 0,
		0, c, s, 0,
		0, -s, c, 0,
		0, 0, 0, 1,
	)
}

// RotateY returns a copy of the receiver matrix rotated around the Y axis by the given angle.
func (m Mat4) RotateY(rad float32) Mat4 {
	if math32.Abs(rad) < _epsilon {
		return m
	}
	s := math32.Sin(rad)
	c := math32.Cos(rad)
	a00, a01, a02, a03 := m.X.X, m.X.Y, m.X.Z, m.X.W
	a20, a21, a22, a23 := m.Z.X, m.Z.Y, m.Z.Z, m.Z.W
	m.X = V4(
		a00*c-a20*s,
		a01*c-a21*s,
		a02*c-a22*s,
		a03*c-a23*s,
	)
	m.Z = V4(
		a00*s+a20*c,
		a01*s+a21*c,
		a02*s+a22*c,
		a03*s+a23*c,
	)
	return m
}

// RotateY creates an Y rotation matrix by the given angle.
func RotateY(ang float32) Mat4 {
	s := math32.Sin(ang)
	c := math32.Cos(ang)
	return M4(
		c, 0, -s, 0,
		0, 1, 0, 0,
		s, 0, c, 0,
		0, 0, 0, 1,
	)
}

// RotateZ returns a copy of the receiver matrix rotated around the Z axis by the given angle.
func (m Mat4) RotateZ(rad float32) Mat4 {
	if math32.Abs(rad) < _epsilon {
		return m
	}
	s := math32.Sin(rad)
	c := math32.Cos(rad)
	a00, a01, a02, a03 := m.X.X, m.X.Y, m.X.Z, m.X.W
	a10, a11, a12, a13 := m.Y.X, m.Y.Y, m.Y.Z, m.Y.W
	m.X = V4(
		a00*c+a10*s,
		a01*c+a11*s,
		a02*c+a12*s,
		a03*c+a13*s,
	)
	m.Y = V4(
		a10*c-a00*s,
		a11*c-a01*s,
		a12*c-a02*s,
		a13*c-a03*s,
	)
	return m
}

// RotateZ creates an Z rotation matrix by the given angle.
func RotateZ(ang float32) Mat4 {
	s := math32.Sin(ang)
	c := math32.Cos(ang)
	return M4(
		c, s, 0, 0,
		-s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	)
}

// Frob returns the Frobenius norm of the receiver matrix.
func (m Mat4) Frob() float32 {
	return math32.Sqrt(
		math32.Pow(m.X.X, 2) +
			math32.Pow(m.X.Y, 2) +
			math32.Pow(m.X.Z, 2) +
			math32.Pow(m.X.W, 2) +
			math32.Pow(m.Y.X, 2) +
			math32.Pow(m.Y.Y, 2) +
			math32.Pow(m.Y.Z, 2) +
			math32.Pow(m.Y.W, 2) +
			math32.Pow(m.Z.X, 2) +
			math32.Pow(m.Z.Y, 2) +
			math32.Pow(m.Z.Z, 2) +
			math32.Pow(m.Z.W, 2) +
			math32.Pow(m.W.X, 2) +
			math32.Pow(m.W.Y, 2) +
			math32.Pow(m.W.Z, 2) +
			math32.Pow(m.W.W, 2))
}

// Add the argument matrix to a copy of the receiver, returning the sum.
func (m Mat4) Add(b Mat4) Mat4 {
	m.X = m.X.Add(b.X)
	m.Y = m.Y.Add(b.Y)
	m.Z = m.Z.Add(b.Z)
	m.W = m.W.Add(b.W)
	return m
}

// Subtract the argument matrix from a copy of the receiver, returning the
// difference.
func (m Mat4) Subtract(b Mat4) Mat4 {
	m.X = m.X.Subtract(b.X)
	m.Y = m.Y.Subtract(b.Y)
	m.Z = m.Z.Subtract(b.Z)
	m.W = m.W.Subtract(b.W)
	return m
}

// MultiplyScalar multiplies the argument to each element in a copy of the
// receiver, returning the product.
func (m Mat4) MultiplyScalar(s float32) Mat4 {
	m.X = m.X.Scale(s)
	m.Y = m.Y.Scale(s)
	m.Z = m.Z.Scale(s)
	m.W = m.W.Scale(s)
	return m
}

// Abs returns the element-wise absolute value of the receiver matrix.
func (m Mat4) Abs() Mat4 {
	return M4(
		math32.Abs(m.X.X),
		math32.Abs(m.X.Y),
		math32.Abs(m.X.Z),
		math32.Abs(m.X.W),
		math32.Abs(m.Y.X),
		math32.Abs(m.Y.Y),
		math32.Abs(m.Y.Z),
		math32.Abs(m.Y.W),
		math32.Abs(m.Z.X),
		math32.Abs(m.Z.Y),
		math32.Abs(m.Z.Z),
		math32.Abs(m.Z.W),
		math32.Abs(m.W.X),
		math32.Abs(m.W.Y),
		math32.Abs(m.W.Z),
		math32.Abs(m.W.W),
	)
}

// Equals returns true only if all elements of the given matrix are exactly
// equal to the corresponding elements in the receiver matrix.
func (m Mat4) Equals(b Mat4) bool {
	return (m.X.X == b.X.X &&
		m.X.Y == b.X.Y &&
		m.X.Z == b.X.Z &&
		m.X.W == b.X.W &&
		m.Y.X == b.Y.X &&
		m.Y.Y == b.Y.Y &&
		m.Y.Z == b.Y.Z &&
		m.Y.W == b.Y.W &&
		m.Z.X == b.Z.X &&
		m.Z.Y == b.Z.Y &&
		m.Z.Z == b.Z.Z &&
		m.Z.W == b.Z.W &&
		m.W.X == b.W.X &&
		m.W.Y == b.W.Y &&
		m.W.Z == b.W.Z &&
		m.W.W == b.W.W)
}

// Within returns true only if all of the given argument's elements are within
// epsilon of the receiver's elements.
func (m Mat4) Within(b Mat4, epsilon float32) bool {
	d := m.Subtract(b).Abs()
	m = m.Abs()
	b = b.Abs()
	return (d.X.X <= epsilon*math32.Max(1, math32.Max(m.X.X, b.X.X)) &&
		d.X.Y <= epsilon*math32.Max(1, math32.Max(m.X.Y, b.X.Y)) &&
		d.X.Z <= epsilon*math32.Max(1, math32.Max(m.X.Z, b.X.Z)) &&
		d.X.W <= epsilon*math32.Max(1, math32.Max(m.X.W, b.X.W)) &&
		d.Y.X <= epsilon*math32.Max(1, math32.Max(m.Y.X, b.Y.X)) &&
		d.Y.Y <= epsilon*math32.Max(1, math32.Max(m.Y.Y, b.Y.Y)) &&
		d.Y.Z <= epsilon*math32.Max(1, math32.Max(m.Y.Z, b.Y.Z)) &&
		d.Y.W <= epsilon*math32.Max(1, math32.Max(m.Y.W, b.Y.W)) &&
		d.Z.X <= epsilon*math32.Max(1, math32.Max(m.Z.X, b.Z.X)) &&
		d.Z.Y <= epsilon*math32.Max(1, math32.Max(m.Z.Y, b.Z.Y)) &&
		d.Z.Z <= epsilon*math32.Max(1, math32.Max(m.Z.Z, b.Z.Z)) &&
		d.Z.W <= epsilon*math32.Max(1, math32.Max(m.Z.W, b.Z.W)) &&
		d.W.X <= epsilon*math32.Max(1, math32.Max(m.W.X, b.W.X)) &&
		d.W.Y <= epsilon*math32.Max(1, math32.Max(m.W.Y, b.W.Y)) &&
		d.W.Z <= epsilon*math32.Max(1, math32.Max(m.W.Z, b.W.Z)) &&
		d.W.W <= epsilon*math32.Max(1, math32.Max(m.W.W, b.W.W)))
}

// TransformMat4 returns a tranformed copy of the receiver vector by appyling
// the argument transformation matrix.
func (v Vec3) TransformMat4(m Mat4) Vec3 {
	xco := V3(m.X.X, m.Y.X, m.Z.X)
	yco := V3(m.X.Y, m.Y.Y, m.Z.Y)
	zco := V3(m.X.Z, m.Y.Z, m.Z.Z)
	wco := V3(m.X.W, m.Y.W, m.Z.W)
	v = V3(v.Dot(xco), v.Dot(yco), v.Dot(zco))
	if w := v.Dot(wco); w != 0 {
		v = v.Scale(1 / w)
	}
	return v
}
