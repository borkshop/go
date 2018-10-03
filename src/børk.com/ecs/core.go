package ecs

import "fmt"

// ID identifies an individual entity under some Scope.
// The high 8-bits of an ID value are its generation number, which is useful
// for things like be used for best-effort use-after-free detection.
type ID uint64

const (
	idBits       = 56
	idGenMask ID = 0xff00000000000000 // 8-bit generation
	idSeqMask ID = 0x00ffffffffffffff // 56-bit id
)

// String representation of the ID, clearly shows the sequence and generation
// numbers.
func (id ID) String() string {
	gen, seq := id>>idBits, id&idSeqMask
	if gen == 0 {
		if seq != 0 {
			return fmt.Sprintf("INVALID_ZeroID(seq:%d)", gen)
		}
		return "ZeroID"
	}
	return fmt.Sprintf("%d(gen:%d)", seq, gen)
}

// genseq returns the generation and sequence numbers.
func (id ID) genseq() (uint8, uint64) {
	gen, seq := id>>idBits, id&idSeqMask
	if gen == 0 {
		panic("invalid use of gen-0 ID")
	}
	return uint8(gen), uint64(seq)
}

// setgen returns a copy of the ID with the 8 generations bits replaced with
// the given ones.
func (id ID) setgen(gen uint8) ID {
	seq := id & idSeqMask
	if gen == 0 {
		panic("invalid use of gen-0 ID")
	}
	return seq | (ID(gen) << idBits)
}

// Type describe entity component composition: each entity in a Scope has a
// Type value describing which components it currently has; entities only exist
// if they have a non-zero type; each component within a scope must be
// registered with a distinct type bit.
type Type uint64

// genType is a generationally-numbered Type value. The generation number is
// useful for things like best-effort use-after-free detection.
type genType struct {
	gen  uint8
	Type Type
}

// HasAll returns true only if the receiver type has all of the argument type
// bits set.
func (typ Type) HasAll(t Type) bool { return typ&t == t }

// HasAny returns true only if the receiver type has any of the argument type
// bit set.
func (typ Type) HasAny(t Type) bool { return typ&t != 0 }

func (typ Type) String() string {
	return fmt.Sprintf("T+%016X", uint64(typ))
}

// Entity is a handle within a Scope's ID space.
type Entity struct {
	Scope *Scope
	ID    ID
}

func (ent Entity) String() string {
	return fmt.Sprintf("entity(%p %v %v)", ent.Scope, ent.ID, ent.Type())
}

// EntityCreatedFunc is a convenience for a creation-only watcher.
type EntityCreatedFunc func(Entity, Type)

// EntityDestroyedFunc is a convenience for a destruction-only watcher.
type EntityDestroyedFunc func(Entity, Type)

// EntityCreated calls the aliased function.
func (f EntityCreatedFunc) EntityCreated(e Entity, t Type) { f(e, t) }

// EntityDestroyed is a no-op.
func (f EntityCreatedFunc) EntityDestroyed(e Entity, t Type) {}

// EntityCreated is a no-op.
func (f EntityDestroyedFunc) EntityCreated(e Entity, t Type) {}

// EntityDestroyed calls the aliased function.
func (f EntityDestroyedFunc) EntityDestroyed(e Entity, t Type) { f(e, t) }

// Destroy the Entity; a convenience for SetType(0).
func (ent Entity) Destroy() bool {
	return ent.SetType(0)
}

func (ent Entity) typ() (genType, uint64) {
	gen, seq := ent.ID.genseq()
	if gen == 0 {
		panic("invalid use of gen-0 ID")
	}
	typ := ent.Scope.typs[seq]
	if gen != typ.gen {
		panic(fmt.Sprintf("mis-use of entity of generation %v, expected %v", gen, typ.gen))
	}
	return typ, seq
}

// Type returns the type of the entity. Panics if Entity's generation is out of
// sync with Scope's.
func (ent Entity) Type() Type {
	typ, _ := ent.typ()
	return typ.Type
}

