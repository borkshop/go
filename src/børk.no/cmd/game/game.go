package main

import (
	"fmt"
	"image"
	"io"
	"log"
	"math/rand"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"

	"børk.no/ecs"
)

/* TODO
- shift genRoom.{exits,walls} to relation on gameRoom entities
- ui for placing room / hallway blueprints
- digging mechanic
- building mechanic
- automated agents that do goals
- shard(s):
  - world database (at least for level)
  - simulation shards around agent regions
  - agent FoS (field of simulation)
*/

type game struct {
	ag agentSystem

	// TODO shard(s)
	ecs.Scope
	ren   render
	pos   position
	rooms rooms
	gen   roomGen
	goals goalSystem

	// ui
	sim  image.Rectangle
	view image.Rectangle
	drag dragState
	pop  popup
}

const (
	gamePosition ecs.Type = 1 << iota
	gameRender
	gameCollides
	gameInput
	gameSpawn
	gameGoal
	gameRoom
	gameGen

	gameBlueprint = gamePosition | gameRender | gameGoal

	gameWall       = gamePosition | gameRender | gameCollides
	gameFloor      = gamePosition | gameRender
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
	describe(w, ent, []descSpec{
		{gameInput, "Ctl", nil},
		{gameCollides, "Col", nil},
		{gamePosition, "Pos", g.describePosition},
		{gameRender, "Ren", g.describeRender},
	})
}

func (g *game) describeRender(ent ecs.Entity) fmt.Stringer   { return g.ren.Get(ent) }
func (g *game) describePosition(ent ecs.Entity) fmt.Stringer { return g.pos.Get(ent) }

var (
	playerStyle    = renStyle(50, '@', ansi.SGRAttrBold|ansi.RGB(0x60, 0xb0, 0xd0).FG())
	spiritStyle    = renStyle(50, '^', ansi.SGRAttrBold|ansi.RGB(0x60, 0xd0, 0xb0).FG())
	wallStyle      = renStyle(5, '#', ansi.SGRAttrBold|ansi.RGB(0x18, 0x18, 0x18).BG()|ansi.RGB(0x30, 0x30, 0x30).FG())
	floorStyle     = renStyle(4, '·', ansi.RGB(0x10, 0x10, 0x10).BG()|ansi.RGB(0x18, 0x18, 0x18).FG())
	doorStyle      = renStyle(6, '+', ansi.RGB(0x18, 0x18, 0x18).BG()|ansi.RGB(0x60, 0x40, 0x30).FG())
	blueprintStyle = renStyle(15, '?', ansi.RGB(0x08, 0x18, 0x28).BG()|ansi.RGB(0x50, 0x60, 0x70).FG())

	corporealApp = entApps(playerStyle, addEntityType(gameCollides))
	ghostApp     = entApps(spiritStyle, deleteEntityType(gameCollides))
)

func blueprint(t ecs.Type, rs renderStyle, goals ...goal) entitySpec {
	bs := blueprintStyle
	bs.r = rs.r
	return entSpec(gameBlueprint, bs, goalApp(
		radiusGoal(1),
		chainGoals(goals...),
		entSpec(t, rs),
	))
}

func newGame() *game {
	g := &game{}
	g.init()
	g.gen.roomGenConfig = roomGenConfig{
		Player:        entSpec(gamePlayer, playerStyle),
		Wall:          blueprint(gameWall, wallStyle),
		Floor:         blueprint(gameFloor, floorStyle),
		Door:          blueprint(gameDoor, doorStyle),
		PlaceAttempts: 3,
		RoomSize:      image.Rect(5, 3, 21, 13),
		MinHallSize:   2,
		MaxHallSize:   8,
		ExitDensity:   25,
	}

	for minsz := g.gen.RoomSize.Size().Div(4).Mul(3); ; {
		if sz := g.gen.chooseRoomSize(); sz.X >= minsz.X && sz.Y >= minsz.Y {
			g.gen.create(0, image.ZP, image.Rectangle{image.ZP, sz})
			break
		}
	}

	return g
}

