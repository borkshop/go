package ecs_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	. "børk.no/ecs"
)

type testData struct {
	Scope
	ai  ArrayIndex
	rel EntityRelation
}

const testEntType Type = 1

type testDataStep func(t *testing.T, td *testData)

func (td *testData) run(t *testing.T, steps []testDataStep) {
	if td.ai.Scope != nil {
		panic("re-use of testData")
	}
	td.ai.Init(&td.Scope)
	td.rel.Init(&td.Scope, nil)
	td.Scope.Watch(testEntType, 0, &td.ai)
	for i, f := range steps {
		t.Run(fmt.Sprintf("step %v", i), func(t *testing.T) {
			f(t, td)
		})
	}
}

func TestEntityRelation(t *testing.T) {
	for _, scenario := range [][]testDataStep{
		// simply graph relation scenario
		{
			// create test entities
			func(t *testing.T, td *testData) {
				td.Scope.CreateN(testEntType, 8)
				assert.NotEqual(t, ID(0), td.ai.ID(0), "expected non-zero id 0")
				assert.NotEqual(t, ID(0), td.ai.ID(1), "expected non-zero id 1")
				assert.NotEqual(t, ID(0), td.ai.ID(2), "expected non-zero id 2")
				assert.NotEqual(t, ID(0), td.ai.ID(3), "expected non-zero id 3")
				assert.NotEqual(t, ID(0), td.ai.ID(4), "expected non-zero id 4")
				assert.NotEqual(t, ID(0), td.ai.ID(5), "expected non-zero id 5")
				assert.NotEqual(t, ID(0), td.ai.ID(6), "expected non-zero id 6")
				assert.NotEqual(t, ID(0), td.ai.ID(7), "expected non-zero id 7")
				td.expectARelations(t, [][]ID{
					{td.ai.ID(0)},
					{td.ai.ID(1)},
					{td.ai.ID(2)},
					{td.ai.ID(3)},
					{td.ai.ID(4)},
					{td.ai.ID(5)},
					{td.ai.ID(6)},
					{td.ai.ID(7)},
				})
			},
			// build a simple complete depth-3 binary tree (on 7 of those entities)
			func(t *testing.T, td *testData) {
				td.rel.InsertMany(0, td.ai.ID(0), td.ai.ID(1), td.ai.ID(2))
				td.rel.InsertMany(0, td.ai.ID(1), td.ai.ID(3), td.ai.ID(4))
				td.rel.Insert(0, td.ai.ID(2), td.ai.ID(5))
				td.rel.Insert(0, td.ai.ID(2), td.ai.ID(6))
				td.expectARelations(t, [][]ID{
					{td.ai.ID(0), td.ai.ID(1), td.ai.ID(2)},
					{td.ai.ID(1), td.ai.ID(3), td.ai.ID(4)},
					{td.ai.ID(2), td.ai.ID(5), td.ai.ID(6)},
					{td.ai.ID(3)},
					{td.ai.ID(4)},
					{td.ai.ID(5)},
					{td.ai.ID(6)},
					{td.ai.ID(7)},
				})
			},
			// delete the left sub-tree
			func(t *testing.T, td *testData) {
				td.rel.DeleteA(td.ai.ID(1))
				td.expectARelations(t, [][]ID{
					{td.ai.ID(0), td.ai.ID(1), td.ai.ID(2)},
					{td.ai.ID(1)},
					{td.ai.ID(2), td.ai.ID(5), td.ai.ID(6)},
					{td.ai.ID(3)},
					{td.ai.ID(4)},
					{td.ai.ID(5)},
					{td.ai.ID(6)},
					{td.ai.ID(7)},
				})
			},
		},
	} {
		var td testData
		td.run(t, scenario)
	}
}

func (td *testData) expectARelations(t *testing.T, rels [][]ID) {
	for _, rel := range rels {
		var expected []ID
		switch len(rel) {
		case 0:
			panic("bogus test expectation data")
		case 1:
		default:
			expected = rel[1:]
		}
		ents := td.rel.Bs(td.rel.LookupA(rel[0]), nil)
		assert.Equal(t, expected, ents.IDs, "expected id:%v related IDs", rel[0])
	}
}
