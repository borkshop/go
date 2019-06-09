package gl

import "github.com/chewxy/math32"

// Frustum creates a frustum view matrix with the given bounds.
func Frustum(left, right, bottom, top, near, far float32) Mat4 {
	rl := 1 / (right - left)
	tb := 1 / (top - bottom)
	nf := 1 / (near - far)
	return M4(
		(near*2)*rl, 0, 0, 0,
		0, (near*2)*tb, 0, 0,
		(right+left)*rl, (top+bottom)*tb, (far+near)*nf, -1,
		0, 0, (far*near*2)*nf, 0,
	)
}

// Perspective creates a perspective projection matrix with the given bounds.
func Perspective(fovy, aspect, near, far float32) Mat4 {
	f := 1.0 / math32.Tan(fovy/2)
	nf := 1 / (near - far)
	return M4(
		f/aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far+near)*nf, -1,
		0, 0, (2*far*near)*nf, 0,
	)
}

// FOV defines a field of view with up/down and left/right angles specified in
// radians.
type FOV struct {
	Up, Down    float32
	Left, Right float32
}

// PerspectiveFOV creates a perspecitve projection matrix with given FOV
// parameters and near/far values.
func PerspectiveFOV(fov FOV, near, far float32) Mat4 {
	var (
		upTan    = math32.Tan(fov.Up)
		downTan  = math32.Tan(fov.Down)
		leftTan  = math32.Tan(fov.Left)
		rightTan = math32.Tan(fov.Right)
		xScale   = 2.0 / (leftTan + rightTan)
		yScale   = 2.0 / (upTan + downTan)
	)
	return M4(
		xScale, 0, 0, 0,
		0, yScale, 0, 0,
		-((leftTan - rightTan) * xScale * 0.5), ((upTan - downTan) * yScale * 0.5), far/(near-far), -1,
		0, 0, (far*near)/(near-far), 0,
	)
}

// Ortho creates an orthogonal projection matrix with given frutum view bounds.
func Ortho(left, right, bottom, top, near, far float32) Mat4 {
	lr := 1 / (left - right)
	bt := 1 / (bottom - top)
	nf := 1 / (near - far)
	return M4(
		-2*lr, 0, 0, 0,
		0, -2*bt, 0, 0,
		0, 0, 2*nf, 0,
		(left+right)*lr, (top+bottom)*bt, (far+near)*nf, 1,
	)
}

// LookAt creates a look-at matrix with the given eye position, focal point,
// and up axis
func LookAt(eye, center, up Vec3) Mat4 {
	z := eye.Subtract(center)
	if length := z.Length(); length == 0 {
		return I4()
	} else {
		z = z.Scale(1 / length)
	}

	x := V3(up.Y*z.Z-up.Z*z.Y, up.Z*z.X-up.X*z.Z, up.X*z.Y-up.Y*z.X)
	if length := x.Length(); length == 0 {
		x = V3(0, 0, 0)
	} else {
		x = x.Scale(1 / length)
	}

	y := V3(z.Y*x.Z-z.Z*x.Y, z.Z*x.X-z.X*x.Z, z.X*x.Y-z.Y*x.X)
	if length := y.Length(); length == 0 {
		y = V3(0, 0, 0)
	} else {
		y = y.Scale(1 / length)
	}

	return M4(
		x.X, y.X, z.X, 0,
		x.Y, y.Y, z.Y, 0,
		x.Z, y.Z, z.Z, 0,
		-eye.Dot(x), -eye.Dot(y), -eye.Dot(z), 1,
	)
}
