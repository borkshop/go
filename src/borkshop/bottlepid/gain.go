package bottlepid

// Zero is for no gains for a term of the PID controller.
//
// Use Zero to eliminate a term (P, I, or D) from the controller.
var Zero = Gain{Over: 0, Under: 1}

// Gain is a fraction of integers.
//
// The PID controller uses fractions to compute the gain of each of its terms
// (P, I, od D).
type Gain struct {
	Over, Under int
}

// G is a short hand for expressing gain values like G(2, 3) for two thirds.
func G(over, under int) Gain {
	return Gain{Over: over, Under: under}
}

// Gain multiplies an integer by a fraction.
func (g Gain) Mul(n int) int {
	return n * g.Over / g.Under
}
