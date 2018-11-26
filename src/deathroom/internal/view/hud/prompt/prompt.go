package prompt

import (
	"fmt"
	"unicode/utf8"

	"deathroom/internal/moremath"
	"deathroom/internal/point"
	"deathroom/internal/view"

	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"
)

// Prompt represents a set of actions that the user may select from. Each
// action has an associated key rune, message, and action. Prompts may be
// chained, i.e. the user is being shown a sub-prompt. Prompts may be left or
// right aligned when rendered.
type Prompt struct {
	prior    *Prompt
	canceled bool
	mess     string
	align    view.Align
	action   []promptAction
	keys     []rune
}

// Runner represents an action invoked by a Prompt. RunPrompt is called when a
// Prompt action has been invoked by the user. It gets the prior Prompt value,
// and is expected to return a new Prompt value to display to the user.
//
// If the returned prompt value has no actions, this indicates that an action
// was taken.
type Runner interface {
	RunPrompt(prior Prompt) Prompt
}

// Func is a convenient way to implement Runner arounnd a single function.
type Func func(prior Prompt) Prompt

// RunPrompt calls the aliased function.
func (f Func) RunPrompt(pr Prompt) Prompt { return f(pr) }

type promptAction struct {
	ch   rune
	mess string
	run  Runner
}

const (
	headerOverhead = 1
	headerFmt      = "%s:"
	exitRune       = '0'
	exitLeftMess   = "0) Exit Menu"
	exitRightMess  = "Exit Menu (0"
	actionOverhead = 3
	actionLeftFmt  = "%s) %s"
	actionRightFmt = "%s (%s"
)

func (act promptAction) renderActionLeft() string {
	return fmt.Sprintf(actionLeftFmt, string(act.ch), act.mess)
}

func (act promptAction) renderActionRight() string {
	return fmt.Sprintf(actionRightFmt, act.mess, string(act.ch))
}

// RenderSize calculates how much space the prompt could use and how much it
// needs. TODO: not yet paginated.
func (pr *Prompt) RenderSize() (wanted, needed point.Point) {
	if pr.Len() == 0 {
		return
	}

	// header
	if n := utf8.RuneCountInString(pr.mess); n > 0 {
		needed.X = moremath.MaxInt(needed.X, n)
		wanted.X = moremath.MaxInt(wanted.X, n+headerOverhead)
		needed.Y++
		wanted.Y++
	}

	// TODO: vary {needed wanted}.Y for pagination
	for _, act := range pr.action {
		n := utf8.RuneCountInString(act.mess)
		needed.X = moremath.MaxInt(needed.X, n+actionOverhead)
		wanted.X = moremath.MaxInt(wanted.X, n+actionOverhead)
		needed.Y++
		wanted.Y++
	}

	// footer
	needed.X = moremath.MaxInt(needed.X, utf8.RuneCountInString(exitLeftMess))
	wanted.X = moremath.MaxInt(wanted.X, utf8.RuneCountInString(exitLeftMess))
	needed.Y++
	wanted.Y++

	return wanted, needed
}

// Render the prompt within the given space.
func (pr *Prompt) Render(g view.Grid) {
	gsz := g.Bounds().Size()

	i, pt := 0, ansi.Pt(1, 1)

	var write func(mess string, args ...interface{})
	var exitMess string

	if pr.align&view.AlignCenter == view.AlignRight {
		exitMess = exitRightMess
		write = func(mess string, args ...interface{}) {
			pt.X = gsz.X
			g.WriteStringRTL(pt, mess, args...)
		}
	} else {
		exitMess = exitLeftMess
		write = func(mess string, args ...interface{}) {
			g.WriteString(pt, mess, args...)
		}
	}

	if pr.mess != "" {
		g.WriteString(pt, headerFmt, pr.mess) // TODO y not write() ?
		pt.Y++
	}
	for ; pt.Y < gsz.Y && i < pr.Len(); pt, i = ansi.Pt(1, pt.Y+1), i+1 {
		write(pr.action[i].renderActionRight())
	}
	write(exitMess)
	// if i < pr.Len() TODO: paginate
}

// HandleInput processes input events for the prompt.
func (pr *Prompt) HandleInput(ctx view.Context, input *platform.Events) error {
	// TODO: pagination support

	for pr.prior != nil && pr.Len() > 0 {
		pr.canceled = false
		pr.collectKeys()
		switch r := input.TakeRune(pr.keys...); r {
		case 0x1b:
			*pr = pr.Unwind()
			pr.canceled = true
		case exitRune:
			*pr = pr.Pop()
		default:
			if next, ok := pr.runKey(r); ok {
				*pr = next
			} else {
				*pr = pr.Unwind()
				pr.canceled = true
			}
		}
	}

	return nil
}