// Seq returns the Entity's sequence number, validating it and the generation
// number. Component data managers should use this to map internal data
// (directly, indirectly, or otherwise) rather than the raw ID itself.
func (ent Entity) Seq() uint64 {
	_, seq := ent.typ()
	return seq
}

// SetType updates the type of the entity, calling any requisite watchers.
// Panics if Entity's generation is out of sync with Scope's.
//
// Setting the type to 0 will completely destroy the entity, marking its ID for
// future reuse. In a best-effort to prevent use-after-free bugs, the ID's
// generation number is incremented before returning it to the free list,
// invalidating any future use of the prior generation's handle.
func (ent Entity) SetType(newType Type) bool {
	if ent.Scope == nil || ent.ID == 0 {
		panic("invalid entity handle")
	}
	priorTyp, seq := ent.typ()
	return ent.setType(priorTyp, seq, newType)
}

// AddType adds type bits to the entity.
// It's a more cohesive version of ent.SetType(ent.Type() | typ).
func (ent Entity) AddType(typ Type) bool {
	if ent.Scope == nil || ent.ID == 0 {
		panic("invalid entity handle")
	}
	priorTyp, seq := ent.typ()
	return ent.setType(priorTyp, seq, priorTyp.Type|typ)
}

// DeleteType adds type bits to the entity.
// It's a more cohesive version of ent.SetType(ent.Type() & ^typ).
func (ent Entity) DeleteType(typ Type) bool {
	if ent.Scope == nil || ent.ID == 0 {
		panic("invalid entity handle")
	}
	priorTyp, seq := ent.typ()
	return ent.setType(priorTyp, seq, priorTyp.Type&^typ)
}

func (ent Entity) setType(priorTyp genType, seq uint64, newType Type) bool {
	typeChange := priorTyp.Type ^ newType
	if typeChange == 0 {
		return false
	}

	ent.Scope.typs[seq].Type = newType

	if destroyTyp := priorTyp.Type & typeChange; destroyTyp != 0 {
		ent.dispatchDestroy(newType, destroyTyp)
	}

	if newType == 0 {
		gen := priorTyp.gen + 1
		if gen == 0 {
			gen = 1
		}
		ent.Scope.typs[seq].gen = gen // further reuse of this Entity handle should panic
		ent.Scope.free = append(ent.Scope.free, ent.ID.setgen(gen))
		return true
	}

	if createTyp := newType & typeChange; createTyp != 0 {
		ent.dispatchCreate(newType, createTyp)
	}

	return true
}

// ZE is the zero entity
var ZE Entity

// Entities is a collection of entity ids from the same scope.
type Entities struct {
	Scope *Scope
	IDs   []ID
}

// ID returns the i-th entity id.
func (es Entities) ID(i int) ID { return es.IDs[i] }

// Entity returns an entity handle for the i-th ID.
func (es Entities) Entity(i int) Entity { return Entity{es.Scope, es.IDs[i]} }

// FilterAll filters the collection of entities to ones whose type has all of
// the given type bits.
func (es Entities) FilterAll(t Type) {
	i := 0
	for j := 0; j < len(es.IDs); j++ {
		if es.Entity(j).Type().HasAll(t) {
			es.IDs[i] = es.IDs[j]
			i++
		}
	}
	es.IDs = es.IDs[:i]
}

// FilterAny filters the collection of entities to ones whose type has any of
// the given type bits.
func (es Entities) FilterAny(t Type) {
	i := 0
	for j := 0; j < len(es.IDs); j++ {
		if es.Entity(j).Type().HasAny(t) {
			es.IDs[i] = es.IDs[j]
			i++
		}
	}
	es.IDs = es.IDs[:i]
}

// Ent is a convenience constructor for an entity handle.
func Ent(s *Scope, id ID) Entity { return Entity{s, id} }

// Ents is a convenience constructor for a collection of entity handles.
func Ents(s *Scope, ids []ID) Entities { return Entities{s, ids} }

func withoutID(ids []ID, id ID) []ID {
	i := 0
	for j := 0; j < len(ids); j++ {
		if ids[j] == id {
			continue
		}
		ids[i] = ids[j]
		i++
	}
	return ids[:i]
}
