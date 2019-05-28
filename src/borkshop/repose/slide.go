package repose

// Slide takes two columns and normalizes them about their mean such
// that the differences in the heights of the column does not exceed
// the angle of repose.
func Slide(left, right, repose int) (int, int, int) {
	delta := (left+right)/2 - repose/2
	if left > right {
		delta = delta - right
	} else {
		delta = left - delta
	}
	left -= delta
	right += delta
	return left, right, mag(delta)
}

func mag(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
