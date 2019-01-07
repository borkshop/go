package main

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"

	"borkshop/borkbrand"
	"borkshop/ecs"
	"borkshop/ecs/inspect"
)

/* TODO
- rip out room-based, add ontological gen; probably keep style-based builder
- probably rip out the agent system (free player spawn movement from it)
- inventory system; would be a good place to start a proper player Scope
- items: what're they good for? recipies? player abilities?
- complete the collision system: it needs to leave some trace so that
  collisions can have actions...
- ...speaking of which: actions (pickup items, drop inventory, etc)
*/

type game struct {

	// ag tracks agents across various scopes, primarily for the
	// purpose of calling batch update functions on each scope's
	// population; secondarily it can be used to get easy access to a
	// list of entities of a given type, e.g. to select a random spawn
	// point.
	ag agentSystem

	// shard contains all game entity data; TODO have more than one shard:
	// - coalesce/split regions supported by certain agent populations
	// - to assist, each agent population can define some:
	//   func neededRegion(ecs.ID) image.Rectangle
	// - also need one massive "at rest" or "cold storage" shard for entities
	//   no longer in any agent simulation region
	// - then we only need to run the region(s) supported by player agents at 60hz
	// - anything else (ai-driven regions) can go slower / out-of-band with the UI goroutine
	// - performing (at least most) shard data movement can be done as
	//   inter-frame background work
	// - further per-shard optimizations can be done inter-frame, like moving
	//   component data around to match the spatial index ordering, optimizing
	//   for spatial locality within each system
	shard

	// tmp scratch space
	buf bytes.Buffer

	// ui
	sim  image.Rectangle
	view image.Rectangle
	drag dragState
	pop  popup
}

type shard struct {
	ecs.Scope
	ren render
	pos position
	gen roomGen
}

const (
	gamePosition ecs.Type = 1 << iota
	gameRender
	gameCollides
	gameInput
	gameSpawn
	gameRoom
	gameGen

	gameWall       = gamePosition | gameRender | gameCollides
	gameStack      = gamePosition | gameRender | gameCollides
	gameFloor      = gamePosition | gameRender
	gameDisplay    = gamePosition | gameRender | gameCollides
	gameSpawnPoint = gamePosition | gameSpawn
	gameCharacter  = gamePosition | gameRender | gameCollides
	gamePlayer     = gameCharacter | gameInput
	gameDoor       = gamePosition | gameRender // FIXME | gameCollides
)

const (
	roomWall ecs.Type = 1 << iota
	roomFloor
	roomDoor
)

const (
	playerMoveKey     = "playerMove"
	playerCentroidKey = "playerCentroid"
	playerCountKey    = "playerCount"
)

func (g *game) describe(w io.Writer, ent ecs.Entity) {
	inspect.Describe(w, ent,
		inspect.DescSpec(gameInput, "Ctl", nil),
		inspect.DescSpec(gameCollides, "Col", nil),
		inspect.DescSpec(gamePosition, "Pos", g.describePosition),
		inspect.DescSpec(gameRender, "Ren", g.describeRender),
	)
}

func (g *game) describeRender(ent ecs.Entity) string   { return g.ren.Get(ent).describe("\r\n     ") }
func (g *game) describePosition(ent ecs.Entity) string { return g.pos.Get(ent).String() }

const (
	floorLayer = iota + 1
	aisleLayer
	wallLayer
	furnishLayer
	agentLayer
)

var (
	blackStyle = renStyle(furnishLayer, '[', ']', borkbrand.White.FG()|borkbrand.Black.BG())
	whiteStyle = renStyle(furnishLayer, '[', ']', borkbrand.Black.FG()|borkbrand.White.BG())
	blondStyle = renStyle(furnishLayer, '[', ']', borkbrand.Brown.FG()|borkbrand.Blond.BG())
	brownStyle = renStyle(furnishLayer, '[', ']', borkbrand.Blond.FG()|borkbrand.Brown.BG())

	playerStyle = renStyle(agentLayer, ')', '(', ansi.SGRAttrBold|borkbrand.White.FG()|borkbrand.Guest.BG())
	spiritStyle = renStyle(agentLayer, ')', '(', ansi.SGRAttrBold|borkbrand.Guest.FG())
	wallStyle   = renStyle(wallLayer, '>', '<', ansi.SGRAttrBold|borkbrand.BorkBlue.BG()|borkbrand.DarkBork.FG())
	stackStyle  = renStyle(wallLayer, '[', ']', borkbrand.Brown.FG()|borkbrand.Blond.BG())
	aisleStyle  = renStyle(aisleLayer, '•', '•', borkbrand.Aisle.BG()|borkbrand.Floor.FG())
	floorStyle  = renStyle(floorLayer, '·', '·', borkbrand.Floor.BG()|borkbrand.Black.FG())

	corporealApp = entApps(playerStyle, addEntityType(gameCollides))
	ghostApp     = entApps(spiritStyle, deleteEntityType(gameCollides))
)

