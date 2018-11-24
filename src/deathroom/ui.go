package main

import (
	"fmt"
	"image"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"deathroom/internal/ecs"
	"deathroom/internal/moremath"
	"deathroom/internal/perf"
	"deathroom/internal/point"
	"deathroom/internal/view"
	"deathroom/internal/view/hud"
	"deathroom/internal/view/hud/prompt"

	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"
)

// TODO refactor action keys / items / bar / prompt etc; should be able to
// support adom-style diagonal keys.

var actionKeys = map[rune]action{
	// arrow keys
	rune(ansi.CUD): moveAction(0, 1),
	rune(ansi.CUU): moveAction(0, -1),
	rune(ansi.CUB): moveAction(-1, 0),
	rune(ansi.CUF): moveAction(1, 0),

	// nethack vi keys
	'y': moveAction(-1, -1),
	'u': moveAction(1, -1),
	'n': moveAction(1, 1),
	'b': moveAction(-1, 1),
	'h': moveAction(-1, 0),
	'j': moveAction(0, 1),
	'k': moveAction(0, -1),
	'l': moveAction(1, 0),

	// stay key
	'.': moveAction(0, 0),

	// phase shift
	'_': actionFunc((*world).phaseShift),

	// item inspection
	',': actionFunc((*world).inspectHere),
}

type action interface {
	act(w *world, subject ecs.Iterator) bool
}

type actionFunc func(w *world, subject ecs.Iterator) bool

func (f actionFunc) act(w *world, subject ecs.Iterator) bool { return f(w, subject) }

func moveAction(x, y int) movementAction {
	return movementAction{point.Point{X: x, Y: y}}
}

type movementAction struct {
	pt point.Point
}

func (move movementAction) act(w *world, subject ecs.Iterator) bool {
	for subject.Next() {
		w.addPendingMove(subject.Entity(), move.pt)
	}
	if w.ui.bar.IsRoot() {
		w.updateInspectAction(subject)
	}
	return true
}

type actionItem interface {
	prompt.Runner
	label() string
}

type keyedActionItem interface {
	actionItem
	key() rune
}

type actionBar struct {
	prompt.Prompt
	items []actionItem
	sep   string
}

type labeldRunner struct {
	prompt.Runner
	lb string
}

func (lr labeldRunner) label() string { return lr.lb }

func labeled(run prompt.Runner, mess string, args ...interface{}) actionItem {
	if len(args) > 0 {
		mess = fmt.Sprintf(mess, args...)
	}
	return labeldRunner{run, mess}
}

func (ab *actionBar) Clear() {
	ab.Prompt = ab.Prompt.Unwind()
}

func (ab *actionBar) addAction(ai actionItem) {
	ab.setAction(len(ab.items), ai)
}

func (ab *actionBar) removeLabel(mess string) {
	for i := range ab.items {
		if ab.items[i].label() == mess {
			if j := i + 1; j < len(ab.items) {
				copy(ab.items[i:], ab.items[j:])
			}
			ab.items = ab.items[:len(ab.items)-1]
			return
		}
	}
}

func (ab *actionBar) replaceLabel(mess string, ai actionItem) {
	for i := range ab.items {
		if ab.items[i].label() == mess {
			ab.setAction(i, ai)
			return
		}
	}
	ab.addAction(ai)
}

func (ab *actionBar) setAction(i int, ai actionItem) {
	if i >= len(ab.items) {
		items := make([]actionItem, i+1)
		copy(items, ab.items)
		ab.items = items
	}

	ab.items[i] = ai

	ab.Prompt.Clear()
	for i, ai := range ab.items {
		if r := ab.rune(i); r != 0 {
			ab.AddAction(r, ai, ai.label())
		}
	}
}

func (ab actionBar) rune(i int) rune {
	ai := ab.items[i]
	if ai == nil {
		return 0
	}
	if kai, ok := ai.(keyedActionItem); ok {
		return kai.key()
	} else if i < 9 {
		return '1' + rune(i)
	}
	return 0
}

func (ab actionBar) label(i int) string {
	ai := ab.items[i]
	if r := ab.rune(i); r != 0 {
		return fmt.Sprintf("%s|%s", ai.label(), string(r))
	}
	return fmt.Sprintf("%s|Ø", ai.label())
}

