package anansi

import (
	"os"
)

// NewTerm constructs a new terminal attached the given file pair, and with the
// given context.
func NewTerm(in, out *os.File, cs ...Context) *Term {
	term := &Term{}
	term.Input.File = in
	term.Output.File = out
	_ = term.AddContext(cs...)
	return term
}

// Term combines a terminal file handle with attribute control and further
// Context-ual state.
type Term struct {
	Attr
	Mode
	Input
	Output

	active bool
	ctx    Context
}

// AddContext to a terminal, Enter()-ing them if it is already active.
func (term *Term) AddContext(cs ...Context) error {
	term.initContext()
	if ctx := Contexts(cs...); ctx != nil {
		if term.active {
			if err := ctx.Enter(term); err != nil {
				_ = ctx.Exit(term)
				return err
			}
		}
		term.ctx = Contexts(term.ctx, ctx)
	}
	return nil
}

func (term *Term) initContext() {
	if term.ctx == nil {
		term.ctx = Contexts(
			&term.Input,
			&term.Output,
			&term.Attr,
			&term.Mode)
	}
}

// RunWith runs the given function within the terminal's context, Enter()ing it
// if necessary, and Exit()ing it if Enter() was called after the given
// function returns. Exit() is called even if the within function returns an
// error or panics.
func (term *Term) RunWith(within func(*Term) error) (err error) {
	if term.active {
		return within(term)
	}
	term.initContext()

	term.active = true
	defer func() {
		term.active = false
	}()

	defer func() {
		if cerr := term.ctx.Exit(term); cerr == nil {
			err = cerr
		}
		if cl, ok := term.ctx.(interface{ Close() error }); ok {
			if cerr := cl.Close(); err == nil {
				err = cerr
			}
		}
	}()
	if err = term.ctx.Enter(term); err == nil {
		err = within(term)
	}
	return err
}

// RunWithout runs the given function without the terminal's context, Exit()ing
// it if necessary, and Enter()ing it if deactivation was necessary.
// Re-Enter() is not called is not done if a non-nil error is returned, or if
// the without function panics.
func (term *Term) RunWithout(without func(*Term) error) (err error) {
	if !term.active {
		return without(term)
	}
	if err = term.ctx.Exit(term); err == nil {
		term.active = false
		if err = without(term); err == nil {
			if err = term.ctx.Enter(term); err == nil {
				term.active = true
			}
		}
	}
	return err
}
