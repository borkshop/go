package ecs

import "fmt"

// Scope is a frame of reference for defining typed entities, supporting
// coordination between various stakeholders interested in entity type information.
//
// Scope is itself (an implicit) component manager of generational type
// information for every ID.
//
// It maintains a free list of prior used entity IDs that are no longer used
// (having Type 0). Any free entity IDs are re-used first before defining new
// IDs.
type Scope struct {
	typs []genType
	free []ID

	wats   []Watcher
	watAll []Type // wats[i] tracks Type.HasAll(watAll[i])
	watAny []Type // wats[i] tracks Type.HasAny(watAll[i])
}

// Watcher is a stakeholder in Entity type changes.
//
// A component data manager may be implemented as a Watcher that (de)allocates
// data when entity types come (and go). A natural choice for storage here is
// array data, either directly indexed using Entity.Seq(), or indirectly
// through an ArrayIndex.
//
// An entity processing system may be implemented as a Watcher that retains
// entity references, e.g. using one or more Entities collections. Its update
// logic can then be implemented directly on this collected data, rather than
// needing to iterate or query entity type information during update.
type Watcher interface {
	EntityCreated(e Entity, t Type)
	EntityDestroyed(e Entity, t Type)
}

// Watchers is a compound Watcher.
type Watchers []Watcher

// EntityCreated calls each watcher in order.
func (wats Watchers) EntityCreated(e Entity, t Type) {
	for i := 0; i < len(wats); i++ {
		wats[i].EntityCreated(e, t)
	}
}

// EntityDestroyed calls each watcher in reverse order.
func (wats Watchers) EntityDestroyed(e Entity, t Type) {
	for i := len(wats) - 1; i >= 0; i-- {
		wats[i].EntityDestroyed(e, t)
	}
}

// Len returns the number of defined entities (with non-zero type).
func (sc *Scope) Len() int {
	return len(sc.typs) - len(sc.free)
}

// ID returns the entity id for the given sequence number, useful when using
// direct indexing. Returns zero ID if the given sequence has no defined type.
func (sc *Scope) ID(seq int) ID {
	typ := sc.typs[seq]
	if typ.Type == 0 {
		return 0
	}
	return ID(seq) | (ID(typ.gen) << idBits)
}

// Clear destroys all entities defined within the scope.
func (sc *Scope) Clear() {
	for seq, typ := range sc.typs {
		if typ.Type != 0 {
			id := ID(seq) | (ID(typ.gen) << idBits)
			ent := Entity{sc, id}
			ent.setType(typ, uint64(seq), 0) // TODO optimize
		}
	}
}

// MustOwn panics unless the scope own the given entity.
// The name argument is used to provide a more useful panic message if non-empty.
func (sc *Scope) MustOwn(ent Entity, name string) {
	if ent.Scope != sc {
		if name == "" {
			panic("invalid entity")
		} else {
			panic(fmt.Sprintf("invalid %s entity", name))
		}
	}
}

// Watch changes in entity types, calling the given Watcher when all of the
// given bits are destroyed / created. If all is 0 then the Watcher is called
// when any type bits are destroyed/created.
//
// Watcher Create is called when all given bits have been added to an entities
// type; in other words, compound Create watching fires last.
//
// Conversely, Watcher Destroy is called when any of the given "all" bits is
// removed; in other words, compound Destroy watching fires early and often.
//
// The Watcher is passed any new/old type bits to Create/Destroy.
func (sc *Scope) Watch(all, any Type, wat Watcher) {
	sc.watAll = append(sc.watAll, all)
	sc.watAny = append(sc.watAny, any)
	sc.wats = append(sc.wats, wat)
}

// RemoveWatcher removes a watcher from any/all Types registered by Watch.
func (sc *Scope) RemoveWatcher(wat Watcher) {
	j := 0
	for i := 0; i < len(sc.wats); i++ {
		if sc.wats[i] == wat {
			continue
		}
		if i != j {
			sc.watAll[j] = sc.watAll[i]
			sc.watAny[j] = sc.watAny[i]
			sc.wats[j] = sc.wats[i]
		}
		j++
	}
	sc.watAll = sc.watAll[:j]
	sc.watAny = sc.watAny[:j]
	sc.wats = sc.wats[:j]
}

// Create a new entity with the given Type, returning a handle to it.
//
// Fires any Watcher's whose all criteria are fully satisfied by the new Type,
// and whose any criteria (if non-zero) are have at least one bit satisfied.
func (sc *Scope) Create(newType Type) (ent Entity) {
	if newType != 0 {
		ent = Entity{sc, sc.create()}
		typ, seq := ent.typ()
		if typ.Type != 0 {
			panic(fmt.Sprintf("refusing to reuse an entity with non-zero type: %v", typ))
		}
		sc.typs[seq].Type = newType
		ent.dispatchCreate(newType, newType)
	}
	return ent
}

// CreateN creates N entities with the given type, returning a collection of
// their IDs.
func (sc *Scope) CreateN(newType Type, n int) (es Entities) {
	es.Scope = sc
	for i := 0; i < n; i++ {
		es.IDs = append(es.IDs, sc.Create(newType).ID)
	}
	return es
}

// Entity resolves an ID to an Entity within the scope.
// Returns zero Entity value for zero id.
// Panics if the ID is invalid.
func (sc *Scope) Entity(id ID) Entity {
	if id == 0 {
		return ZE
	}
	ent := Entity{sc, id}
	ent.typ() // check gen
	return ent
}

func (sc *Scope) create() ID {
	if i := len(sc.free) - 1; i >= 0 {
		id := sc.free[i]
		sc.free = sc.free[:i]
		return id
	}
	sc.typs = append(sc.typs, genType{gen: 1})
	return ID(len(sc.typs) - 1).setgen(1)
}

// TODO maybe move type Entity definition in from core.go

func (ent Entity) dispatchCreate(newType, createdType Type) {
	for i := 0; i < len(ent.Scope.watAll); i++ {
		if all := ent.Scope.watAll[i]; all == 0 || (newType.HasAll(all) && createdType.HasAny(all)) {
			if any := ent.Scope.watAny[i]; any == 0 || createdType.HasAny(any) {
				ent.Scope.wats[i].EntityCreated(ent, createdType)
			}
		}
	}
}

func (ent Entity) dispatchDestroy(newType, destroyedType Type) {
	for i := 0; i < len(ent.Scope.watAll); i++ {
		if all := ent.Scope.watAll[i]; all == 0 || (!newType.HasAll(all) && (newType | destroyedType).HasAll(all)) {
			if any := ent.Scope.watAny[i]; any == 0 || destroyedType.HasAny(any) {
				ent.Scope.wats[i].EntityDestroyed(ent, destroyedType)
			}
		}
	}
}
