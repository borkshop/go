package main

import (
	"image"

	"børk.no/ecs"
)

type rooms struct {
	ecs.ArrayIndex
	r []image.Rectangle

	parts ecs.EntityRelation
}

func (rooms *rooms) Init(s *shard, t ecs.Type) {
	rooms.ArrayIndex.Init(&s.Scope)
	rooms.ArrayIndex.Init(&s.Scope)
	rooms.parts.Init(&s.Scope, nil)
	s.Scope.Watch(t, 0, rooms)
}

func (rooms *rooms) EntityCreated(ent ecs.Entity, _ ecs.Type) {
	i := rooms.ArrayIndex.Insert(ent)
	for i >= len(rooms.r) {
		if i < cap(rooms.r) {
			rooms.r = rooms.r[:i+1]
		} else {
			rooms.r = append(rooms.r, image.ZR)
		}
	}
	rooms.r[i] = image.ZR
}

func (rooms *rooms) Get(ent ecs.Entity) *image.Rectangle {
	if i, def := rooms.ArrayIndex.Get(ent); def {
		return &rooms.r[i]
	}
	return nil
}

func (rooms *rooms) GetID(id ecs.ID) *image.Rectangle {
	if i, def := rooms.ArrayIndex.GetID(id); def {
		return &rooms.r[i]
	}
	return nil
}
