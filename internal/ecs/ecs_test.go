package ecs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jcorbin/execs/internal/ecs"
)

const (
	scData ecs.ComponentType = 1 << iota
	scD2
)

type stuff struct {
	ecs.Core

	d1 []int
	d2 [][]int
}

func newStuff() *stuff {
	s := &stuff{
		d1: []int{0},
		d2: [][]int{nil},
	}
	s.RegisterAllocator(scData, s.allocData)
	s.RegisterCreator(scD2, s.createD2)
	s.RegisterDestroyer(scD2, s.destroyD2)
	return s
}

func (s *stuff) allocData(id ecs.EntityID, t ecs.ComponentType) {
	s.d1 = append(s.d1, 0)
	s.d2 = append(s.d2, nil)
}

func (s *stuff) createD2(id ecs.EntityID, t ecs.ComponentType) {
	if s.d2[id] == nil {
		s.d2[id] = make([]int, 0, 5)
	}
}

func (s *stuff) destroyD2(id ecs.EntityID, t ecs.ComponentType) {
	s.d2[id] = s.d2[id][:0]
}

func TestBasics(t *testing.T) {
	s := newStuff()
	assert.True(t, s.Empty())

	e1 := s.AddEntity(scData)
	assert.False(t, s.Empty())

	assert.Nil(t, s.d2[e1.ID()])
	e1.Add(scD2)
	assert.NotNil(t, s.d2[e1.ID()])
	assert.Equal(t, 0, len(s.d2[e1.ID()]))

	s.d2[e1.ID()] = append(s.d2[e1.ID()], 3, 1, 4)
	assert.Equal(t, 3, len(s.d2[e1.ID()]))

	e2 := s.AddEntity(scData | scD2)
	assert.NotNil(t, s.d2[e2.ID()])
	assert.Equal(t, 0, len(s.d2[e2.ID()]))

	e1.Delete(scD2)
	assert.Equal(t, 0, len(s.d2[e1.ID()]))
	assert.NotNil(t, s.d2[e1.ID()])

	e1.Destroy()

	e3 := s.AddEntity(scData | scD2)
	assert.Equal(t, e1.ID(), e3.ID())

	assert.False(t, s.Empty())
	s.Clear()
	assert.True(t, s.Empty())
}

func TestIter_empty(t *testing.T) {
	s := newStuff()
	it := s.Iter(ecs.AllClause)
	assert.Equal(t, 0, it.Count())

	assert.False(t, it.Next())
	assert.Equal(t, ecs.NilEntity, it.Entity())
	assert.Equal(t, ecs.EntityID(0), it.ID())
	assert.Equal(t, ecs.NoType, it.Type())
}

func TestIter_one(t *testing.T) {
	s := newStuff()

	s1 := s.AddEntity(scData)
	s.d1[s1.ID()] = 3

	it := s.Iter(ecs.AllClause)
	assert.Equal(t, 1, it.Count())

	assert.True(t, it.Next())
	assert.Equal(t, s1, it.Entity())
	assert.Equal(t, ecs.EntityID(1), it.ID())
	assert.Equal(t, scData, it.Type())

	assert.False(t, it.Next())
	assert.Equal(t, ecs.NilEntity, it.Entity())
	assert.Equal(t, ecs.EntityID(0), it.ID())
	assert.Equal(t, ecs.NoType, it.Type())
}

func TestIter_two(t *testing.T) {
	s := newStuff()

	e1 := s.AddEntity(scData)
	s.d1[e1.ID()] = 3
	e2 := s.AddEntity(scData | scD2)
	s.d1[e2.ID()] = 4
	s.d2[e2.ID()] = append(s.d2[e2.ID()], 2, 2, 3, 5, 8)

	it := s.Iter(ecs.AllClause)
	assert.Equal(t, 2, it.Count())

	// iterate all 3
	assert.True(t, it.Next())
	assert.Equal(t, e1, it.Entity())
	assert.Equal(t, ecs.EntityID(1), it.ID())
	assert.Equal(t, scData, it.Type())

	assert.True(t, it.Next())
	assert.Equal(t, e2, it.Entity())
	assert.Equal(t, ecs.EntityID(2), it.ID())
	assert.Equal(t, scData|scD2, it.Type())

	assert.False(t, it.Next())
	assert.Equal(t, ecs.NilEntity, it.Entity())
	assert.Equal(t, ecs.EntityID(0), it.ID())
	assert.Equal(t, ecs.NoType, it.Type())

	// filtering
	it = s.Iter(ecs.All(scD2))
	assert.Equal(t, 1, it.Count())

	assert.True(t, it.Next())
	assert.Equal(t, e2, it.Entity())
	assert.Equal(t, ecs.EntityID(2), it.ID())
	assert.Equal(t, scData|scD2, it.Type())

	assert.False(t, it.Next())
	assert.Equal(t, ecs.NilEntity, it.Entity())
	assert.Equal(t, ecs.EntityID(0), it.ID())
	assert.Equal(t, ecs.NoType, it.Type())
}