func (ab *actionBar) RenderSize() (wanted, needed point.Point) {
	if ab.Prompt.Len() == 0 {
		return
	}
	if !ab.Prompt.IsRoot() {
		return ab.Prompt.RenderSize()
	}
	if len(ab.items) == 0 {
		return
	}

	i := 0
	wanted.X = utf8.RuneCountInString(ab.label(i))
	needed.X = wanted.X
	wanted.Y = len(ab.items)
	needed.Y = len(ab.items)
	i++

	nsep := utf8.RuneCountInString(ab.sep)
	for ; i < len(ab.items); i++ {
		n := utf8.RuneCountInString(ab.label(i))
		wanted.X += nsep
		wanted.X += n
		if n > needed.X {
			needed.X = n
		}
	}

	return wanted, needed
}

func (ab *actionBar) Render(g view.Grid) {
	if !ab.Prompt.IsRoot() {
		ab.Prompt.Render(g)
		return
	}

	// TODO: maybe use EITHER one row OR one column, not a mix (grid of action
	// items)

	gsz := g.Bounds().Size()

	pt, i := ansi.Pt(1, 1), 0

	pt.X += g.WriteString(pt, ab.label(i))
	i++
	// TODO: missing seps
	for ; i < len(ab.items); i++ {
		lb := ab.label(i)
		if rem := gsz.X - pt.X; rem >= utf8.RuneCountInString(ab.sep)+utf8.RuneCountInString(lb) {
			pt.X += g.WriteString(pt, ab.sep)
			pt.X += g.WriteString(pt, lb)
		} else {
			pt.X = 1
			pt.Y++
			pt.X = g.WriteString(pt, lb)
		}
	}
}

type worldItemAction struct {
	w         *world
	item, ent ecs.Entity
}

func (wia worldItemAction) addAction(pr *prompt.Prompt, ch rune) bool {
	name := wia.w.getName(wia.item, "unknown item")
	return pr.AddAction(ch, wia, name)
}

func (wia worldItemAction) RunPrompt(pr prompt.Prompt) prompt.Prompt {
	if item := wia.w.items[wia.item.ID()]; item != nil {
		return item.interact(pr, wia.w, wia.item, wia.ent)
	}
	return pr.Unwind()
}

type ui struct {
	View *view.View

	shouldProc bool // TODO refactor

	hud.Logs
	perfDash perf.Dash
	prompt   prompt.Prompt
	bar      actionBar
}

type bodySummary struct {
	w   *world
	bo  *body
	ent ecs.Entity
	a   view.Align

	charge      int
	hp, maxHP   int
	armorParts  []string
	damageParts []string
	chargeParts []string
}

func makeBodySummary(w *world, ent ecs.Entity) view.Renderable {
	bs := bodySummary{
		w:   w,
		bo:  w.bodies[ent.ID()],
		ent: ent,
	}
	bs.build()
	return bs
}

func (bs *bodySummary) reset() {
	n := bs.bo.Len() + 1
	bs.charge = 0
	bs.armorParts = nstrings(1, n, bs.armorParts)
	bs.damageParts = nstrings(0, n, bs.damageParts)
	bs.chargeParts = nstrings(1, 1, bs.chargeParts)
}

func (bs *bodySummary) build() {
	bs.reset()

	bs.charge = bs.w.getCharge(bs.ent)

	for it := bs.bo.Iter(ecs.All(bcPart | bcHP)); it.Next(); {
		bs.hp += bs.bo.hp[it.ID()]
		bs.maxHP += bs.bo.maxHP[it.ID()]
	}

	headArmor := 0
	for it := bs.bo.Iter(ecs.All(bcPart | bcHead)); it.Next(); {
		headArmor += bs.bo.armor[it.ID()]
	}

	torsoArmor := 0
	for it := bs.bo.Iter(ecs.All(bcPart | bcTorso)); it.Next(); {
		torsoArmor += bs.bo.armor[it.ID()]
	}

	for _, part := range bs.bo.rel.Leaves(ecs.AllRel(brControl), nil) {
		bs.damageParts = append(bs.damageParts, fmt.Sprintf(
			"%v+%v",
			bs.bo.PartAbbr(part),
			bs.bo.dmg[part.ID()],
		))
	}
	sort.Strings(bs.damageParts)

	bs.armorParts[0] = fmt.Sprintf("Armor: %v %v", headArmor, torsoArmor)
	bs.chargeParts[0] = fmt.Sprintf("Charge: %v", bs.charge)
}

func (bs bodySummary) RenderSize() (wanted, needed point.Point) {
	needed.Y = 5 + 1
	needed.X = moremath.MaxInt(
		7,
		stringsWidth(" ", bs.chargeParts),
	)

	for i := 0; i < len(bs.damageParts); {
		j := i + 2
		if j > len(bs.damageParts) {
			j = len(bs.damageParts)
		}
		needed.X = moremath.MaxInt(needed.X, stringsWidth(" ", bs.damageParts[i:j]))
		needed.Y++
		i = j
	}

	needed.Y++ // XXX why

	return needed, needed
}

