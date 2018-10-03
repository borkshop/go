package main

import (
	"fmt"
	"image"
	"strings"

	"børk.com/ecs"
)

type entitySpec struct {
	t ecs.Type
	entityApp
}

type entityApp interface {
	apply(s *shard, ent ecs.Entity)
}

type entityAppFunc func(s *shard, ent ecs.Entity)

func (f entityAppFunc) apply(s *shard, ent ecs.Entity) {
	f(s, ent)
}

func entSpec(t ecs.Type, apps ...entityApp) entitySpec {
	return entitySpec{t, entApps(apps...)}
}

func entApps(apps ...entityApp) (app entityApp) {
	for i := range apps {
		app = chainEntityApp(app, apps[i])
	}
	return app
}

func (spec entitySpec) String() string {
	return fmt.Sprintf("t:%v %v", spec.t, spec.entityApp)
}

func (spec entitySpec) create(s *shard, pos image.Point) ecs.Entity {
	ent := s.Scope.Create(spec.t)
	if spec.t.HasAll(gamePosition) {
		s.pos.GetID(ent.ID).SetPoint(pos)
	}
	spec.apply(s, ent)
	return ent
}

func (spec entitySpec) apply(s *shard, ent ecs.Entity) {
	ent.SetType(spec.t)
	if spec.entityApp != nil {
		spec.entityApp.apply(s, ent)
	}
}

type entityApps []entityApp

func (apps entityApps) String() string {
	parts := make([]string, len(apps))
	for i := range apps {
		parts[i] = fmt.Sprint(apps[i])
	}
	return strings.Join(parts, " ")
}

func (apps entityApps) apply(s *shard, ent ecs.Entity) {
	for i := range apps {
		apps[i].apply(s, ent)
	}
}

func chainEntityApp(a, b entityApp) entityApp {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	as, haveAs := a.(entityApps)
	bs, haveBs := b.(entityApps)
	switch {
	case haveAs && haveBs:
		return append(as, bs...)
	case haveAs:
		return append(as, b)
	case haveBs:
		bs = append(bs, nil)
		copy(bs[1:], bs)
		bs[0] = a
		return bs
	}
	return entityApps{a, b}
}

type addEntityType ecs.Type
type deleteEntityType ecs.Type

func (t addEntityType) apply(_ *shard, ent ecs.Entity)    { ent.AddType(ecs.Type(t)) }
func (t deleteEntityType) apply(_ *shard, ent ecs.Entity) { ent.DeleteType(ecs.Type(t)) }