func setupRelTest(aFlags, bFlags ecs.RelationFlags) (a, b *stuff, rel *ecs.Relation) {
	a = newStuff()
	a1 := a.AddEntity(scData)
	a2 := a.AddEntity(scData)
	a3 := a.AddEntity(scData)
	a4 := a.AddEntity(scData)
	a5 := a.AddEntity(scData)
	a6 := a.AddEntity(scData)
	a7 := a.AddEntity(scData)
	_ = a.AddEntity(scData) // a8

	b = newStuff()
	b1 := b.AddEntity(scData)
	b2 := b.AddEntity(scData)
	b3 := b.AddEntity(scData)
	b4 := b.AddEntity(scData)
	b5 := b.AddEntity(scData)
	b6 := b.AddEntity(scData)
	b7 := b.AddEntity(scData)
	_ = b.AddEntity(scData) // b8

	rel = ecs.NewRelation(&a.Core, aFlags, &b.Core, bFlags)

	rel.InsertMany(func(insert func(r ecs.RelationType, a ecs.Entity, b ecs.Entity) ecs.Entity) {

		insert(0, a1, b2)
		insert(0, a1, b3)
		insert(0, a2, b4)
		insert(0, a2, b5)
		insert(0, a3, b6)
		insert(0, a3, b7)

		insert(0, a2, b1)
		insert(0, a3, b1)
		insert(0, a4, b2)
		insert(0, a5, b2)
		insert(0, a6, b3)
		insert(0, a7, b3)

	})

	return a, b, rel
}

type testCases []testCase
type testCase struct {
	name string
	run  func(t *testing.T)
}

func (tcs testCases) run(t *testing.T) {
	for _, tc := range tcs {
		t.Run(tc.name, tc.run)
	}
}