func (bs bodySummary) partHPColor(part ecs.Entity) ansi.SGRColor {
	if part == ecs.NilEntity {
		return itemColors[0]
	}
	id := bs.bo.Deref(part)
	if !part.Type().All(bcPart) {
		return itemColors[0]
	}
	hp := bs.bo.hp[id]
	maxHP := bs.bo.maxHP[id]
	return safeColorsIX(itemColors, 1+(len(itemColors)-2)*hp/maxHP)
}

func (bs bodySummary) Render(g view.Grid) {
	// TODO: bodyHPColors ?
	// TODO: support scaling body with grafting

	w := g.Bounds().Dx()
	pt := ansi.Pt(1, 1)
	mess := fmt.Sprintf("%.0f%%", float64(bs.hp)/float64(bs.maxHP)*100)
	pt.X = (w - len(mess)) / 2
	g.WriteString(pt, mess)
	pt.Y++

	//  0123456
	// 0  _O_
	// 1 / | \
	// 2 = | =
	// 3  / \
	// 4_/   \_

	xo := (w - 7) / 2
	for _, part := range []struct {
		off image.Point
		ch  rune
		t   ecs.ComponentType
	}{
		{image.Pt(2, 0), '_', bcUpperArm | bcLeft},
		{image.Pt(3, 0), 'O', bcHead},
		{image.Pt(4, 0), '_', bcUpperArm | bcRight},

		{image.Pt(1, 1), '/', bcForeArm | bcLeft},
		{image.Pt(3, 1), '|', bcTorso},
		{image.Pt(5, 1), '\\', bcForeArm | bcRight},

		{image.Pt(1, 2), '=', bcHand | bcLeft},
		{image.Pt(3, 2), '|', bcTorso},
		{image.Pt(5, 2), '=', bcHand | bcRight},

		{image.Pt(2, 3), '/', bcThigh | bcLeft},
		{image.Pt(4, 3), '\\', bcThigh | bcRight},

		{image.Pt(0, 4), '_', bcFoot | bcLeft},
		{image.Pt(1, 4), '/', bcCalf | bcLeft},
		{image.Pt(5, 4), '\\', bcCalf | bcRight},
		{image.Pt(6, 4), '_', bcFoot | bcRight},
	} {
		it := bs.bo.Iter(ecs.All(bcPart | part.t))
		if it.Next() {
			if i, ok := g.CellOffset(pt.Add(image.Pt(xo, 0)).Add(part.off)); ok {
				g.Rune[i] = part.ch
				g.Attr[i] = bs.partHPColor(it.Entity()).FG()
			}
		}
	}

	pt.Y += 5

	pt.X = 1
	g.WriteString(pt, strings.Join(bs.chargeParts, " "))
	pt.Y++

	for i := 0; i < len(bs.damageParts); {
		j := i + 2
		if j > len(bs.damageParts) {
			j = len(bs.damageParts)
		}
		pt.X = 1
		g.WriteString(pt, strings.Join(bs.damageParts[i:j], " "))
		pt.Y++
		i = j
	}
}

func (ui *ui) init(v *view.View, perf *perf.Perf) {
	ui.View = v
	ui.Logs.Init(1000)
	ui.Logs.Align = view.AlignLeft | view.AlignTop | view.AlignHFlush
	ui.bar.sep = " "
	ui.perfDash.Perf = perf
}

type inputHandler interface {
	HandleInput(ctx view.Context, input platform.Events) error
}

type uiPrompt interface {
	inputHandler
	Active() bool
	Canceled() bool
	Clear()
}

func (ui *ui) HandleInput(ctx view.Context, input platform.Events) error {
	ui.shouldProc = false

	// run perf dashboard ui
	if err := ui.perfDash.HandleInput(ctx, input); err != nil {
		return err
	}

	// run prompt ui
	if err := ui.runPrompt(&ui.prompt, ctx, input); err != nil {
		return err
	}

	// escape to stop
	var rerr error
	if input.HasTerminal(0x1b) {
		rerr = view.ErrStop
	}

	// run action bar
	if err := ui.runPrompt(&ui.bar, ctx, input); err != nil {
		return err
	}

	return rerr
}

func (ui *ui) runPrompt(p uiPrompt, ctx view.Context, input platform.Events) error {
	// run prompt only if a significant action hasn't been taken
	if !ui.shouldProc {
		wasPrompting := p.Active()
		if err := p.HandleInput(ctx, input); err != nil {
			return err
		}
		stillPrompting := p.Active()
		if acted := wasPrompting && !stillPrompting; acted {
			ui.shouldProc = !p.Canceled()
		}
		if !stillPrompting {
			p.Clear()
		}
	}
	return nil
}