// Run runs the i-th action, returning the resulting next prompt state and
// true; if i is invalid, the current prompt and fals.
func (pr Prompt) Run(i int) (next Prompt, ok bool) {
	if i < 0 || i >= pr.Len() {
		return pr, false
	}
	return pr.action[i].run.RunPrompt(pr), true
}

func (pr Prompt) runKey(r rune) (next Prompt, ok bool) {
	for i := 0; i < len(pr.keys); i++ {
		if r == pr.keys[i] {
			return pr.Run(i)
		}
	}
	return pr, false
}

func (pr *Prompt) collectKeys() {
	n := 2 + pr.Len()
	if cap(pr.keys) < n {
		pr.keys = make([]rune, 0, 2*n)
	}
	pr.keys = pr.keys[:0]
	for i := range pr.action {
		pr.keys = append(pr.keys, pr.action[i].ch)
	}
	pr.keys = append(pr.keys, 0x1b, exitRune)
}

// SetMess sets the header message.
func (pr *Prompt) SetMess(mess string, args ...interface{}) {
	if len(args) > 0 {
		pr.mess = fmt.Sprintf(mess, args...)
	} else if len(mess) > 0 {
		pr.mess = mess
	} else {
		pr.mess = ""
	}
}

// SetAlign ment for this prompt; only horizontal left/right bits matter.
func (pr *Prompt) SetAlign(align view.Align) {
	pr.align = align
}

// Sub returns a new sub-prompt of the current one with the given header message.
func (pr Prompt) Sub(mess string, args ...interface{}) Prompt {
	sub := Prompt{
		prior: &pr,
		align: pr.align,
	}
	sub.SetMess(mess, args...)
	return sub
}

// Pop returns the parent prompt, if any, or this prompt if it has no parent.
func (pr Prompt) Pop() Prompt {
	if pr.prior != nil {
		return *pr.prior
	}
	return pr
}

// Unwind the prompt, returning the root prompt (which may be the current
// prompt if not a sub-prompt).
func (pr Prompt) Unwind() Prompt {
	for pr.prior != nil {
		pr = *pr.prior
	}
	return pr
}

// Clear prompt state, by unwinding the prompt, clearing its mesage, and
// truncating its actions.
func (pr *Prompt) Clear() {
	*pr = pr.Unwind()
	pr.mess = ""
	pr.action = pr.action[:0]
}

// Canceled returns true only if the prompt was just de-activated (e.g. by
// user backing out of all prompts, ESC-aping).
func (pr Prompt) Canceled() bool { return pr.canceled }

// Active returns true if the prompt needs further user input to take an
// action.
func (pr Prompt) Active() bool { return len(pr.action) > 0 }

// Len returns how many actions are in this prompt.
func (pr Prompt) Len() int { return len(pr.action) }

// IsRoot returns true only if this prompt is not a sub-prompt.
func (pr Prompt) IsRoot() bool { return pr.prior == nil }

// AddAction adds a new action to the prompt with the given activation rune,
// display message, and action to run; if the rune conflicts with an already
// added action, then the addition fails and false is returned; otherwise true
// is returned.
func (pr *Prompt) AddAction(ch rune, run Runner, mess string, args ...interface{}) bool {
	for i := range pr.action {
		if pr.action[i].ch == ch {
			return false
		}
	}
	if len(args) > 0 {
		mess = fmt.Sprintf(mess, args...)
	}
	pr.action = append(pr.action, promptAction{ch, mess, run})
	return true
}

// RemoveAction removes an action matching the given rune, runner, or message
// (in that order of precedence); zero values will not match. Returns true if
// an action was removed, false otherwise.
func (pr *Prompt) RemoveAction(ch rune, run Runner, mess string) bool {
	for i := range pr.action {
		if (ch != 0 && pr.action[i].ch == ch) ||
			(run != nil && pr.action[i].run == run) ||
			(mess != "" && pr.action[i].mess == mess) {
			pr.action = append(pr.action[:i], pr.action[i+1:]...)
			return true
		}
	}
	return false
}

// SetActionMess updates the message on an existing action, matched by run or
// runner; it returns true only if an action was updated.
func (pr *Prompt) SetActionMess(ch rune, run Runner, mess string, args ...interface{}) bool {
	if len(args) > 0 {
		mess = fmt.Sprintf(mess, args...)
	}
	for i := range pr.action {
		if (ch != 0 && pr.action[i].ch == ch) ||
			(run != nil && pr.action[i].run == run) {
			pr.action[i].mess = mess
			return true
		}
	}
	return false
}

// RunPrompt runs the prompt as a sub-prompt of another; causes Prompt to
// implement Runner, allowing prompts to be added as actions to other prompts.
func (pr Prompt) RunPrompt(prior Prompt) Prompt {
	return Prompt{
		prior:  &prior,
		align:  prior.align,
		mess:   pr.mess,
		action: pr.action,
	}
}
