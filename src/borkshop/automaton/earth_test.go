package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlideInt64(t *testing.T) {
	tests := []struct {
		giveLeft   int64
		giveRight  int64
		giveRepose int64
		wantLeft   int64
		wantRight  int64
		wantFlow   int64
	}{
		{
			0, 0, 0,
			0, 0, 0,
		},
		{
			2, 0, 1,
			1, 1, 1,
		},
		{
			2, 1, 1,
			2, 1, 0,
		},
		{
			1, 2, 1,
			1, 2, 0,
		},
		{
			0, 3, 1,
			1, 2, 1,
		},
		{
			3, 0, 1,
			2, 1, 1,
		},
		{
			100, 300, 2,
			199, 201, 99,
		},
		{
			300, 100, 4,
			202, 198, 98,
		},
		{
			0, 50, 2,
			24, 26, 24,
		},
		{
			0, 50, 1,
			25, 25, 25,
		},
		{
			0, 100, 10,
			45, 55, 45,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			delta := SlideInt64(tt.giveLeft, tt.giveRight, tt.giveRepose)
			left := tt.giveLeft - delta
			right := tt.giveRight + delta
			flow := mag64(delta)
			assert.Equal(t, tt.wantLeft, left, "unexpect outcome on left")
			assert.Equal(t, tt.wantRight, right, "unexpected outcome on right")
			assert.Equal(t, tt.wantFlow, flow, "unexpected outcome with flow")
		})
	}
}