func (g *game) init() {
	g.ag.registerFunc(g.movePlayers, 0, gamePlayer)
	g.ag.registerFunc(g.spawnPlayers, 1, gameSpawnPoint)

	// TODO better shard construction
	g.pos.Init(&g.Scope)
	g.ren.Init(&g.Scope)
	g.rooms.Init(&g.Scope)
	g.gen.Init(&g.Scope)
	g.goals.Init(&g.Scope)

	// TODO better dep coupling
	g.ren.pos = &g.pos
	g.gen.rooms = &g.rooms
	g.gen.g = g

	g.Scope.Watch(gameRoom, 0, &g.rooms)
	g.Scope.Watch(gameGen, 0, &g.gen)
	g.Scope.Watch(gamePosition, 0, &g.pos)
	g.Scope.Watch(gamePosition|gameRender, 0, &g.ren)
	g.Scope.Watch(gameGoal, 0, &g.goals)
	g.ag.watch(&g.Scope)
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
	if r := g.drag.process(ctx); r != image.ZR {
		r = r.Canon().Add(g.view.Min)
		n := 0
		for q := g.pos.Within(r); q.Next(); n++ {
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
				if r, _ := rend.Cell(); r == '^' {
					corporealApp.apply(g, g.Entity(id))
				} else {
					ghostApp.apply(g, g.Entity(id))
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
	view, port := centerView(g.view, centroid, ctx.Output.Size)
	g.view = view

	// run generation within a simulation region around the player
	g.sim = g.gen.expandSimRegion(g.view)
	genning := g.gen.run(g.sim)
	if !genning {
		// create a spawn point once generation has stopped
		if len(g.ag.ids[&g.Scope][gameSpawnPoint]) == 0 {
			origin := g.rooms.r[0].Min.Add(g.rooms.r[0].Size().Div(2))
			maxd := compMag(port.Size())

			if id := chooseRandomID(g.rooms.Len(), g.rooms.ID, func(i int, id ecs.ID) int {
				r := &g.rooms.r[i]
				if !r.In(port) {
					return 0
				}
				d := compMag(r.Min.Add(r.Size().Div(2)).Sub(origin))
				sz := r.Size()
				return (maxd - d) * sz.X * sz.Y
			}); id != 0 {
				r := g.rooms.GetID(id)
				log.Printf("add spawn in id:%v r:%v", id, r)
				spawn := g.Create(gameSpawnPoint)
				mid := r.Min.Add(r.Size().Div(2))
				g.pos.Get(spawn).SetPoint(mid)
				g.rooms.parts.Insert(0, id, spawn.ID)
			}
		}
	}

	// Ctrl-mouse to inspect entities
	if m, haveMouse := ctx.Input.LastMouse(false); haveMouse && m.State.IsMotion() {
		any := false
		if m.State&ansi.MouseModControl != 0 {
			pq := g.pos.At(m.Point.Add(g.view.Min))
			if pq.Next() {
				any = true
				g.pop.buf.Reset()
				g.pop.buf.Grow(1024)
				g.describe(&g.pop.buf, pq.handle().Entity())
				for pq.Next() {
					_, _ = g.pop.buf.WriteString("\r\n\n")
					g.describe(&g.pop.buf, pq.handle().Entity())
				}
			}
			if any {
				g.pop.processBuf()
				g.pop.setAt(m.Point)
				g.pop.active = true
			} else {
				g.pop.active = false
			}
		}
	}

	ctx.Output.Clear()
	g.ren.drawRegionInto(g.view, &ctx.Output.Grid)

	// entity count in upper-left
	ctx.Output.To(image.Pt(1, 1))
	fmt.Fprintf(ctx.Output, "%v entities %v rooms", g.Scope.Len(), g.rooms.Used())
	if genning {
		fmt.Fprintf(ctx.Output, " (%v generating)", g.gen.Used())
	}
	ctx.Output.To(image.Pt(1, 2))
	fmt.Fprintf(ctx.Output, "view:%v", g.view)
	ctx.Output.To(image.Pt(2, 3))
	fmt.Fprintf(ctx.Output, "sim:%v", g.sim)

	if g.drag.active {
		dr := g.drag.r.Canon()
		eachCell(&ctx.Output.Grid, dr, func(cell anansi.Cell) {
			dc := uint32(0x1000)
			if cell.X == dr.Min.X ||
				cell.Y == dr.Min.Y ||
				cell.X == dr.Max.X-1 ||
				cell.Y == dr.Max.Y-1 {
				dc = 0x2000
			}
			// TODO better brighten function
			if r := cell.Rune(); r == 0 {
				cell.SetRune(' ') // TODO this shouldn't be necessary, test and fix anansi.Screen
			}
			a := cell.Attr()
			c, _ := a.BG()
			cr, cg, cb, ca := c.RGBA()
			cell.SetAttr(a.SansBG() | ansi.RGBA(cr+dc, cg+dc, cb+dc, ca).BG())
		})
	} else if g.pop.active {
		g.pop.drawInto(&ctx.Output.Grid)
	}

	return err
}

func compMag(p image.Point) (n int) {
	if p.X < 0 {
		n -= p.X
	} else {
		n += p.X
	}
	if p.Y < 0 {
		n -= p.Y
	} else {
		n += p.Y
	}
	return n
}

type dragState struct {
	active bool
	r      image.Rectangle
}

func (ds *dragState) process(ctx *platform.Context) (r image.Rectangle) {
	for id, typ := range ctx.Input.Type {
		if typ == platform.EventMouse {
			m := ctx.Input.Mouse(id)
			if b, isPress := m.State.IsPress(); isPress && b == 0 {
				ds.r.Min = m.Point
				ctx.Input.Type[id] = platform.EventNone
			} else if m.State.IsDrag() {
				ds.r.Max = m.Point
				if ds.r.Min == image.ZP {
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
				ds.r = image.ZR
				break
			}
		}
	}
	return r
}

func eachCell(g *anansi.Grid, r image.Rectangle, f func(anansi.Cell)) {
	for p := r.Min; p.Y < r.Max.Y; p.Y++ {
		for p.X = r.Min.X; p.X < r.Max.X; p.X++ {
			f(g.Cell(p))
		}
	}
}

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