func newGame() *game {
	g := &game{}
	g.init()

	const itemZ = 40

	g.gen.roomGenConfig = roomGenConfig{
		Player: entSpec(gamePlayer,
			playerStyle,
		),
		Wall:          entSpec(gameWall, wallStyle),
		Stack:         entSpec(gameWall, stackStyle),
		Floor:         entSpec(gameFloor, floorStyle),
		Aisle:         entSpec(gameFloor, aisleStyle),
		PlaceAttempts: 3,
		MinHallSize:   2,
		MaxHallSize:   8,
		ExitDensity:   25,
	}

	return g
}

func (g *game) init() {
	g.shard.init(g)

	g.ag.registerFunc(g.movePlayers, 0, gamePlayer)
	g.ag.registerFunc(g.spawnPlayers, 1, gameSpawnPoint)
	g.ag.watch(&g.Scope)
}

func (s *shard) init(g *game) {
	s.pos.Init(&s.Scope, gamePosition)
	s.ren.Init(&s.Scope, gamePosition|gameRender, &s.pos)
	s.gen.Init(s, gameGen)
}

func (g *game) Update(ctx *platform.Context) (err error) {
	// Ctrl-C interrupts
	if ctx.Input.HasTerminal('\x03') {
		err = errInt
	}

	// Ctrl-Z suspends
	if ctx.Input.CountRune('\x1a') > 0 {
		defer func() {
			if err == nil {
				err = ctx.Suspend()
			} // else NOTE don't bother suspending, e.g. if Ctrl-C was also present
		}()
	}

	// process any drag region
	if r := g.drag.process(ctx); r != ansi.ZR {
		ir := r.ToImage().Canon().Add(g.view.Min)
		n := 0
		for q := g.pos.Within(ir); q.Next(); n++ {
			posd := q.handle()
			rend := g.ren.Get(posd.Entity())
			log.Printf("%v %v", posd, rend)
		}
		log.Printf("queried %v entities in %v", n, r)
		g.pop.active = false
	}

	// process control input
	if ctx.Input.CountRune('^')%2 == 1 {
		for _, id := range g.ag.ids[&g.Scope][gamePlayer] {
			if rend := g.ren.GetID(id); !rend.zero() {
				if r, _, _ := rend.Cell(); r == '^' {
					corporealApp.apply(&g.shard, g.Entity(id))
				} else {
					ghostApp.apply(&g.shard, g.Entity(id))
				}
			}
		}
	}
	agCtx := nopAgentContext
	if move, interacted := parseTotalMove(ctx.Input); interacted {
		agCtx = addAgentValue(agCtx, playerMoveKey, move)
		g.pop.active = false
	} else if g.drag.active {
		g.pop.active = false
	}

	// run agents
	agCtx, agErr := g.ag.update(agCtx, &g.Scope)
	if err == nil {
		err = agErr
	}

	// center view on player (if any)
	centroid, _ := agCtx.Value(playerCentroidKey).(image.Point)
	size := ctx.Output.Bounds().Size()
	size.X /= 2
	view, _ := centerView(g.view, centroid, size)
	g.view = view

	// run generation within a simulation region around the player
	genning := g.gen.run(g.view)
	if !genning {
		spawn := g.Create(gameSpawnPoint)
		g.pos.Get(spawn).SetPoint(image.ZP)
	}

	// Ctrl-mouse to inspect entities
	if m, haveMouse := ctx.Input.LastMouse(false); haveMouse && m.State.IsMotion() {
		if m.State&ansi.MouseModControl != 0 {
			g.inspect(m.Point)
		}
	}

	ctx.Output.Clear()
	g.ren.drawRegionInto(g.view, &ctx.Output.Grid)

	// at := ansi.Pt(1, ctx.Output.Bounds().Dy())

	// entity count in upper-left
	if ctx.HUD.Visible {
		pt := ansi.Pt(1, 2)
		ctx.Output.To(pt)
		fmt.Fprintf(ctx.Output, "%v entities", g.Scope.Len())

		pt = ansi.Pt(1, pt.Y+1)
		ctx.Output.To(pt)
		fmt.Fprintf(ctx.Output, "view:%v", g.view)

		pt = ansi.Pt(1, pt.Y+1)
		ctx.Output.To(pt)
		fmt.Fprintf(ctx.Output, "sim:%v", g.sim)
	}

	if g.drag.active {
		dr := g.drag.r.Canon()
		// TODO better compositing routine?
		eachCell(ctx.Output.Grid, dr, func(g anansi.Grid, pt ansi.Point, i int) {
			dc := uint32(0x1000)
			if pt.X == dr.Min.X ||
				pt.Y == dr.Min.Y ||
				pt.X == dr.Max.X-1 ||
				pt.Y == dr.Max.Y-1 {
				dc = 0x2000
			}
			// TODO better brighten function
			if g.Rune[i] == 0 {
				g.Rune[i] = ' ' // TODO is this necessary anymore?
			}
			a := g.Attr[i]
			c, _ := a.BG()
			cr, cg, cb, ca := c.RGBA()
			g.Attr[i] = a.SansBG() | ansi.RGBA(cr+dc, cg+dc, cb+dc, ca).BG()
		})
	} else if g.pop.active {
		g.pop.drawInto(&ctx.Output.Grid)
	}

	return err
}

