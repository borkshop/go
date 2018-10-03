package main

import (
	"børk.com/ecs"
)

// items contains component data allowing an item definition to be
// attached to an entity (in a world shard).
type items struct {
	defs *itemDefinitions

	ecs.ArrayIndex
	item []ecs.ID
}

const itemType ecs.Type = 1

type itemDefinitions struct {
	setup bool
	ecs.Scope
	info []itemInfo
}

type itemInfo struct {
	name string

	// spec is used to instantiate the item in a world shard
	worldSpec entitySpec

	// TODO once we have an inventory system, we could just as well define
	// inventorySpec entitySpec
}

func (its *items) Init(s *shard, t ecs.Type, defs *itemDefinitions) {
	its.defs = defs
	its.ArrayIndex.Init(&s.Scope)
	s.Scope.Watch(t, 0, its)
}

func (its *items) EntityCreated(ent ecs.Entity, _ ecs.Type) {
	i := its.ArrayIndex.Insert(ent)
	for i >= len(its.item) {
		if i < cap(its.item) {
			its.item = its.item[:i+1]
		} else {
			its.item = append(its.item, 0)
		}
	}
	its.item[i] = 0
}

func (defs *itemDefinitions) init() {
	if !defs.setup {
		defs.setup = true
		defs.Scope.Watch(itemType, 0, defs)
	}
}

func (defs *itemDefinitions) EntityCreated(e ecs.Entity, _ ecs.Type) {
	i := int(e.Seq())
	for i >= len(defs.info) {
		if i < cap(defs.info) {
			defs.info = defs.info[:i+1]
		} else {
			defs.info = append(defs.info, itemInfo{})
		}
	}
	defs.info[i] = itemInfo{}
}

func (defs *itemDefinitions) EntityDestroyed(e ecs.Entity, _ ecs.Type) {
	i := int(e.Seq())
	defs.info[i] = itemInfo{}
}

func (defs *itemDefinitions) load(infos []itemInfo) {
	defs.init()
	for i := range infos {
		item := defs.Create(itemType)
		defs.SetInfo(item, infos[i])
	}
}

func (defs *itemDefinitions) Info(item ecs.Entity) itemInfo          { return defs.info[item.Seq()] }
func (defs *itemDefinitions) SetInfo(item ecs.Entity, info itemInfo) { defs.info[item.Seq()] = info }
