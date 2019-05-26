// Package bottlepid provides a PID Controller (Proportional, Integral, Differential).
//
// A PID controller is a generational algorithm that attempts to influence an
// independent variable (y, or the "value") by adjusting a dependent variable
// (x, or the "control").
// Each generation computes a new control value based on three weighted inputs.
//  P: proportional to the difference between the desired and current value,
//     the "error".
//  I: accumulated in proportion to the error over all prior generations.
//  D: proportional to the rate the value has changed since the previous
//     generation.
package bottlepid

// Controller configures a PID controller, based on the influence of each term,
// the target value, and the range of the target value.
type Controller struct {
	Proportional Gain
	Integral     Gain
	Differential Gain
	Value        int
	Min, Max     int
}

// Generation represents the state of the PID controller in one generation.
//
// The controller reads from the prior generation and writes all of the values
// of the new generation.
// Swapping the previous and next generations allows progress without
// allocation.
type Generation struct {
	Value        int
	Control      int
	Proportional int
	Integral     int
	Differential int
}

// WriteTo reports the state of a generation.
// func (gen *Generation) WriteTo(f io.Writer) {
// 	fmt.Fprintf(f, "%d\n", gen.Control)
// 	fmt.Fprintf(f, "We are trying to make Y go from %d to %d\n", gen.Value, pid.Value)
// 	fmt.Fprintf(f, "The difference (error) is %d\n", err)
// 	fmt.Fprintf(f, "The error integrated over all time is %d\n", gen.Integral)
// 	fmt.Fprintf(f, "The value of Y changed from %d to %d since the last tick\n", prev.Value, gen.Value)
// 	fmt.Fprintf(f, "So, we are changing X from %d to %d\n", prev.Control, gen.Control)
// 	fmt.Fprintf(f, "\n")
// }

// Tick applies the PID controller algorithm, producing a new generation's
// state from the prior state and the current measured value.
//
// The algorithm in particular adjusts the Control value for the new generation.
func (pid *Controller) Tick(next, prev *Generation, value int) {
	err := pid.Value - value

	next.Value = value
	next.Proportional = pid.Proportional.Mul(err)
	next.Differential = pid.Differential.Mul(value - prev.Value)
	next.Integral = clamp(prev.Integral+pid.Integral.Mul(err), pid.Min, pid.Max)
	next.Control = clamp(next.Proportional+next.Integral+next.Differential, pid.Min, pid.Max)
}

func clamp(i, min, max int) int {
	if i > max {
		return max
	}
	if i < min {
		return min
	}
	return i
}
