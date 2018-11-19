package ecsTime

import "github.com/jcorbin/execs/internal/ecs"

// Timers implements a timer facility attached to an ecs.Core.
type Timers struct {
	// NOTE: dense storage strategy, may want to explicate that
	// eventually and provide alternate
	core *ecs.Core
	t    ecs.ComponentType

	// TODO: shift to a heap rather than iterating and decrementing every time;
	// maybe introspectively after len(timers) passes a certain point?
	timers []timer
}

type timer struct {
	t        ecs.ComponentType
	remain   int
	period   int
	callback func(ecs.Entity)
}

// Init sets up the timer facility, attached to the given ecs.Core, and using
// the supplied ComponentType to indicate "has a timer". The given
// ComponentType MUST NOT be registered by another allocator.
func (ti *Timers) Init(core *ecs.Core, t ecs.ComponentType) {
	if ti.core != nil {
		panic("Timers already initialized")
	}
	ti.core = core
	ti.t = t
	ti.timers = []timer{{}}
	ti.core.RegisterAllocator(ti.t, ti.alloc)
	ti.core.RegisterDestroyer(ti.t, ti.destroyTimer)
}

// After attaches a one-shot timer to the given entity that expires after N
// System.Process() ticks. The timer calls the given function once expired. Any
// prior timer attached to the entity is overwritten.
func (ti *Timers) After(ent ecs.Entity, N int, callback func(ecs.Entity)) {
	id := ti.core.Deref(ent)
	ent.Add(ti.t)
	ti.timers[id] = timer{remain: N, callback: callback}
}

// Every attaches a periodic timer to the given entity that fires every N
// System.Process() ticks. The timer calls the given function every time. Any
// prior timer attached to the entity is overwritten.
func (ti *Timers) Every(ent ecs.Entity, N int, callback func(ecs.Entity)) {
	id := ti.core.Deref(ent)
	ent.Add(ti.t)
	ti.timers[id] = timer{remain: N, period: N, callback: callback}
}

// Cancel deletes any timer attached to the given entity, returning true only
// if there was such a timer to delete.
func (ti *Timers) Cancel(ent ecs.Entity) bool {
	_ = ti.core.Deref(ent)
	if ent.Type().All(ti.t) {
		ent.Delete(ti.t)
		return true
	}
	return false
}

// Process calls any timers whose time has come.
//
// Callback functions are called (in an ARBITRARY order) in one batch AFTER all
// expired timers have been processed. Therefore callbacks may re-set a
// one-shot, or cancel a periodic (their own timer, or another).
func (ti *Timers) Process() {
	for it := ti.core.Iter(ecs.All(ti.t)); it.Next(); {
		t := &ti.timers[it.ID()]
		if t.remain <= 0 {
			continue
		}
		t.remain--
		if t.remain > 0 {
			continue
		}
		ent := it.Entity()
		defer t.process(ent)(ent)
	}
}

func (t *timer) process(ent ecs.Entity) func(ecs.Entity) {
	callback := t.callback
	if t.period != 0 {
		t.remain = t.period // interval refresh
	} else {
		ent.Delete(t.t) // one shot
	}
	return callback
}

func (ti *Timers) alloc(id ecs.EntityID, t ecs.ComponentType)        { ti.timers = append(ti.timers, timer{}) }
func (ti *Timers) destroyTimer(id ecs.EntityID, t ecs.ComponentType) { ti.timers[id] = timer{} }
