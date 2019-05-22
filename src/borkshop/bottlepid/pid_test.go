package bottlepid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStablePID(t *testing.T) {
	controller := Controller{
		Proportional: G(1, 3),
		Differential: G(1, 3),
		Integral:     G(1, 3),
		Value:        1000000,
		Min:          0,
		Max:          100000000000,
	}

	var a, b Generation

	for i := 0; i < 60; i++ {
		controller.Tick(&a, &b, b.Control-1000)
		a, b = b, a
		t.Logf("control %d value %d\n", a.Control, a.Value)
	}
	assert.Equal(t, a.Control-1000, a.Value)
}
