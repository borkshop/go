package main

import (
	"image"
	"math/rand"

	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"

	"borkshop/ecs"
)

func (g *game) movePlayers(ctx agentContext, es ecs.Entities) (agentContext, error) {
	move, haveMove := ctx.Value(playerMoveKey).(image.Point)
	var centroid image.Point

	for i := range es.IDs {
		player := es.Entity(i)
		posd := g.pos.Get(player)
		pos := posd.Point()

		// TODO other options beyond apply-to-all
		if haveMove {
			// TODO proper movement system
			if newPos := pos.Add(move); g.pos.collides(player, newPos) == ecs.ZE {
				posd.SetPoint(newPos)
				pos = newPos
			}
		}

		centroid = centroid.Add(pos)
	}

	ctx = addAgentValue(ctx,
		playerCentroidKey, centroid.Div(len(es.IDs)),
		playerCountKey, len(es.IDs))
	return ctx, nil
}

func (g *game) spawnPlayers(ctx agentContext, es ecs.Entities) (agentContext, error) {
	if n, _ := ctx.Value(playerCountKey).(int); n == 0 {
		id := es.IDs[0]
		for i := 1; i < len(es.IDs); i++ {
			if rand.Intn(i+1) == 0 {
				id = es.IDs[i]
			}
		}
		spawnPos := g.pos.GetID(id).Point()
		g.gen.Player.create(&g.shard, spawnPos)
		// log.Printf("spawn player @%v", spawnPos)
	}
	return ctx, nil
}

func parseTotalMove(in *platform.Events) (move image.Point, interacted bool) {
	for id := range in.Type {
		if dp, any := parseMove(in, id); any {
			move = move.Add(dp)
			interacted = true
		}
	}
	return move, interacted
}

func parseMove(in *platform.Events, id int) (_ image.Point, interacted bool) {
	defer func() {
		if interacted {
			in.Type[id] = platform.EventNone
		}
	}()

	// TODO support numpad

	switch in.Type[id] {
	case platform.EventEscape:
		esc := in.Escape(id)
		if d, isMove := ansi.DecodeCursorCardinal(esc.ID, esc.Arg); isMove {
			return d, true
		}

	case platform.EventRune:
		switch in.Rune(id) {
		case 'y':
			return image.Pt(-1, -1), true
		case 'u':
			return image.Pt(1, -1), true
		case 'n':
			return image.Pt(1, 1), true
		case 'b':
			return image.Pt(-1, 1), true
		case 'h':
			return image.Pt(-1, 0), true
		case 'j':
			return image.Pt(0, 1), true
		case 'k':
			return image.Pt(0, -1), true
		case 'l':
			return image.Pt(1, 0), true
		case '.':
			return image.ZP, true
		}
	}

	return image.ZP, false
}

// TODO proper movement / collision system
func (pos *position) collides(ent ecs.Entity, p image.Point) (hit ecs.Entity) {
	if ent.Type()&gameCollides != 0 {
		n := 0
		for q := pos.At(p); q.Next(); {
			hitPosd := q.handle()
			other := hitPosd.Entity()
			typ := other.Type()
			// log.Printf("q:%v coll check %v type:%v", q, other, typ)
			if typ&gameCollides != 0 {
				// TODO better than last wins
				hit = other
			}
			n++
		}
		// FIXME
		// if hit != ecs.ZE {
		// 	log.Printf("%v at %v hit:%v type:%v", n, p, hit, hit.Type())
		// } else {
		// 	log.Printf("%v at %v hit:none", n, p)
		// }
	}
	return hit
}

func centerView(view image.Rectangle, centroid, size image.Point) (_, port image.Rectangle) {
	if view == image.ZR {
		offset := centroid.Sub(size.Div(2))
		view = image.Rectangle{offset, size.Add(offset)}
	} else if view.Size() != size {
		view = image.Rectangle{
			view.Min.Sub(view.Size().Sub(size).Div(2)),
			view.Min.Add(size),
		}
	}
	ds := view.Size().Div(8)
	port = image.Rectangle{
		view.Min.Add(ds),
		view.Max.Sub(ds),
	}
	view = view.Add(compMinMax(
		centroid.Sub(port.Min),
		centroid.Sub(port.Max),
	))
	return view, port
}

func compMinMax(min, max image.Point) (pt image.Point) {
	if min.X < 0 {
		pt.X = min.X
	} else if max.X > 0 {
		pt.X = max.X
	}
	if min.Y < 0 {
		pt.Y = min.Y
	} else if max.Y > 0 {
		pt.Y = max.Y
	}
	return pt
}
