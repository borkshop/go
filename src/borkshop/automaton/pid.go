package main

import "fmt"

type PID struct {
	// Static
	Target           int64
	ProportionalGain int64
	IntegralGain     int64
	DifferentialGain int64
	Min              int64
	Max              int64
	// Dynamic
	Value        int64
	Control      int64
	Proportional int64
	Integral     int64
	Differential int64
}

func (c *PID) Tick(value int64) {
	err := c.Target - value
	c.Proportional = c.ProportionalGain * err
	c.Differential = c.DifferentialGain * (value - c.Value)
	c.Integral = clamp64(c.Integral+c.IntegralGain*err+c.Differential, c.Min, c.Max)
	c.Control = clamp64(c.Proportional+c.Integral+c.Differential, c.Min, c.Max)
	c.Value = value
}

func (c *PID) String() string {
	return fmt.Sprintf("P:%d I:%d D:%d SP:%d PV:%d CP:%d", c.Proportional, c.Integral, c.Differential, c.Target, c.Value, c.Control)
}
