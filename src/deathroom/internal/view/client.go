package view

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/jcorbin/anansi"
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

	// TODO proper logging interface
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
}

// Client is the interface exposed to the user of View; its various methods are
// called in a loop that provides terminal orchestration.
//
// NOTE may optionally implement Terminate() error and Interrupt() error methods.
type Client interface {
	HandleInput(ctx Context, input *platform.Events) error
	Render(ctx Context, t time.Time, screen *anansi.Screen) error
}

// JustKeepRunning starts a view, and then running newly minted Clients
// provided by the given factory until an error occurs, or the user quits.
// Useful for implementing main.main.
func JustKeepRunning(factory func(v *View) (Client, error)) error {
	var v View
	log.Printf("view run loop: creating terminal")
	term, err := v.newTerm(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	return term.RunWith(func(term *anansi.Term) error {
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
	term, err := v.newTerm(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	return term.RunWith(func(term *anansi.Term) error {
		log.Printf("view run: creating terminal")
		return v.Run(client)
	})
}

func (v *View) runClient(client Client) error {
	return v.run(clientView{v, client})
}

type clientView struct {
	*View
	Client
}

type clientInit interface {
	Client
	Init(Context) error
}

func (cv clientView) Init(ctx Context) error {
	ctx.Infof("running %T client", cv.Client)
	if initr, ok := cv.Client.(clientInit); ok {
		ctx.Debugf("initializing %T client", cv.Client)
		if err := initr.Init(ctx); err != nil {
			return err
		}
		// NOTE client must request first frame when it implement init
	} else {
		ctx.Debugf("requesting initial frame")
		ctx.RequestFrame(renderDelay)
	}
	return nil
}

type clientClose interface {
	Client
	Close(Context) error
}

func (cv clientView) Close(ctx Context) error {
	if closer, ok := cv.Client.(clientClose); ok {
		ctx.Debugf("closing %T client", cv.Client)
		return closer.Close(ctx)
	}
	return nil
}

type clientTerminate interface {
	Client
	Terminate(Context, os.Signal) error
}

func (cv clientView) Terminate(ctx Context, sig os.Signal) error {
	if termr, ok := cv.Client.(clientTerminate); ok {
		ctx.Infof("terminate %T client: %v", cv.Client, sig)
		return termr.Terminate(ctx, sig)
	}
	cv.Infof("terminate: %v", sig)
	return clientTerminalError(sig)
}

type clientInterrupt interface {
	Client
	Interrupt(Context, os.Signal) error
}

func (cv clientView) Interrupt(ctx Context, sig os.Signal) error {
	if intr, ok := cv.Client.(clientInterrupt); ok {
		ctx.Infof("interrupt %T client: %v", cv.Client, sig)
		return intr.Interrupt(ctx, sig)
	}
	cv.Infof("interrupt: %v", sig)
	return clientTerminalError(sig)
}

func (cv clientView) Resized(ctx Context, sig os.Signal) error {
	ctx.Debugf("resized: %v", sig)
	err := cv.screen.SizeToTerm(cv.term)
	if err == nil {
		ctx.RequestFrame(renderDelay)
	}
	return err
}

type clientSignaledError struct {
	sig  os.Signal
	term bool
}

func clientTerminalError(sig os.Signal) clientSignaledError { return clientSignaledError{sig, true} }

func (sigErr clientSignaledError) Error() string { return sigErr.sig.String() }
