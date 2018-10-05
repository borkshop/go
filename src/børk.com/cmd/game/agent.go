package main

import (
	"børk.com/ecs"
)

type agentSystem struct {
	registry map[ecs.Type]agency

	agencies       []agency
	agencyTypes    []ecs.Type
	agencyPriority []int
	agencyOrder    []int

	ids map[*ecs.Scope]map[ecs.Type][]ecs.ID
}

func (as *agentSystem) update(ctx agentContext, scope *ecs.Scope) (_ agentContext, err error) {
	tids := as.ids[scope]
	for _, id := range as.agencyOrder {
		if ids := tids[as.agencyTypes[id]]; len(ids) > 0 {
			if ctx, err = as.agencies[id].updateAgents(ctx, ecs.Ents(scope, ids)); err != nil {
				break
			}
		}
	}
	return ctx, err
}

func (as *agentSystem) registerFunc(
	af func(ctx agentContext, es ecs.Entities) (agentContext, error),
	priority int, t ecs.Type,
) {
	as.register(agencyFunc(af), priority, t)
}

func (as *agentSystem) register(a agency, priority int, t ecs.Type) {
	id := len(as.agencies)

	// append data
	as.agencies = append(as.agencies, a)
	as.agencyTypes = append(as.agencyTypes, t)
	as.agencyPriority = append(as.agencyPriority, priority)

	// insert priority order (lowest priority wins)
	ii := 0
	for jj := len(as.agencyOrder); ii < jj; {
		h := int(uint(ii+jj) >> 1) // avoid overflow when computing h
		// ii ≤ h < jj
		if as.agencyPriority[as.agencyOrder[h]] < priority {
			ii = h + 1 // preserves as.agencyPriority[as.agencyOrder[ii-1]] < priority
		} else {
			jj = h // preserves qi.ks[as.agencyOrder[jj]] >= priority
		}
	}
	as.agencyOrder = append(as.agencyOrder, -1)
	copy(as.agencyOrder[ii+1:], as.agencyOrder[ii:])
	as.agencyOrder[ii] = id

	// note type for any future scopes
	as.typeSet()[t] = nil

	// watch any prior scopes
	for scope, tids := range as.ids {
		if tids == nil {
			tids = make(map[ecs.Type][]ecs.ID)
			as.ids[scope] = tids
		}
		if _, def := tids[t]; !def {
			tids[t] = nil
			scope.Watch(t, 0, as)
		}
	}
}

func (as *agentSystem) typeSet() map[ecs.Type][]ecs.ID {
	if as.ids == nil {
		as.ids = make(map[*ecs.Scope]map[ecs.Type][]ecs.ID, 2)
	}
	typeSet := as.ids[nil]
	if typeSet == nil {
		typeSet = make(map[ecs.Type][]ecs.ID)
		as.ids[nil] = typeSet
	}
	return typeSet
}

func (as *agentSystem) watch(scope *ecs.Scope) {
	typeSet := as.typeSet()
	tids := make(map[ecs.Type][]ecs.ID, len(typeSet))
	as.ids[scope] = tids
	for t := range typeSet {
		scope.Watch(t, 0, as)
		tids[t] = nil
	}
}

func (as *agentSystem) EntityCreated(ent ecs.Entity, _ ecs.Type) {
	tids := as.ids[ent.Scope]
	if tids == nil {
		tids = make(map[ecs.Type][]ecs.ID)
		as.ids[ent.Scope] = tids
	}
	et := ent.Type()
	for t, ids := range tids {
		if et.HasAll(t) {
			tids[t] = append(ids, ent.ID)
		}
	}
}

func (as *agentSystem) EntityDestroyed(ent ecs.Entity, _ ecs.Type) {
	tids := as.ids[ent.Scope]
	if tids == nil {
		tids = make(map[ecs.Type][]ecs.ID)
		as.ids[ent.Scope] = tids
	}
	t := ent.Type()
	ids := tids[t]
	for i := 0; i < len(ids); i++ {
		if ids[i] == ent.ID {
			copy(ids[i:], ids[i+1:])
			tids[t] = ids[:len(ids)-1]
			break
		}
	}
}

type agency interface {
	updateAgents(ctx agentContext, es ecs.Entities) (agentContext, error)
}

type agencyFunc func(ctx agentContext, es ecs.Entities) (agentContext, error)

func (af agencyFunc) updateAgents(ctx agentContext, es ecs.Entities) (agentContext, error) {
	return af(ctx, es)
}

type agentContext interface {
	Value(key interface{}) interface{}
}

func addAgentValue(ctx agentContext, keyvals ...interface{}) agentContext {
	if len(keyvals)%2 != 0 {
		panic("invalid number of arguments to addAgentValue")
	}
	var kvs map[interface{}]interface{}
	switch impl := ctx.(type) {
	case agentValueCtx:
		kvs = make(map[interface{}]interface{}, len(keyvals)/2+1)
		kvs[impl.key] = impl.val
		ctx = agentValuesCtx{impl.agentContext, kvs}
	case agentValuesCtx:
		kvs = impl.kvs
	default:
		if len(keyvals) == 2 {
			return agentValueCtx{ctx, keyvals[0], keyvals[1]}
		}
		kvs = make(map[interface{}]interface{}, len(keyvals)/2)
		if ctx == nil {
			ctx = agentValuesCtx{nopAgentContext, kvs}
		} else {
			ctx = agentValuesCtx{ctx, kvs}
		}
	}
	for i := 0; i < len(keyvals); i += 2 {
		kvs[keyvals[i]] = keyvals[i+1]
	}
	return ctx
}

type agentValueCtx struct {
	agentContext
	key, val interface{}
}

func (vc agentValueCtx) Value(key interface{}) interface{} {
	if key == vc.key {
		return vc.val
	}
	return vc.agentContext.Value(key)
}

var nopAgentContext agentContext = agentNopCtx{}

type agentNopCtx struct{}

func (nc agentNopCtx) Value(key interface{}) interface{} {
	return nil
}

type agentValuesCtx struct {
	agentContext
	kvs map[interface{}]interface{}
}

func (vsc agentValuesCtx) Value(key interface{}) interface{} {
	if val, def := vsc.kvs[key]; def {
		return val
	}
	return vsc.agentContext.Value(key)
}