func TestRelation_destruction(t *testing.T) {
	testCases{
		{"clear A", func(t *testing.T) {
			a, b, r := setupRelTest(0, 0)
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.False(t, r.Empty())
			a.Clear()
			assert.True(t, a.Empty())
			assert.False(t, b.Empty())
			assert.True(t, r.Empty())
		}},

		{"clear B", func(t *testing.T) {
			a, b, r := setupRelTest(0, 0)
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.False(t, r.Empty())
			b.Clear()
			assert.False(t, a.Empty())
			assert.True(t, b.Empty())
			assert.True(t, r.Empty())
		}},

		{"clear rels", func(t *testing.T) {
			a, b, r := setupRelTest(0, 0)
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.False(t, r.Empty())
			r.Clear()
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.True(t, r.Empty())
		}},

		{"A cascades", func(t *testing.T) {
			a, b, r := setupRelTest(ecs.RelationCascadeDestroy, 0)
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.False(t, r.Empty())
			assert.Equal(t, 8, a.Len())
			assert.Equal(t, 8, b.Len())

			b.Ref(1).Destroy()
			assert.Equal(t, 6, a.Len())
			assert.Equal(t, ecs.NoType, a.Ref(2).Type())
			assert.Equal(t, ecs.NoType, a.Ref(3).Type())

			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.False(t, r.Empty())

			b.Clear()
			assert.False(t, a.Empty())
			assert.True(t, b.Empty())
			assert.True(t, r.Empty())
			assert.Equal(t, 1, a.Len())
			assert.Equal(t, 0, b.Len())

			a, b, r = setupRelTest(ecs.RelationCascadeDestroy, 0)
			r.Clear()
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.True(t, r.Empty())
			assert.Equal(t, 1, a.Len())
			assert.Equal(t, 8, b.Len())
		}},

		{"B cascades", func(t *testing.T) {
			a, b, r := setupRelTest(0, ecs.RelationCascadeDestroy)
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.False(t, r.Empty())
			assert.Equal(t, 8, a.Len())
			assert.Equal(t, 8, b.Len())

			a.Ref(1).Destroy()
			assert.Equal(t, 6, b.Len())
			assert.Equal(t, ecs.NoType, b.Ref(2).Type())
			assert.Equal(t, ecs.NoType, b.Ref(3).Type())

			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.False(t, r.Empty())

			a.Clear()
			assert.True(t, a.Empty())
			assert.False(t, b.Empty())
			assert.True(t, r.Empty())
			assert.Equal(t, 0, a.Len())
			assert.Equal(t, 1, b.Len())

			a, b, r = setupRelTest(0, ecs.RelationCascadeDestroy)
			r.Clear()
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.True(t, r.Empty())
			assert.Equal(t, 8, a.Len())
			assert.Equal(t, 1, b.Len())
		}},

		{"A & B cascade", func(t *testing.T) {
			a, b, r := setupRelTest(ecs.RelationCascadeDestroy, ecs.RelationCascadeDestroy)
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.False(t, r.Empty())
			assert.Equal(t, 8, a.Len())
			assert.Equal(t, 8, b.Len())

			a.Ref(1).Destroy()
			assert.Equal(t, 3, a.Len())
			assert.Equal(t, 6, b.Len())
			assert.Equal(t, ecs.NoType, b.Ref(2).Type())
			assert.Equal(t, ecs.NoType, b.Ref(3).Type())

			b.Ref(1).Destroy()
			assert.Equal(t, 1, a.Len())
			assert.Equal(t, 1, b.Len())
			assert.Equal(t, ecs.NoType, a.Ref(2).Type())
			assert.Equal(t, ecs.NoType, a.Ref(3).Type())

			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.True(t, r.Empty())

			a.Clear()
			assert.True(t, a.Empty())
			assert.False(t, b.Empty())
			assert.True(t, r.Empty())
			assert.Equal(t, 0, a.Len())
			assert.Equal(t, 1, b.Len())

			a, b, r = setupRelTest(ecs.RelationCascadeDestroy, ecs.RelationCascadeDestroy)
			r.Clear()
			assert.False(t, a.Empty())
			assert.False(t, b.Empty())
			assert.True(t, r.Empty())
			assert.Equal(t, 1, a.Len())
			assert.Equal(t, 1, b.Len())
		}},
	}.run(t)
}

func setupGraphTest(flag ecs.RelationFlags) (*stuff, *ecs.Graph) {
	s := newStuff()
	s1 := s.AddEntity(scData)
	s2 := s.AddEntity(scData)
	s3 := s.AddEntity(scData)
	s4 := s.AddEntity(scData)
	s5 := s.AddEntity(scData)
	s6 := s.AddEntity(scData)
	s7 := s.AddEntity(scData)

	G := ecs.NewGraph(&s.Core, 0)
	G.InsertMany(func(insert func(r ecs.RelationType, a ecs.Entity, b ecs.Entity) ecs.Entity) {
		insert(0, s1, s2)
		insert(0, s1, s3)
		insert(0, s2, s4)
		insert(0, s2, s5)
		insert(0, s3, s6)
		insert(0, s3, s7)
	})

	return s, G
}

func TestGraph_Roots(t *testing.T) {
	s, G := setupGraphTest(0)
	roots := G.Roots(ecs.AllClause, nil)
	assert.Equal(t, 1, len(roots))
	assert.Equal(t, s.Ref(1), roots[0])
}

func gtids(gt ecs.GraphTraverser) []ecs.EntityID {
	var ids []ecs.EntityID
	for gt.Traverse() {
		ids = append(ids, gt.Node().ID())
	}
	return ids
}

func TestGraph_Traverse(t *testing.T) {
	testCases{

		{"DFS", func(t *testing.T) {
			_, G := setupGraphTest(0)
			gt := G.Traverse(ecs.AllClause, ecs.TraverseDFS)
			gt.Init()
			assert.Equal(t,
				[]ecs.EntityID{1, 2, 4, 5, 3, 6, 7},
				gtids(gt))
		}},

		{"CoDFS", func(t *testing.T) {
			_, G := setupGraphTest(0)
			gt := G.Traverse(ecs.AllClause, ecs.TraverseCoDFS)
			for _, ids := range [][]ecs.EntityID{
				{4, 2, 1},
				{5, 2, 1},
				{6, 3, 1},
				{7, 3, 1},
			} {
				gt.Init(ids[0])
				assert.Equal(t, ids, gtids(gt))
			}
		}},
	}.run(t)
}
