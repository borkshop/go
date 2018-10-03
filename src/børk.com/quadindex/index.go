package quadindex

import (
	"fmt"
	"image"
	"sort"
)

// Index implements a linear quadtree.
type Index struct {
	index
	invalid int
}

// Get the key value stored for the given index.
func (qi *Index) Get(i int) Key {
	if i >= len(qi.ks) {
		return 0
	}
	return qi.ks[i]
}

// Update the point associated with the given index.
func (qi *Index) Update(i int, p image.Point) {
	if i < qi.index.Len() {
		prior := qi.ks[i]
		qi.ks[i] = MakeKey(p) | keySet | keyInval
		if !prior.invalid() {
			qi.invalid++
		}
		return
	}

	n := qi.index.Len()
	qi.index.alloc(i, keyInval)
	qi.ks[i] = MakeKey(p) | keySet | keyInval
	qi.invalid += qi.index.Len() - n
}

// Delete the point associated with the given index.
func (qi *Index) Delete(i int, p image.Point) {
	prior := qi.ks[i]
	qi.ks[i] = keyInval
	if !prior.invalid() {
		qi.invalid++
	}
}

// At returns a query cursor for all stored indices at the given point.
func (qi *Index) At(p image.Point) (qq Cursor) {
	qq.index = &qi.index
	qq.kmin = MakeKey(p)
	qq.kmax = qq.kmin
	qq.iimin = qi.search(qq.kmin)
	qq.iimax = qq.iimin
	if qq.iimin < len(qi.ix) {
		qq.iimax = qi.run(qq.iimax, qq.kmin)
		qq.ii = qq.iimin - 1
	} else {
		qq.ii = len(qi.ix)
	}
	return qq
}

// Within returns a query cursor for all stored indices within the given region.
func (qi *Index) Within(r image.Rectangle) (qq Cursor) {
	qq.index = &qi.index
	qq.r = r
	qq.kmin = MakeKey(r.Min)
	qq.kmax = MakeKey(r.Max)
	qq.iimin = qi.search(qq.kmin)
	qq.iimax = qq.iimin
	if qq.iimin < len(qi.ix) {
		qq.iimax = qi.run(qi.search(qq.kmax), qq.kmax)
		qq.ii = qq.iimin - 1
	} else {
		qq.ii = len(qi.ix)
	}
	return qq
}

func (qi *Index) reindex() {
	// collect invalid
	eoh := 0
	for jj := 0; jj < len(qi.ix); jj++ {
		i := qi.ix[jj]
		if k := qi.ks[i]; k&keyInval != 0 {
			qi.ks[i] &= ^keyInval
			if kk := qi.narrow(0, eoh, qi.ks[qi.ix[jj]]); kk != jj {
				copy(qi.ix[kk+1:jj+1], qi.ix[kk:])
				qi.ix[kk] = i
			}
			eoh++
		}
	}

	// merge
	for iiHead, iiBody := 0, eoh; iiHead < iiBody && iiBody < len(qi.ix); iiHead++ {
		if !qi.Less(iiHead, iiBody) {
			iiBody++
			rotateRight(qi.ix[iiHead:iiBody])
		}
	}

	qi.invalid = 0
}

func (qi *Index) resort() {
	for i, k := range qi.ks {
		if k&keyInval != 0 {
			qi.ks[i] = k & ^keyInval
		}
	}
	sort.Sort(qi.index)
	qi.invalid = 0
}

func (qi *Index) search(k Key) int {
	if qi.invalid > 0 {
		if qi.invalid >= len(qi.ix)/2 {
			qi.resort()
		} else {
			qi.reindex()
		}
	}
	return qi.index.search(k)
}

// Cursor is a point or region query on an Index.
type Cursor struct {
	*index
	r            image.Rectangle
	kmin, kmax   Key
	iimin, iimax int
	ii           int
}

func (qq Cursor) String() string {
	return fmt.Sprintf("quadCursor(%v := range iimin:%v iimax:%v kmin:%v kmax:%v)",
		qq.ii, qq.iimin, qq.iimax, qq.kmin, qq.kmax)
}

// I returns the cursor index, or -1 if the cursor is done (Next() returns
// false forever).
func (qq *Cursor) I() int {
	if qq.ii < qq.iimax {
		return qq.ix[qq.ii]
	}
	return -1
}

// Next advances the cursor if possible and returns true, false otherwise.
func (qq *Cursor) Next() bool {
	for qq.ii++; qq.ii < qq.iimax; qq.ii++ {
		if qq.ks[qq.ix[qq.ii]] > qq.kmax {
			qq.ii = qq.iimax + 1
			return false
		}
		if qq.r == image.ZR {
			return true
		} else if qq.ks[qq.ix[qq.ii]].Pt().In(qq.r) {
			// TODO implement BIGMIN; turns out that Key.Pt above is the long pole
			return true
		}
	}
	return false
}

func rotateRight(ns []int) {
	tmp := ns[len(ns)-1]
	copy(ns[1:], ns)
	ns[0] = tmp
}

type index struct {
	ix []int
	ks []Key
}

func (qi index) Len() int             { return len(qi.ix) }
func (qi index) Less(ii, jj int) bool { return qi.ks[qi.ix[ii]] < qi.ks[qi.ix[jj]] }
func (qi index) Swap(ii, jj int)      { qi.ix[ii], qi.ix[jj] = qi.ix[jj], qi.ix[ii] }

func (qi *index) alloc(i int, init Key) {
	for i >= len(qi.ix) {
		if i < cap(qi.ix) {
			j := len(qi.ix)
			qi.ix = qi.ix[:i+1]
			for ; j <= i; j++ {
				qi.ix[j] = j
			}
		} else {
			qi.ix = append(qi.ix, len(qi.ix))
		}
	}
	qi.ix[i] = i

	for i >= len(qi.ks) {
		if i < cap(qi.ks) {
			j := len(qi.ks)
			qi.ks = qi.ks[:i+1]
			for ; j <= i; j++ {
				qi.ks[j] = init
			}
		} else {
			qi.ks = append(qi.ks, init)
		}
	}
}

func (qi *index) search(k Key) int {
	return qi.narrow(0, len(qi.ix), k)
}

func (qi *index) narrow(ii, jj int, k Key) int {
	k |= keySet
	for ii < jj {
		h := int(uint(ii+jj) >> 1) // avoid overflow when computing h
		// ii ≤ h < jj
		if qi.ks[qi.ix[h]] < k {
			ii = h + 1 // preserves qi.ks[qi.ix[ii-1]] < k
		} else {
			jj = h // preserves qi.ks[qi.ix[jj]] >= k
		}
	}
	return ii
}

func (qi *index) run(ii int, k Key) int {
	for ii < len(qi.ix) && qi.ks[qi.ix[ii]] == k {
		ii++
	}
	return ii
}
