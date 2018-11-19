package eps

import (
	"math"
	"sort"

	"deathroom/internal/ecs"
	"deathroom/internal/point"
)

// TODO: support movement on top of or within an EPS:
// - an ecs.Relation on positioned things:
//   - intent: direction & magnitude? jumping?
//   - outcome: collision (only if solid)? pre-compute "what's here"?

// EPS is an Entity Positioning System; (technically it's not an ecs.System, it
// just has a reference to an ecs.Core).
type EPS struct {
	core *ecs.Core
	t    ecs.ComponentType

	frozen bool
	data
}

type data struct {
	pos   []pos
	posIX []int
}

type pos struct {
	def bool
	point.Point
	key uint64
}

// Init ialize the EPS wrt a given core and component type that
// represents "has a position".
func (eps *EPS) Init(core *ecs.Core, t ecs.ComponentType) {
	eps.core = core
	eps.t = t
	eps.core.RegisterAllocator(eps.t, eps.alloc)
	eps.core.RegisterCreator(eps.t, eps.create)
	eps.core.RegisterDestroyer(eps.t, eps.destroy)

	eps.pos = []pos{pos{}}
	eps.posIX = []int{-1}
}

// Get the position of an entity; the bool argument is true only if
// the entity actually has a position.
func (eps *EPS) Get(ent ecs.Entity) (point.Point, bool) {
	if ent == ecs.NilEntity {
		return point.Zero, false
	}
	id := eps.core.Deref(ent)
	return eps.pos[id].Point, eps.pos[id].def
}

// Set the position of an entity, adding the eps's component if
// necessary.
func (eps *EPS) Set(ent ecs.Entity, pt point.Point) {
	id := eps.core.Deref(ent)
	eps.frozen = true
	if !eps.pos[id].def {
		ent.Add(eps.t)
	}
	eps.pos[id].def = true
	eps.pos[id].Point = pt
	eps.frozen = false
	eps.pos[id].key = zorderKey(eps.pos[id].Point)
	sort.Sort(eps.data) // TODO: worth a fix-one algorithm?
}

// At returns a slice of entities at a given point.
func (eps *EPS) At(pt point.Point) (ents []ecs.Entity) {
	k := zorderKey(pt)
	i, m := eps.data.searchRun(k)
	if m > 0 {
		ents = make([]ecs.Entity, m)
		for j := 0; j < m; i, j = i+1, j+1 {
			xi := eps.posIX[i+1]
			ents[j] = eps.core.Ref(ecs.EntityID(xi))
		}
	}
	return ents
}

// TODO: NN queries, range queries, etc
// func (eps *EPS) Near(pt point.Point, d uint) []ecs.Entity
// func (eps *EPS) Within(box point.Box) []ecs.Entity

func (eps *EPS) alloc(id ecs.EntityID, t ecs.ComponentType) {
	i := len(eps.pos)
	eps.pos = append(eps.pos, pos{})
	eps.posIX = append(eps.posIX, i)
}

func (eps *EPS) create(id ecs.EntityID, t ecs.ComponentType) {
	eps.pos[id].def = true
	eps.pos[id].key = zorderKey(eps.pos[id].Point)
	if !eps.frozen {
		sort.Sort(eps.data) // TODO: worth a fix-one algorithm?
	}
}

func (eps *EPS) destroy(id ecs.EntityID, t ecs.ComponentType) {
	eps.pos[id] = pos{}
	if !eps.frozen {
		sort.Sort(eps.data) // TODO: worth a fix-one algorithm?
	}
}

// TODO: evaluate hilbert instead of z-order
func zorderKey(pt point.Point) (z uint64) {
	// TODO: evaluate a table ala
	// https://graphics.stanford.edu/~seander/bithacks.html#InterleaveTableObvious
	x, y := truncInt32(pt.X), truncInt32(pt.Y)
	for i := uint(0); i < 32; i++ {
		z |= (x&(1<<i))<<i | (y&(1<<i))<<(i+1)
	}
	return z
}

func truncInt32(n int) uint64 {
	if n < math.MinInt32 {
		return 0
	}
	if n > math.MaxInt32 {
		return math.MaxUint32
	}
	return uint64(uint32(n - math.MinInt32))

}

func (d data) Len() int { return len(d.posIX) - 1 }

func (d data) Less(i, j int) bool {
	xi, xj := d.posIX[i+1], d.posIX[j+1]
	if !d.pos[xi].def {
		return true
	} else if !d.pos[xj].def {
		return false
	}
	return d.pos[xi].key < d.pos[xj].key
}

func (d data) Swap(i, j int) {
	i++
	j++
	d.posIX[i], d.posIX[j] = d.posIX[j], d.posIX[i]
}

func (d data) search(key uint64) int {
	return sort.Search(d.Len(), func(i int) bool {
		xi := d.posIX[i+1]
		return d.pos[xi].def && d.pos[xi].key >= key
	})
}

func (d data) searchRun(key uint64) (i, m int) {
	i = d.search(key)
	for j, n := i, d.Len(); j < n; j++ {
		if xi := d.posIX[j+1]; !d.pos[xi].def || d.pos[xi].key != key {
			break
		}
		m++
	}
	return i, m
}
