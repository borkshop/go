package ecs

// EntityRelation relates entities within two (maybe the same) Scope.
// Relations themselves are entities within the Scope of an EntityRelation.
// Related entity IDs are component data within an EntityRelation.
// Domain specific data may be added to a relation by embedding EntityRelation
// and adding component data around it.
type EntityRelation struct {
	a, b *Scope

	Scope

	// uses direct indexing into
	aid []ID
	bid []ID

	aindex map[ID][]ID // a ID -> relation IDs
	bindex map[ID][]ID // b ID -> relation IDs

	tmp []ID
}

// EntityRelation type constants.
const (
	TypeEntityRelation Type = 1 << (63 - iota)
)

// A returns the A entity for the given relation.
func (er *EntityRelation) A(rel Entity) Entity { return er.a.Entity(er.aid[int(rel.Seq())]) }

// B returns the B entity for the given relation.
func (er *EntityRelation) B(rel Entity) Entity { return er.b.Entity(er.bid[int(rel.Seq())]) }

// As returns a collection of A entities for the given relations; re-uses any
// prior []ID capacity given.
func (er *EntityRelation) As(rels Entities, ids []ID) Entities {
	ids = allocRelIDs(rels, ids)
	for i := range rels.IDs {
		ids[i] = er.aid[int(rels.Entity(i).Seq())]
	}
	return Entities{er.a, ids}
}

// Bs returns a collection of B entities for the given relations; re-uses any
// prior capacity in the given bs collection.
func (er *EntityRelation) Bs(rels Entities, ids []ID) Entities {
	ids = allocRelIDs(rels, ids)
	for i := range rels.IDs {
		ids[i] = er.bid[int(rels.Entity(i).Seq())]
	}
	return Entities{er.b, ids}
}

func allocRelIDs(rels Entities, ids []ID) []ID {
	if cap(ids) < len(rels.IDs) {
		n := len(rels.IDs)
		if n < 1024 {
			n *= 2
		} else {
			n = 5 * n / 4
		}
		ids = make([]ID, 0, n)
	}
	return ids[:len(rels.IDs)]
}

// Init ialize an EntityRelation between the given two scopes.
// If B is nil or equal to A, the outcome is the same: an auto-relation between
// entities within the same scope (i.e. a graph).
func (er *EntityRelation) Init(A, B *Scope) {
	if er.a != nil || er.b != nil {
		panic("invalid EntityRelation re-initialization")
	}
	if A == nil {
		panic("must provide an A relation ")
	}

	er.Scope.Watch(TypeEntityRelation, 0, er)

	er.a = A
	er.a.Watch(0, 0, EntityDestroyedFunc(er.onADestroyed))
	er.aindex = make(map[ID][]ID)
	er.bindex = make(map[ID][]ID)

	if B == nil {
		er.b = A
	} else {
		er.b = B
	}
	if er.b != er.a {
		er.b.Watch(0, 0, EntityDestroyedFunc(er.onBDestroyed))
	}
}

func (er *EntityRelation) onADestroyed(ae Entity, _ Type) { er.DeleteA(ae.ID) }
func (er *EntityRelation) onBDestroyed(be Entity, _ Type) { er.DeleteB(be.ID) }

// EntityCreated allocates and clears ID storage space for the given relation
// entity.
func (er *EntityRelation) EntityCreated(rel Entity, _ Type) {
	i := int(rel.Seq())
	for i >= len(er.aid) {
		if i < cap(er.aid) {
			er.aid = er.aid[:i+1]
		} else {
			er.aid = append(er.aid, 0)
		}
	}
	er.aid[i] = 0
	for i >= len(er.bid) {
		if i < cap(er.bid) {
			er.bid = er.bid[:i+1]
		} else {
			er.bid = append(er.bid, 0)
		}
	}
	er.bid[i] = 0
}

// EntityDestroyed clears any stored IDs for the given relation entity.
func (er *EntityRelation) EntityDestroyed(rel Entity, _ Type) {
	i := rel.Seq()
	aid := er.aid[i]
	bid := er.bid[i]
	er.aid[i] = 0
	er.bid[i] = 0
	er.aindex[aid] = withoutID(er.aindex[aid], rel.ID)
	er.bindex[bid] = withoutID(er.bindex[bid], rel.ID)
	// TODO support cascading destroy
}

// Insert creates a relation entity between the given A and B entities.
// The typ argument may provide additional type bits when being used as part of
// a larger EntityRelation-embedding struct.
func (er *EntityRelation) Insert(typ Type, aid, bid ID) Entity {
	// TODO take Entity args for safety
	rel := er.Scope.Create(TypeEntityRelation | typ)
	i := rel.Seq()
	er.aid[i] = aid
	er.bid[i] = bid
	er.aindex[aid] = append(er.aindex[aid], rel.ID)
	er.bindex[bid] = append(er.bindex[bid], rel.ID)
	return rel
}

// InsertMany creates a batch of entity relations from from a single A entity
// to the given B entities. The typ argument is as to Insert().
//
// NOTE The returned batch of Entities is only valid until the next call to
// InsertMany, and MUST NOT be retained.
func (er *EntityRelation) InsertMany(typ Type, aid ID, bids ...ID) Entities {
	// TODO take Entity, Entities args for safety
	er.tmp = er.tmp[:0]
	if cap(er.tmp) < len(bids) {
		er.tmp = make([]ID, 0, len(bids))
	}
	for _, bid := range bids {
		rel := er.Scope.Create(TypeEntityRelation | typ)
		i := rel.Seq()
		er.aid[i] = aid
		er.bid[i] = bid
		er.tmp = append(er.tmp, rel.ID)
		er.bindex[bid] = append(er.bindex[bid], rel.ID)
	}
	er.aindex[aid] = append(er.aindex[aid], er.tmp...)
	return Entities{&er.Scope, er.tmp}
}

// DeleteA destroys any relation entities associated with the given A-side entity.
func (er *EntityRelation) DeleteA(aid ID) {
	ids := er.aindex[aid]
	delete(er.aindex, aid)
	for _, id := range ids {
		Ent(&er.Scope, id).Destroy()
	}
}

// DeleteB destroys any relation entities associated with the given B-side entity.
func (er *EntityRelation) DeleteB(bid ID) {
	ids := er.bindex[bid]
	delete(er.bindex, bid)
	for _, id := range ids {
		Ent(&er.Scope, id).Destroy()
	}
}

// LookupA returns the set of relation entities for a given A-side entity.
func (er *EntityRelation) LookupA(aid ID) Entities {
	return Entities{&er.Scope, er.aindex[aid]}
}

// LookupB returns the set of relation entities for a given B-side entity.
func (er *EntityRelation) LookupB(bid ID) Entities {
	return Entities{&er.Scope, er.bindex[bid]}
}
