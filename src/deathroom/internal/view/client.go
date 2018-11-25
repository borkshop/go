package view

import (
	"errors"
	"io"
	"log"
	"os"
	"time"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"
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
	HandleInput(ctx Context, input platform.Events) error
	Render(ctx Context, viewport Grid) error
}

// JustKeepRunning starts a view, and then running newly minted Clients
// provided by the given factory until an error occurs, or the user quits.
// Useful for implementing main.main.
func JustKeepRunning(factory func(v *View) (Client, error)) error {
	var v View
	log.Printf("view run loop: creating terminal")
	return v.newTerm(os.Stdout).RunWith(func(term *anansi.Term) error {
		for {
			log.Printf("view run loop: creating client")
			client, err := factory(&v)
			if err == nil {
				log.Printf("view run loop: running client")
				err = v.runClient(client)
			}

			if stop, halt, rerr := isStopErr(err); halt {
				log.Printf("view run loop: terminal client error: %v", err)
				return rerr
			} else if stop {
				log.Printf("view run loop: client stopped: %v", err)
			}
		}
	})
}

// Run a Client under this view, returning any error from the run (may be
// caused by the client, or view).
func (v *View) Run(client Client) error {
	if v.term != nil {
		log.Printf("view run: running client")
		err := v.runClient(client)
		stop, halt, rerr := isStopErr(err)
		if halt {
			log.Printf("view run: terminal client error: %v", err)
		} else if stop {
			log.Printf("view run: client stopped: %v", err)
		}
		return rerr
	}

	return v.newTerm(os.Stdout).RunWith(func(term *anansi.Term) error {
		log.Printf("view run: creating terminal")
		return v.Run(client)
	})
}

type clientSignaledError struct {
	sig  os.Signal
	term bool
}

func clientTerminalError(sig os.Signal) clientSignaledError { return clientSignaledError{sig, true} }
func clientStopError(sig os.Signal) clientSignaledError     { return clientSignaledError{sig, false} }

func (sigErr clientSignaledError) Error() string { return sigErr.sig.String() }

func isStopErr(err error) (stop, halt bool, _ error) {
	switch err {
	case io.EOF:
		return true, true, nil
	case ErrStop:
		return true, false, nil
	case nil:
		return false, false, nil
	}
	switch impl := err.(type) {
	case clientSignaledError:
		return true, impl.term, nil
	}
	return true, true, err
}

func (v *View) newTerm(f *os.File) *anansi.Term {
	term := anansi.NewTerm(f,
		&v.termEvents,
		&v.events,
		&v.out,
		v,
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
