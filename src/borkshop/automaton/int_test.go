package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMulFrac64(t *testing.T) {
	tests := []struct {
		num, factor    int64
		fractionalBits uint
		entropy        uint64
		want           int64
	}{
		{
			num:            1,
			factor:         1,
			fractionalBits: 0,
			entropy:        0,
			want:           1,
		},
		{
			num:            1,
			factor:         0,
			fractionalBits: 0,
			entropy:        0,
			want:           0,
		},

		// Half of 2 is always 1, regardless of entropy since there is no
		// mantissa component of the product.
		{
			num:            2,
			factor:         1,
			fractionalBits: 1, // So factor means half
			entropy:        0, // Entropy should be irrelevant
			want:           1,
		},
		{
			num:            2,
			factor:         1,
			fractionalBits: 1, // So factor means half
			entropy:        1, // Entropy should be irrelevant
			want:           1,
		},

		// However, half of 1 needs to be 0 half the time, 1 the other half the
		// time, with varying entropy.
		// Only the last bit of entropy matters.
		// The order does not matter.
		{
			num:            1,
			factor:         1,
			fractionalBits: 1, // Thus factor is half (0.1)
			entropy:        0,
			want:           1, // This is passing as long as the next is the opposite.
		},
		{
			num:            1,
			factor:         1,
			fractionalBits: 1,
			entropy:        1,
			want:           0,
		},

		// Half with two bits of mantissa.
		// Each of these vary the entropy.
		// Half of the cases are 0, half are 1.
		{
			num:            1,
			factor:         2,
			fractionalBits: 2,
			entropy:        0,
			want:           1,
		},
		{
			num:            1,
			factor:         2,
			fractionalBits: 2,
			entropy:        1,
			want:           1,
		},
		{
			num:            1,
			factor:         2,
			fractionalBits: 2,
			entropy:        2,
			want:           0,
		},
		{
			num:            1,
			factor:         2,
			fractionalBits: 2,
			entropy:        3,
			want:           0,
		},
	}

	for _, tt := range tests {
		mask := int64(1<<tt.fractionalBits) - 1
		name := fmt.Sprintf("%xx%x.%x(%x)", tt.num, tt.factor>>tt.fractionalBits, tt.factor&mask, tt.entropy)
		t.Run(name, func(t *testing.T) {
			got := mulFrac64(tt.num, tt.factor, tt.fractionalBits, tt.entropy)
			assert.Equal(t, tt.want, got)
		})
	}
}
