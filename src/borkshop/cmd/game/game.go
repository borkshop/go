package main

import (
	"fmt"
	"image"
	"io"
	"log"
	"strconv"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"

	"borkshop/ecs"
	"borkshop/ecs/inspect"
)

/* TODO
- rip out room-based, add ontological gen; probably keep style-based builder
- rip out goal system (probably)
- probably rip out the agent system (free player spawn movement from it)
- inventory system; would be a good place to start a proper player Scope
- items: what're they good for? recipies? player abilities?
- complete the collision system: it needs to leave some trace so that
  collisions can have actions...
- ...speaking of which: actions (pickup items, drop inventory, etc)
*/

type game struct {
	itemDefs itemDefinitions

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

	// ui
	sim  image.Rectangle
	view image.Rectangle
	drag dragState
	pop  popup
}

type shard struct {
	ecs.Scope
	ren   render
	pos   position
	rooms rooms
	gen   roomGen
	goals goalSystem
	items items

	bodIndex ecs.ArrayIndex
	bod      []body
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
	gameItemInfo
	gameBody

	gameBlueprint = gamePosition | gameRender | gameGoal

	gameWall       = gamePosition | gameRender | gameCollides
	gameFloor      = gamePosition | gameRender
	gameSpawnPoint = gamePosition | gameSpawn
	gameCharacter  = gamePosition | gameRender | gameCollides
	gamePlayer     = gameCharacter | gameInput
	gameDoor       = gamePosition | gameRender // FIXME | gameCollides
	gameItem       = gamePosition | gameRender | gameItemInfo
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

var (
	playerStyle    = renStyle(50, ')', '(', ansi.SGRAttrBold|ansi.RGB(0x0, 0xb0, 0xd0).FG())
	spiritStyle    = renStyle(50, '}', '{', ansi.SGRAttrBold|ansi.RGB(0x60, 0xd0, 0xb0).FG())
	wallStyle      = renStyle(5, '>', '<', ansi.SGRAttrBold|ansi.RGB(0x1f, 0x1f, 0x7f).BG()|ansi.RGB(0, 0, 0x5f).FG())
	floorStyle     = renStyle(4, '·', '·', ansi.RGB(0x7f, 0x7f, 0x7f).BG()|ansi.RGB(0x18, 0x18, 0x18).FG())
	aisleStyle     = renStyle(4, '•', '•', ansi.RGB(0x9f, 0x9f, 0x9f).BG()|ansi.RGB(0x7f, 0x7f, 0x7f).FG())
	doorStyle      = renStyle(6, '⫤', '⊫', ansi.RGB(0x18, 0x18, 0x18).BG()|ansi.RGB(0x60, 0x40, 0x30).FG())
	blueprintStyle = renStyle(15, '?', '¿', ansi.RGB(0x08, 0x18, 0x28).BG()|ansi.RGB(0x50, 0x60, 0x70).FG())

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

	const itemZ = 40

	g.itemDefs.load([]itemInfo{
		{"wrench", entSpec(gameItem, renStyle(itemZ, '🔧', ' ', ansi.SGRAttrBold|ansi.RGB(0xc0, 0xc8, 0xd0).FG()))},

		// 🔩
		// {"screwdriver"},

		{"hammer", entSpec(gameItem, renStyle(itemZ, '🔨', ' ', ansi.SGRAttrBold|ansi.RGB(0xd0, 0xc0, 0xb0).FG()))},

		// {"finishing nail"},
		// {"carpentry nail"},
		// {"drywall screw"},

		// {"doorknob"},
		{"plywood sheet", entSpec(gameItem, renStyle(itemZ, '▤', ' ', ansi.SGRAttrBold|ansi.RGB(0xd0, 0xd0, 0x60).FG()))},
		// {"angle bracket"},

		// {"zip tie"},
		// {"plastic bag"},

	})

	g.gen.roomGenConfig = roomGenConfig{
		Player: entSpec(gamePlayer|gameBody,
			playerStyle,
			&defaultBodyDef,
			entityAppFunc(func(s *shard, ent ecs.Entity) {
				i, _ := s.bodIndex.GetID(ent.ID)
				bod := &s.bod[i]
				for i := 0; i < bod.slots.Len(); i++ {
					part := bod.slots.Entity(i)
					part.AddType(bodyRune | bodyRuneAttr)
					for _, r := range strconv.FormatInt(int64(i), 16) {
						bod.runes[part.Seq()] = r
						break
					}
					bod.runeAttr[part.Seq()] = ansi.RGB(0x20, 0x40, 0xb0).FG()
				}
				for i := 0; i < bod.hands.Len(); i++ {
					part := bod.hands.Entity(i)
					part.AddType(bodyRune | bodyRuneAttr)
					bod.runeAttr[part.Seq()] = ansi.RGB(0x20, 0xb0, 0x40).FG()
				}
			}),
		),
		Wall:          entSpec(gameWall, wallStyle),
		Floor:         entSpec(gameFloor, floorStyle),
		Aisle:         entSpec(gameFloor, aisleStyle),
		Door:          entSpec(gameDoor, doorStyle),
		PlaceAttempts: 3,
		RoomSize:      image.Rect(5, 3, 21, 13),
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
	s.rooms.Init(s, gameRoom)
	s.gen.Init(s, gameGen)
	s.goals.Init(s, gameGoal)
	s.items.Init(s, gameItem, &g.itemDefs)
	s.Scope.Watch(gameBody, 0, ecs.Watchers{
		&s.bodIndex,
		ecs.EntityCreatedFunc(s.bodyCreated),
	})
}

func (s *shard) bodyCreated(e ecs.Entity, t ecs.Type) {
	i, _ := s.bodIndex.GetID(e.ID)
	for i >= len(s.bod) {
		if i < cap(s.bod) {
			s.bod = s.bod[:i+1]
		} else {
			s.bod = append(s.bod, body{})
		}
	}
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
		any := false
		if m.State&ansi.MouseModControl != 0 {
			pq := g.pos.At(m.Point.ToImage().Add(g.view.Min))
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
				g.pop.setAt(m.Point.Add(image.Pt(1, 1)))
				g.pop.active = true
			} else {
				g.pop.active = false
			}
		}
	}

	ctx.Output.Clear()
	g.ren.drawRegionInto(g.view, &ctx.Output.Grid)

	at := ansi.Pt(1, ctx.Output.Bounds().Dy())

	for _, id := range g.ag.ids[&g.Scope][gamePlayer] {
		player := g.Entity(id)
		if i, def := g.bodIndex.Get(player); def {
			rend := g.ren.Get(player)
			_, _, a := rend.Cell() // TODO better integrate body attrs
			bod := &g.bod[i]
			at.Y -= bod.Size().Y
			bod.RenderInto(ctx.Output.Grid, at, a)
			at = at.Add(bod.Size()).Add(image.Pt(1, 0))
		}
	}

	// entity count in upper-left
	if ctx.HUD.Visible {
		pt := ansi.Pt(1, 2)
		ctx.Output.To(pt)
		fmt.Fprintf(ctx.Output, "%v entities %v rooms", g.Scope.Len(), g.rooms.Used())

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
