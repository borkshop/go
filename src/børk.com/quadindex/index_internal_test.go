package quadindex

import (
	"fmt"
	"image"
	"math/rand"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndex_reindex(t *testing.T) {
	// TODO test adding things in generations
	// TODO test incrementally moving things
	for _, spec := range []indexUpdateSpec{
		{nn: 10, nmag: 100, noff: 50, nup: 1},
		{nn: 100, nmag: 100, noff: 50, nup: 10},
		{nn: 1000, nmag: 100, noff: 50, nup: 100},
	} {
		for dnup := spec.nup; spec.nup < 8*spec.nn/10; spec.nup += dnup {
			if !t.Run(fmt.Sprintf("update %v / %v points", spec.nup, spec.nn), spec.run) {
				break
			}
		}
	}
}

func BenchmarkIndex_reindex(b *testing.B) {
	for n := 1000; n < 10000; n += 1000 {
		spec := indexUpdateSpec{nn: n, nmag: 100, noff: 50, nup: n / 10}
		for dnup := spec.nup; spec.nup < 8*spec.nn/10; spec.nup += dnup {
			b.Run(fmt.Sprintf("N=%v U=%v", spec.nn, spec.nup), func(b *testing.B) {
				spec.gen()
				spec.qi.resort()
				qiSorter := sort.Interface(spec.qi)
				require.True(b, sort.IsSorted(qiSorter), "resort() must work")
				ids := make([]int, 0, spec.nup)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					for _, id := range selectIDs(spec.rng, spec.maxID, ids[:0]) {
						spec.qi.Update(id, spec.pg.gen(spec.rng))
					}
					spec.qi.reindex()
					require.True(b, sort.IsSorted(qiSorter), "reindex() must work")
				}
			})
		}
	}
}

type indexUpdateSpec struct {
	nn   int // number of initial points
	nmag int // max magnitude of variation
	noff int // offset for rand.Intn(nmag)
	nup  int // number to update after fill

	rng   *rand.Rand
	pg    pointGen
	qi    Index
	maxID int
}

func (spec *indexUpdateSpec) init() {
	spec.rng = rand.New(rand.NewSource(0))
	spec.pg = pointGen{spec.nmag, spec.noff}
}

func (spec *indexUpdateSpec) gen() {
	spec.init()
	for i := 0; i < spec.nn; i++ {
		spec.qi.Update(spec.maxID, spec.pg.gen(spec.rng))
		spec.maxID++
	}
}

func (spec indexUpdateSpec) run(t *testing.T) {
	spec.gen()
	require.False(t, sort.IsSorted(spec.qi), "must be initially unsorted")
	spec.qi.resort()
	require.True(t, sort.IsSorted(spec.qi), "resort() must work")
	for _, id := range selectIDs(spec.rng, spec.maxID, make([]int, 0, spec.nup)) {
		spec.qi.Update(id, spec.pg.gen(spec.rng))
	}
	require.False(t, sort.IsSorted(spec.qi), "should be sorted after update")
	spec.qi.reindex()
	assert.True(t, sort.IsSorted(spec.qi), "reindex() should work")
}

func selectIDs(rng *rand.Rand, maxID int, ids []int) []int {
	for id := 0; id < maxID; id++ {
		if len(ids) < cap(ids) {
			ids = append(ids, id)
		} else if i := rng.Intn(id + 1); i < len(ids) {
			ids[i] = id
		}
	}
	return ids
}

type pointGen struct {
	max int
	off int
}

func (pg pointGen) gen(rng *rand.Rand) image.Point {
	return image.Pt(
		rng.Intn(pg.max)-pg.off,
		rng.Intn(pg.max)-pg.off,
	)
}
