package view

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"
	termbox "github.com/nsf/termbox-go"
)

// ErrStop may be returned by a client method to mean "we're done, break run loop".
var ErrStop = errors.New("client stop")

// Context abstracts the platform under which a client is running.
//
// TODO say more; more methods; etc
// TODO support custom signal notification
type Context interface {
	RequestFrame(time.Duration)
}

// Client is the interface exposed to the user of View; its various methods are
// called in a loop that provides terminal orchestration.
//
// NOTE may optionally implement Terminate() error and Interrupt() error methods.
type Client interface {
	HandleInput(Context, platform.Events) error
	Render(Context, Grid) error
}

// KeyEvent represents a terminal key event.
type KeyEvent struct {
	Mod termbox.Modifier
	Key termbox.Key
	Ch  rune
}

// JustKeepRunning starts a view, and then running newly minted Clients
// provided by the given factory until an error occurs, or the user quits.
// Useful for implementing main.main.
func JustKeepRunning(factory func(v *View) (Client, error)) error {
	var v View
	return v.newTerm(os.Stdout).RunWith(func(term *anansi.Term) error {
		for {
			client, err := factory(&v)
			if err == nil {
				err = v.runClient(client)
			}
			switch err {
			case nil, ErrStop:
				continue
			case io.EOF:
				return nil
			default:
				return err
			}
		}
	})
}

// Run a Client under this view, returning any error from the run (may be
// caused by the client, or view).
func (v *View) Run(client Client) error {
	if v.term != nil {
		switch err := v.runClient(client); err {
		case nil, ErrStop, io.EOF:
			return nil
		default:
			return err
		}
	}

	return v.newTerm(os.Stdout).RunWith(func(term *anansi.Term) error {
		return v.Run(client)
	})
}

func (v *View) newTerm(f *os.File) *anansi.Term {
	term := anansi.NewTerm(f,
		&v.out, &v.events, &v.termEvents, v,
	)
	term.SetEcho(false)
	term.SetRaw(true)
	term.AddMode(

		// TODO if mouse enabled
		// ansi.ModeMouseSgrExt,
		// ansi.ModeMouseBtnEvent,
		// ansi.ModeMouseAnyEvent,

		ansi.ModeAlternateScreen,
	)
	return term
}
