package bottle

// Resetter writes an initial generation.
type Resetter interface {
	Reset(gen *Generation)
}

// Resetters is a list of resetters to reset.
type Resetters []Resetter

var _ Resetter = Resetters(nil)

// Reset resets the generation with all the resetters.
func (resetters Resetters) Reset(gen *Generation) {
	for i := 0; i < len(resetters); i++ {
		resetters[i].Reset(gen)
	}
}
