package quadindex_test

import (
	"image"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"borkshop/quadindex"
)

func TestIndex_queries(t *testing.T) {
	for _, tc := range []struct {
		name   string
		data   []image.Point
		zeros  []image.Point
		within map[image.Rectangle][]int
	}{
		{
			name:  "origin",
			data:  []image.Point{image.Pt(0, 0)},
			zeros: testPointRing(image.Rect(-1, -1, 1, 1)),
			within: map[image.Rectangle][]int{
				testRectAround(image.Pt(0, 0)):   {0}, // centered around only
				testRectAround(image.Pt(-1, -1)): nil, // Max exclusive
				testRectAround(image.Pt(1, 1)):   {0}, // Min inclusive
				testRectAround(image.Pt(42, 42)): nil, // far point
			},
		},
		{
			name: "four-quadrant square",
			data: []image.Point{
				image.Pt(8, 8),
				image.Pt(8, -8),
				image.Pt(-8, -8),
				image.Pt(-8, 8),
			},

			zeros: flattenPoints(
				testRingAround(image.Pt(8, 8)),
				testRingAround(image.Pt(8, -8)),
				testRingAround(image.Pt(-8, -8)),
				testRingAround(image.Pt(-8, 8)),
			),

			within: map[image.Rectangle][]int{
				testRectAround(image.Pt(8, 8)):   {0},
				testRectAround(image.Pt(8, -8)):  {1},
				testRectAround(image.Pt(-8, -8)): {2},
				testRectAround(image.Pt(-8, 8)):  {3},
				image.Rect(7, -9, 9, 9):          {0, 1},
				image.Rect(-9, -9, -7, 9):        {2, 3},
				image.Rect(-9, 7, 9, 9):          {0, 3},
				image.Rect(-9, -9, 9, -7):        {1, 2},
			},
		},

		{
			name: "3x3 square",
			data: testPointRect(image.Rect(0, 0, 3, 3)),
		},

		{
			name: "center point on a 3x3 square",
			data: append(
				testPointRect(image.Rect(0, 0, 3, 3)),
				image.Pt(1, 1)),
		},

		{
			name: "3x3 inset on a 5x5 square",
			data: append(
				testPointRect(image.Rect(0, 0, 5, 5)),
				testPointRect(image.Rect(1, 1, 4, 4))...),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var qi quadindex.Index
			for i, pt := range tc.data {
				qi.Update(i, pt)
			}

			// readback via At
			for i, pt := range tc.data {
				found := false
				for q := qi.At(pt); q.Next(); {
					if q.I() == i {
						found = true
						break
					}
				}
				if !assert.True(t, found, "expected to find [%v]%v", i, pt) {
					t.Logf("q: %v", qi.At(pt))
					dump(&qi, t.Logf)
				}
			}
			if t.Failed() {
				return
			}

			// At zeros
			for _, pt := range tc.zeros {
				if q := qi.At(pt); !assert.False(t, q.Next(), "expected zero at %v", pt) {
					t.Logf("q: %v", qi.At(pt))
					dump(&qi, t.Logf)
				}
			}
			if t.Failed() {
				return
			}

			// Within queries
			for r, is := range tc.within {
				var res []int
				for q := qi.Within(r); q.Next(); {
					res = append(res, q.I())
				}
				sort.Ints(res)
				if !assert.Equal(t, is, res, "expected points within %v", r) {
					t.Logf("q: %v", qi.Within(r))
					dump(&qi, t.Logf)
				}
			}
		})
	}
}

func TestIndex_mutation(t *testing.T) {
	var qi quadindex.Index

	// generate
	func() {
		id := 0
		qi.Update(id, image.ZP)
		id++
		for _, pt := range testPointRing(image.Rect(-2, -2, 2, 2)) {
			qi.Update(id, pt)
			assert.Equal(t, pt, qi.Get(id).Pt(), "must be able to readback point[%v]", id)
			id++
		}
	}()
	if t.Failed() {
		return
	}

	// move around
	func() {
		id := 0
		for stepi, step := range []struct {
			d image.Point
			n int
			e image.Point
		}{
			{image.Pt(0, 0), 0, image.Pt(0, 0)},
			{image.Pt(1, 0), 5, image.Pt(1, 0)},
			{image.Pt(0, 1), 5, image.Pt(1, 1)},
			{image.Pt(-1, 0), 5, image.Pt(-1, 1)},
			{image.Pt(0, -1), 5, image.Pt(-1, -1)},
		} {
			for i := 0; i < step.n; i++ {
				pos := qi.Get(id).Pt()
				pos = pos.Add(step.d)
				any := false
				for q := qi.At(pos); q.Next(); {
					any = true
					t.Logf("hit id:%v @%v", q.I(), pos)
					break
				}
				if any {
					break
				}
				qi.Update(id, pos)
				t.Logf("move id:%v to %v", id, pos)
			}
			assert.Equal(t, step.e, qi.Get(id).Pt(), "expected position after step[%v]", stepi)
		}
	}()
}

func dump(qi *quadindex.Index, logf func(string, ...interface{})) {
	logf("i,ix,key")
	for i := 0; i < qi.Len(); i++ {
		ix, k := qi.Data(i)
		logf("%v,%v,%v", i, ix, k)
	}
}

func flattenPoints(ptss ...[]image.Point) []image.Point {
	n := 0
	for _, pts := range ptss {
		n += len(pts)
	}
	r := make([]image.Point, 0, n)
	for _, pts := range ptss {
		r = append(r, pts...)
	}
	return r
}

func testRectAround(pt image.Point) image.Rectangle {
	return image.Rectangle{
		pt.Sub(image.Pt(1, 1)),
		pt.Add(image.Pt(1, 1)),
	}
}

func testRingAround(pt image.Point) []image.Point {
	return testPointRing(testRectAround(pt))
}

func testPointRing(r image.Rectangle) (pts []image.Point) {
	pt := r.Min
	pts = make([]image.Point, 0, 2*r.Dx()+2*r.Dy())
	for _, st := range []struct {
		d image.Point
		n int
	}{
		{image.Pt(1, 0), r.Dx()},
		{image.Pt(0, 1), r.Dy()},
		{image.Pt(-1, 0), r.Dx()},
		{image.Pt(0, -1), r.Dy()},
	} {
		for i := 0; i < st.n; i++ {
			pts = append(pts, pt)
			pt = pt.Add(st.d)
		}
	}
	return pts
}

func testPointRect(r image.Rectangle) (pts []image.Point) {
	pts = make([]image.Point, 0, r.Dx()*r.Dy())
	for pt := r.Min; pt.Y < r.Max.Y; pt.Y++ {
		for pt.X = r.Min.X; pt.X < r.Max.X; pt.X++ {
			pts = append(pts, pt)
		}
	}
	return pts
}