func (w *world) HandleInput(ctx view.Context, input platform.Events) (rerr error) {
	ctx.RequestFrame(10 * time.Millisecond) // TODO need-based

	defer func() {
		if rerr != nil {
			_ = w.perf.Close()
		} else if err := w.perf.Err(); err != nil {
			rerr = err
		}
	}()

	// game over check
	if w.over {
		return nil
	}

	// advance the world once we get through input processing, if a significant action was taken, and no error
	w.shouldProc = false
	defer func() {
		if w.shouldProc {
			if rerr == nil {
				w.Process()
			}
			w.shouldProc = false
		}
	}()

	// run ui
	if err := w.ui.HandleInput(ctx, input); err != nil {
		return err
	}

	// player action dispatch
	if !w.shouldProc {
		w.runActions(w.Iter(ecs.All(playMoveMask)), actionKeys, ctx, input)
	}

	return nil
}

func (w *world) runActions(
	subject ecs.Iterator, actionKeys map[rune]action,
	ctx view.Context, input platform.Events,
) {
	if !subject.Any() {
		return
	}
	for i := 0; i < len(input.Type); i++ {
		if input.Type[i] == platform.EventRune {
			r := input.Rune(i)
			if act, def := actionKeys[r]; def {
				input.Type[i] = platform.EventNone
				if w.shouldProc = act.act(w, subject); !w.shouldProc {
					continue
				}
			}
			break
		}
	}
}

func (w *world) phaseShift(subject ecs.Iterator) bool {
	for subject.Next() {
		ent := subject.Entity()
		if ent.Type().All(wcCollide) {
			ent.Delete(wcCollide)
			w.Glyphs[ent.ID()] = '~'
		} else {
			ent.Add(wcCollide)
			w.Glyphs[ent.ID()] = 'X'
		}
		// TODO more flexible glyph mapping
	}
	return true
}

func (w *world) updateInspectAction(subject ecs.Iterator) bool {
	subject.Next()
	player := subject.Entity()
	if itemPrompt, haveItemsHere := w.itemPrompt(w.prompt, player); haveItemsHere {
		w.ui.bar.replaceLabel("Inspect", labeled(itemPrompt, "Inspect"))
	} else {
		w.ui.bar.removeLabel("Inspect")
	}
	return false
}

func (w *world) inspectHere(subject ecs.Iterator) bool {
	// TODO properly integrate as bar action
	// TODO re-use itemPrompt built by updateInspectAction

	subject.Next()
	player := subject.Entity()

	if itemPrompt, haveItemsHere := w.itemPrompt(w.prompt, player); haveItemsHere {
		w.prompt = itemPrompt.RunPrompt(w.prompt.Unwind())
	}
	return false
}

func (w *world) Render(ctx view.Context, termGrid view.Grid) error {
	hud := hud.HUD{
		Logs:  w.ui.Logs,
		World: w.renderViewport(termGrid.Bounds().Size()),
	}

	hud.HeaderF(">%v souls v %v demons", w.Iter(ecs.All(wcSoul)).Count(), w.Iter(ecs.All(wcAI)).Count())

	hud.AddRenderable(&w.ui.bar, view.AlignLeft|view.AlignBottom)
	hud.AddRenderable(&w.ui.prompt, view.AlignLeft|view.AlignBottom)

	hud.AddRenderable(w.ui.perfDash, view.AlignRight|view.AlignBottom)

	for it := w.Iter(ecs.All(wcSoul | wcBody)); it.Next(); {
		hud.AddRenderable(makeBodySummary(w, it.Entity()),
			view.AlignBottom|view.AlignRight|view.AlignHFlush)
	}

	hud.Render(termGrid)
	return nil
}

