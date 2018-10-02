package main

import (
	"fmt"
	"image"
	"strings"

	"børk.no/ecs"
)

type goalSystem struct {
	ecs.ArrayIndex
	data []goal
}

type goal interface {
	check(s *shard, subject, object ecs.Entity) goalReq
	fulfill(s *shard, subject, object ecs.Entity)
}

type goalReq interface {
	error
	value(key interface{}) []interface{}
}

type goals []goal
type goalReqs []goalReq

func (goalSys *goalSystem) Init(s *shard, t ecs.Type) {
	goalSys.ArrayIndex.Init(&s.Scope)
	s.Scope.Watch(t, 0, goalSys)
}

func (goalSys *goalSystem) Get(ent ecs.Entity) goal {
	if i, def := goalSys.ArrayIndex.Get(ent); def {
		return goalSys.data[i]
	}
	return nil
}

func (goalSys *goalSystem) GetID(id ecs.ID) goal {
	if i, def := goalSys.ArrayIndex.GetID(id); def {
		return goalSys.data[i]
	}
	return nil
}

func (goalSys *goalSystem) EntityDestroyed(ent ecs.Entity, _ ecs.Type) {
	if i, def := goalSys.ArrayIndex.Delete(ent); def {
		goalSys.data[i] = nil
	}
}

func (goalSys *goalSystem) EntityCreated(ent ecs.Entity, _ ecs.Type) {
	i := goalSys.ArrayIndex.Insert(ent)

	for i >= len(goalSys.data) {
		if i < cap(goalSys.data) {
			goalSys.data = goalSys.data[:i+1]
		} else {
			goalSys.data = append(goalSys.data, nil)
		}
	}
	goalSys.data[i] = nil
}

func (gs goals) check(s *shard, subject, object ecs.Entity) (req goalReq) {
	for i := range gs {
		req = chainGoalReq(req, gs[i].check(s, subject, object))
	}
	return req
}

func (gs goals) fulfill(s *shard, subject, object ecs.Entity) {
	for i := range gs {
		gs[i].fulfill(s, subject, object)
	}
}

func (grs goalReqs) value(key interface{}) (vals []interface{}) {
	for i := range grs {
		if ivals := grs[i].value(key); len(ivals) > 0 {
			vals = append(vals, ivals...)
		}
	}
	return nil
}

func (grs goalReqs) Error() string {
	parts := make([]string, len(grs))
	for i := range grs {
		parts[i] = fmt.Sprint(grs[i])
	}
	return strings.Join(parts, " ")
}

func chainGoals(goals ...goal) (g goal) {
	for i := range goals {
		g = chainGoal(g, goals[i])
	}
	return g
}

func chainGoal(a, b goal) goal {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	as, haveAs := a.(goals)
	bs, haveBs := b.(goals)
	if haveAs && haveBs {
		return append(as, bs...)
	}
	if haveAs {
		return append(as, b)
	}
	if haveBs {
		bs = append(bs, nil)
		copy(bs[1:], bs)
		bs[0] = a
		return bs
	}
	return goals{a, b}
}

func chainGoalReq(a, b goalReq) goalReq {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	as, haveAs := a.(goalReqs)
	bs, haveBs := b.(goalReqs)
	if haveAs && haveBs {
		return append(as, bs...)
	}
	if haveAs {
		return append(as, b)
	}
	if haveBs {
		bs = append(bs, nil)
		copy(bs[1:], bs)
		bs[0] = a
		return bs
	}
	return goalReqs{a, b}
}

func goalApp(goals ...goal) goalApplication {
	return goalApplication{chainGoals(goals...)}
}

type goalApplication struct {
	goal
}

func (ga goalApplication) apply(s *shard, ent ecs.Entity) {
	if i, def := s.goals.ArrayIndex.Get(ent); def {
		s.goals.data[i] = ga.goal
	}
}

func (spec entitySpec) check(s *shard, subject, object ecs.Entity) goalReq { return nil }
func (spec entitySpec) fulfill(s *shard, subject, object ecs.Entity)       { spec.apply(s, object) }

type radiusGoal int
type radiusReq struct {
	rad radiusGoal
	pos image.Point
}

func (rg radiusGoal) check(s *shard, subject, object ecs.Entity) goalReq {
	return radiusReq{rg, s.pos.Get(object).Point()}
}

func (rg radiusGoal) fulfill(s *shard, subject, object ecs.Entity) {}

func (rq radiusReq) Error() string {
	return fmt.Sprintf("must be within %v cells of %v", rq.rad, rq.pos)
}

func (rq radiusReq) value(key interface{}) []interface{} {
	if key == gamePosition {
		n := int(rq.rad)
		return []interface{}{image.Rectangle{
			rq.pos.Sub(image.Pt(n, n)),
			rq.pos.Add(image.Pt(n+1, n+1))}}
	}
	return nil
}
