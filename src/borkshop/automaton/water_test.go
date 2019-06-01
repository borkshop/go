package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShedInt64(t *testing.T) {
	tests := []struct {
		earthLeft  int64
		waterLeft  int64
		earthRight int64
		waterRight int64
		wantFlow   int64
	}{
		{0, 0, 0, 0, 0},
		{
			0, 100,
			0, 0,
			50,
		},
		{
			0, 0,
			0, 100,
			-50,
		},
		{
			100, 0,
			0, 100,
			0,
		},
		{
			0, 100,
			100, 0,
			0,
		},
		{
			50, 50,
			0, 100,
			0,
		},
		{
			0, 100,
			50, 50,
			0,
		},
		{
			10, 10,
			0, 0,
			10,
		},
		{
			0, 0,
			10, 10,
			-10,
		},
		{
			10, 10,
			10, 0,
			5,
		},
		{
			100, 10,
			0, 0,
			10,
		},
		{
			0, 0,
			100, 10,
			-10,
		},
		{
			1, 1,
			0, 0,
			1,
		},
		{
			0, 0,
			1, 1,
			-1,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			flow := ShedInt64(tt.waterLeft, tt.waterRight, tt.earthLeft, tt.earthRight)
			assert.Equal(t, tt.wantFlow, flow, "unexpected outcome with flow")
		})
	}
}