func (w *world) renderViewport(size image.Point) view.Grid {
	// collect world extent, and compute a viewport focus position
	var (
		bbox  point.Box
		focus point.Point
	)
	for it := w.Iter(ecs.All(renderMask)); it.Next(); {
		pos, _ := w.pos.Get(it.Entity())
		if it.Type().All(wcSoul) {
			// TODO: centroid between all souls would be a way to move beyond
			// "last wins"
			focus = pos
		}
		bbox = bbox.ExpandTo(pos)
	}

	// center clamped grid around focus
	offset := bbox.TopLeft.Add(bbox.Size().Div(2)).Sub(focus)
	ofbox := bbox.Add(offset)
	if ofbox.TopLeft.X < 0 {
		offset.X -= ofbox.TopLeft.X
	}
	if ofbox.TopLeft.Y < 0 {
		offset.Y -= ofbox.TopLeft.Y
	}

	if dx := ofbox.Size().X; size.X > dx {
		size.X = dx
	}
	if dy := ofbox.Size().Y; size.Y > dy {
		size.Y = dy
	}

	// TODO: re-use
	grid := view.MakeGrid(size)
	zVals := make([]uint8, len(grid.Rune))

	// TODO: use an pos range query
	for it := w.Iter(ecs.Clause(wcPosition, wcGlyph|wcAttr)); it.Next(); {
		pos, _ := w.pos.Get(it.Entity())
		pos = pos.Add(offset)
		gi, ok := grid.CellOffset(ansi.PtFromImage(image.Point(pos)))
		if !ok {
			continue
		}

		if it.Type().All(wcGlyph) {
			var attr ansi.SGRAttr
			var zVal uint8

			zVal = 1

			// TODO: move to hp update
			if it.Type().All(wcBody) && it.Type().Any(wcSoul|wcAI) {
				zVal = 255
				hp, maxHP := w.bodies[it.ID()].HPRange()
				if !it.Type().All(wcSoul) {
					zVal--
					attr = safeColorsIX(aiColors, 1+(len(aiColors)-2)*hp/maxHP).FG()
				} else {
					attr = safeColorsIX(soulColors, 1+(len(soulColors)-2)*hp/maxHP).FG()
				}
			} else if it.Type().All(wcSoul) {
				zVal = 127
				attr = soulColors[0].FG()
			} else if it.Type().All(wcAI) {
				zVal = 126
				attr = aiColors[0].FG()
			} else if it.Type().All(wcItem) {
				zVal = 10
				attr = itemColors[len(itemColors)-1].FG()
				if dur, ok := w.items[it.ID()].(durableItem); ok {
					attr = itemColors[0].FG()
					if hp, maxHP := dur.HPRange(); maxHP > 0 {
						attr = safeColorsIX(itemColors, (len(itemColors)-1)*hp/maxHP).FG()
					}
				}
			} else {
				zVal = 2
				if it.Type().All(wcAttr) {
					attr = w.Attr[it.ID()]
				}
			}

			if ch := w.Glyphs[it.ID()]; zVal >= zVals[gi] && ch != 0 {
				grid.Rune[gi] = ch
				grid.Attr[gi] = attr
				zVals[gi] = zVal
			} else {
				continue
			}
		}

		if it.Type().All(wcAttr) {
			if attr := w.Attr[it.ID()]; attr != 0 {
				grid.Attr[gi] = attr.SansFG()
			}
		}
	}

	return grid
}

func (w *world) itemPrompt(pr prompt.Prompt, ent ecs.Entity) (prompt.Prompt, bool) {
	// TODO: once we have a proper spatial index, stop relying on
	// collision relations for this
	prompting := false
	for i, cur := 0, w.moves.Cursor(
		ecs.RelClause(mrCollide, mrItem),
		func(r ecs.RelationType, rel, a, b ecs.Entity) bool { return a == ent },
	); i < 9 && cur.Scan(); i++ {
		if !prompting {
			pr = pr.Sub("Items Here")
			prompting = true
		}
		worldItemAction{w, cur.B(), ent}.addAction(&pr, '1'+rune(i))
	}
	return pr, prompting
}

func (bo *body) interact(pr prompt.Prompt, w *world, item, ent ecs.Entity) prompt.Prompt {
	if ent.Type().All(wcBody) {
		pr = pr.Sub(w.getName(item, "unknown item"))

		for i, it := 0, bo.Iter(ecs.All(bcPart)); i < 9 && it.Next(); i++ {
			part := it.Entity()
			rem := bodyRemains{w, bo, part, item, ent}
			// TODO: inspect menu when more than just scavengable

			// any part can be scavenged
			pr.AddAction('1'+rune(i), prompt.Func(rem.scavenge), rem.describeScavenge())
		}

	} else if ent.Type().All(wcSoul) {
		w.log("you have no body!")
	}
	return pr
}

func safeColorsIX(colors []ansi.SGRColor, i int) ansi.SGRColor {
	if i < 0 {
		return colors[1]
	}
	if i >= len(colors) {
		return colors[len(colors)-1]
	}
	return colors[i]
}

func nstrings(n, m int, ss []string) []string {
	if m > cap(ss) {
		return make([]string, n, m)
	}
	return ss[:n]
}

func stringsWidth(sep string, parts []string) int {
	n := (len(parts) - 1) + utf8.RuneCountInString(sep)
	for _, part := range parts {
		n += utf8.RuneCountInString(part)
	}
	return n
}