func (g *game) inspect(screenAt ansi.Point) {
	worldAt := screenAt.ToImage().Add(g.view.Min)
	g.buf.Reset()
	if pq := g.pos.At(worldAt); pq.Next() {
		g.buf.Grow(1024)
		g.describe(&g.buf, pq.handle().Entity())
		for pq.Next() {
			_, _ = g.buf.WriteString("\r\n\n")
			g.describe(&g.buf, pq.handle().Entity())
		}
		g.pop.Reload(g.buf.Bytes(), screenAt)
	} else {
		g.pop.Reset()
	}
}

type dragState struct {
	active bool
	r      ansi.Rectangle
}

func (ds *dragState) process(ctx *platform.Context) (r ansi.Rectangle) {
	for id, typ := range ctx.Input.Type {
		if typ == platform.EventMouse {
			m := ctx.Input.Mouse(id)
			if b, isPress := m.State.IsPress(); isPress && b == 0 {
				ds.r.Min = m.Point
				ctx.Input.Type[id] = platform.EventNone
			} else if m.State.IsDrag() {
				ds.r.Max = m.Point
				if ds.r.Min == ansi.ZP {
					ds.r.Min = m.Point
					ds.r.Max = m.Point
				}
				ds.active = true
				ctx.Input.Type[id] = platform.EventNone
			} else {
				if ds.active && m.State.IsRelease() {
					ds.r.Max = m.Point
					r = ds.r
					ctx.Input.Type[id] = platform.EventNone
				}
				ds.active = false
				ds.r = ansi.ZR
				break
			}
		}
	}
	return r
}

func eachCell(g anansi.Grid, r ansi.Rectangle, f func(g anansi.Grid, pt ansi.Point, i int)) {
	for p := r.Min; p.Y < r.Max.Y; p.Y++ {
		for p.X = r.Min.X; p.X < r.Max.X; p.X++ {
			if i, ok := g.CellOffset(p); ok {
				f(g, p, i)
			}
		}
	}
}

// chooseRandomID implements weighted random selection on an arbitrarily
// ordering of entity IDs: any ecs.ArrayIndex.Len() and .ID can be used for n
// and i2id, user needs only to provide a weighting function.
func chooseRandomID(n int, i2id func(i int) ecs.ID, wf func(i int, id ecs.ID) int) (rid ecs.ID) {
	var ws int
	for i := 0; i < n; i++ {
		if id := i2id(i); id != 0 {
			if w := wf(i, id); w > 0 {
				ws += w
				if rid == 0 || rand.Intn(ws+1) <= w {
					rid = id
				}
			}
		}
	}
	return rid
}
