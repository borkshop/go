package view

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"
)

type syntheticSignal string

func (ss syntheticSignal) String() string { return string(ss) }
func (ss syntheticSignal) Signal()        {}

var ctrlCSignal = syntheticSignal("user Ctrl-C interrupt")

// View implements terminal user interaction, combining anansi.Input,
// anansi.Output, signal processing, and other common terminal idioms (like
// redraw on Ctrl-L, stop on Ctrl-C, etc).
//
// Screen layout is organized into a header, footer, and min grid area.
//
// A log is provided, whose tail is displayed beneath the header.
//
// TODO consider replacing with loosely coupled anansi.Context pieces.
//
// TODO observability / introspection / other Nice To Haves? (reconcile with anansi/x/platform)
type View struct {
	term     *anansi.Term
	sigterm  anansi.Signal
	sigint   anansi.Signal
	sigwinch anansi.Signal
	sigio    anansi.Signal
	screen   anansi.Screen
	events   platform.Events
	tick     *time.Timer
}

const renderDelay = 10 * time.Millisecond

// Enter sets up input event notification, and starts the render timer.
func (v *View) Enter(term *anansi.Term) error {
	if v.term != nil {
		return errors.New("view already active")
	}
	v.term = term
	v.RequestFrame(0)
	return nil
}

// Exit stops the render timer.
func (v *View) Exit(term *anansi.Term) error {
	if v.tick != nil {
		v.tick.Stop()
		v.tick = nil
	}
	v.term = nil
	return nil
}

// RequestFrame sets the frame render timer to fire after dur time has elapsed.
func (v *View) RequestFrame(dur time.Duration) {
	if v.tick == nil {
		v.tick = time.NewTimer(dur)
	} else {
		// TODO track prior deadline, no-op if will already fire before now+dur?
		v.tick.Reset(renderDelay)
	}
}

// Debugf logs a debug message.
func (v *View) Debugf(mess string, args ...interface{}) {
	// TODO standard annotations
	// TODO log control / toggle
	log.Printf(mess, args...)
}

// Infof logs an info message.
func (v *View) Infof(mess string, args ...interface{}) {
	// TODO standard annotations
	// TODO log control / toggle
	log.Printf(mess, args...)
}

type viewApp interface {
	Init(Context) error
	Close(Context) error
	Terminate(Context, os.Signal) error
	Interrupt(Context, os.Signal) error
	Resized(Context, os.Signal) error
	HandleInput(ctx Context, input *platform.Events) error
	Render(ctx Context, t time.Time, screen *anansi.Screen) error
}

func (v *View) run(app viewApp) error {
	ctx := Context(v)
	err := app.Init(ctx)
	if err != nil {
		return nil
	}
	defer func() {
		isStop, _, _ := isStopErr(err)
		if cerr := app.Close(ctx); isStop {
			err = cerr
		}
	}()
	for err == nil {
		select {
		case sig := <-v.sigterm.C:
			err = app.Terminate(ctx, sig)
		case sig := <-v.sigint.C:
			err = app.Interrupt(ctx, sig)
		case sig := <-v.sigwinch.C:
			err = app.Resized(ctx, sig)
		case sig := <-v.sigio.C:
			ctx.Debugf("input ready: %v", sig)
			err = v.processInput(ctx, app)
		case t := <-v.tick.C:
			ctx.Debugf("tick t:%v", t)
			err = v.render(ctx, t, app)
		}
	}
	return err
}

func (v *View) processInput(ctx Context, app viewApp) error {
	if err := v.events.Poll(); err != nil {
		return err
	}
	if haveAnyInput(&v.events) {
		ctx.Debugf("polled input: %v", dumpEvents(&v.events))
		return v.handleInput(ctx, app)
	}
	return nil
}

func (v *View) handleInput(ctx Context, app viewApp) error {
	// synthesize interrupt on Ctrl-C
	if v.events.CountRune(0x03) > 0 {
		ctx.Infof("Ctrl-C -> sigint")
		raiseSignal(v.sigint.C, ctrlCSignal)
	}

	// force full redraw on Ctrl-L
	if v.events.CountRune(0x0c) > 0 {
		ctx.Infof("Ctrl-L -> invalidate")
		v.screen.Invalidate()
		ctx.RequestFrame(renderDelay)
	}

	// pass remaining input to app
	if haveAnyInput(&v.events) {
		return app.HandleInput(ctx, &v.events)
	}
	return nil
}

func (v *View) render(ctx Context, t time.Time, app viewApp) error {
	err := app.Render(ctx, t, &v.screen)
	if ferr := v.term.Flush(&v.screen); err == nil {
		err = ferr
	}
	return err
}

func (v *View) newTerm(in, out *os.File) (*anansi.Term, error) {
	v.sigterm = anansi.Notify(syscall.SIGTERM)
	v.sigint = anansi.Notify(syscall.SIGINT)
	v.sigwinch = anansi.Notify(syscall.SIGWINCH)

	term := anansi.NewTerm(in, out,
		&v.sigterm,
		&v.sigint,
		&v.sigwinch,
		&v.screen,
		v,
	)
	_ = term.SetEcho(false)
	_ = term.SetRaw(true)
	term.AddMode(

		// TODO if mouse enabled
		// ansi.ModeMouseSgrExt,
		// ansi.ModeMouseBtnEvent,
		// ansi.ModeMouseAnyEvent,

		ansi.ModeAlternateScreen,
	)

	v.events.Input = &term.Input
	v.sigio.C = make(chan os.Signal, 1)
	if err := term.Notify(v.sigio.C); err != nil {
		return nil, err
	}

	return term, nil
}

func raiseSignal(ch chan<- os.Signal, sig os.Signal) {
	select {
	case ch <- sig:
	default:
	}
}

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

func haveAnyInput(es *platform.Events) bool {
	for i := 0; i < len(es.Type); i++ {
		if es.Type[i] != platform.EventNone {
			return true
		}
	}
	return false
}

func dumpEvents(es *platform.Events) []string {
	ss := make([]string, 0, len(es.Type))
	for i := 0; i < len(es.Type); i++ {
		switch es.Type[i] {
		case platform.EventRune:
			ss = append(ss, fmt.Sprintf("%q", es.Rune(i)))
		case platform.EventEscape:
			ss = append(ss, es.Escape(i).String())
		case platform.EventMouse:
			ss = append(ss, es.Mouse(i).String())
		}
	}
	return ss
}
