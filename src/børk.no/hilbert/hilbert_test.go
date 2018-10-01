package hilbert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecode(t *testing.T) {
	assert.Equal(t, Decode(0, 2), Decode(4, 2))
	assert.Equal(t, Decode(1, 4), Decode(17, 4))
	assert.NotEqual(t, Decode(1, 4), Decode(9, 4))

	a := Decode(0, 2)
	t.Logf("%s\n", a)
	b := Decode(1, 2)
	t.Logf("%s\n", b)
	c := Decode(2, 2)
	t.Logf("%s\n", c)
	d := Decode(3, 2)
	t.Logf("%s\n", d)
	e := Decode(4, 2)
	t.Logf("%s\n", e)
}
