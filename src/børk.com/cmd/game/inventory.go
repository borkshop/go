package main

import (
	"image"

	"børk.com/ecs"
)

const (
	invSlotData ecs.Type = 1 << iota
	invPosition
	invRender
	invItemData

	invSlot = invPosition | invSlotData
	invItem = invPosition | invRender | invItemData
)

// inventory implements a scope of renderable spatial entities, including
// slots, and items.
type inventory struct {
	setup bool
	size  image.Point

	slotItems, itemSlots map[ecs.ID]ecs.ID // TODO update these

	ecs.Scope
	pos position
	ren render

	slotIndex ecs.ArrayIndex
	slotName  []string

	itemIndex ecs.ArrayIndex
	item      []ecs.Entity
}

func (inv *inventory) Init() {
	if !inv.setup {
		inv.setup = true
		inv.pos.Init(&inv.Scope, invPosition)
		inv.ren.Init(&inv.Scope, invRender, &inv.pos)
		inv.Scope.Watch(invSlot, 0, ecs.EntityCreatedFunc(inv.slotCreated))
		inv.Scope.Watch(invItem, 0, ecs.EntityCreatedFunc(inv.itemCreated))
	}
}

func (inv *inventory) slotCreated(ent ecs.Entity, _ ecs.Type) {
	i := inv.slotIndex.Insert(ent)
	for i >= len(inv.slotName) {
		if i < cap(inv.slotName) {
			inv.slotName = inv.slotName[:i+1]
		} else {
			inv.slotName = append(inv.slotName, "")
		}
	}
	inv.slotName[i] = ""
}

func (inv *inventory) itemCreated(ent ecs.Entity, _ ecs.Type) {
	i := inv.itemIndex.Insert(ent)
	for i >= len(inv.item) {
		if i < cap(inv.item) {
			inv.item = inv.item[:i+1]
		} else {
			inv.item = append(inv.item, ecs.ZE)
		}
	}
	inv.item[i] = ecs.ZE
}

func (inv *inventory) SlotName(ent ecs.Entity) string {
	if i, def := inv.slotIndex.Get(ent); def {
		return inv.slotName[i]
	}
	if slotID, def := inv.itemSlots[ent.ID]; def {
		return inv.SlotName(inv.Entity(slotID))
	}
	return ""
}

func (inv *inventory) SetSlotName(ent ecs.Entity, name string) {
	if i, def := inv.slotIndex.Get(ent); def {
		inv.slotName[i] = name
	}
}

func (inv *inventory) Item(ent ecs.Entity) ecs.Entity {
	if i, def := inv.itemIndex.Get(ent); def {
		return inv.item[i]
	}
	if itemID, def := inv.slotItems[ent.ID]; def {
		return inv.Item(inv.Entity(itemID))
	}
	return ecs.ZE
}

func (inv *inventory) SetItem(ent ecs.Entity, item ecs.Entity) {
	if i, def := inv.itemIndex.Get(ent); def {
		inv.item[i] = item
	}
}

func (inv *inventory) PlaceItemAt(item ecs.Entity, pt image.Point) (priorItem ecs.Entity, ok bool) {
	var slot ecs.Entity
	for pq := inv.pos.At(pt); pq.Next(); {
		switch ent := pq.handle().Entity(); {
		case ent.Type().HasAll(invItem):
			return ent, false
		case ent.Type().HasAll(invSlot):
			slot = ent
		}
	}
	return inv.PlaceItem(item, slot)
}

func (inv *inventory) PlaceItem(item, slot ecs.Entity) (priorItem ecs.Entity, ok bool) {
	inv.MustOwn(item, "item")
	inv.MustOwn(slot, "slot")
	if itemID, def := inv.slotItems[slot.ID]; def {
		return inv.Entity(itemID), false
	}
	if priorSlotID, def := inv.itemSlots[item.ID]; def {
		delete(inv.slotItems, priorSlotID)
	} else {
		if inv.slotItems == nil {
			inv.slotItems = make(map[ecs.ID]ecs.ID)
		}
		if inv.itemSlots == nil {
			inv.itemSlots = make(map[ecs.ID]ecs.ID)
		}
	}
	inv.slotItems[slot.ID] = item.ID
	inv.itemSlots[item.ID] = slot.ID
	return ecs.ZE, true
}
