package view

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jcorbin/anansi"
	"github.com/jcorbin/anansi/x/platform"
)

type syntheticSignal string

func (ss syntheticSignal) String() string { return string(ss) }
func (ss syntheticSignal) Signal()        {}

var (
	ctrlCSignal = syntheticSignal("user Ctrl-C interrupt")
	sizeSignal  = syntheticSignal("acquire size")
	quitSignal  = syntheticSignal("user quit")
)

// View implements terminal user interaction, combining anansi.Input,
// anansi.Output, signal processing, and other common terminal idioms (like
// redraw on Ctrl-L, stop on Ctrl-C, etc).
//
// Screen layout is organized into a header, footer, and min grid area.
//
// A log is provided, whose tail is displayed beneath the header.
//
// TODO consider replacing with loosely coupled anansi.Context pieces.
type View struct {
	termEvents
	term   *anansi.Term
	events platform.Events
	out    anansi.Output
	screen anansi.Screen
	tick   *time.Timer
}

const renderDelay = 10 * time.Millisecond

// Enter sets up input event notification, and starts the render timer.
func (v *View) Enter(term *anansi.Term) error {
	if v.term != nil {
		return errors.New("view already active")
	}
	v.term = term
	err := v.events.Notify(v.sigio)
	if err == nil {
		raiseSignal(v.sigwinch, sizeSignal)
	}
	return err
}

// Exit stops the render timer.
func (v *View) Exit(term *anansi.Term) error {
	v.term = nil
	if v.tick != nil {
		v.tick.Stop()
		v.tick = nil
	}
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

func (v *View) runClient(client Client) (rerr error) {
	type initable interface{ Init(Context) error }
	type terminatable interface{ Terminate() error }
	type interruptable interface{ Interrupt() error }
	type closeable interface{ Close() error }

	log.Printf("running client %T %v", client, client)

	if closer, ok := client.(closeable); ok {
		log.Printf("deferring closer.Close")
		defer func() {
			if cerr := closer.Close(); cerr != nil {
				if rerr == nil || rerr == ErrStop || rerr == io.EOF {
					rerr = cerr
				}
			}
		}()
	}

	// TODO: observability / introspection / other Nice To Haves? (reconcile with anansi/x/platform)

	if initr, ok := client.(initable); ok {
		log.Printf("running initable.Init")
		if err := initr.Init(v); err != nil {
			return err
		}
		// NOTE client must request first frame when it implement init
	} else {
		log.Printf("requesting initial frame")
		v.RequestFrame(renderDelay)
	}

	/* TODO: punch list
	 * - async sigio polling is busted
	 * - color problem with floor tiles at least; TBD where
	 */
	for {
		select {
		case <-v.sigterm:
			log.Printf("sigterm")
			if termr, ok := client.(terminatable); ok {
				return termr.Terminate()
			}
			return io.EOF // TODO better error?

		case <-v.sigint:
			log.Printf("sigint")
			if intr, ok := client.(interruptable); ok {
				return intr.Interrupt()
			}
			return io.EOF // TODO better error?

		case <-v.sigwinch:
			log.Printf("sigwinch")
			sz, err := v.term.Size()
			if err != nil {
				return err
			}
			v.screen.Resize(sz)
			v.RequestFrame(renderDelay)

		case <-v.sigio:
			log.Printf("sigio")
			if err := v.events.Poll(); err != nil {
				return err
			}
			log.Printf("polled input %v", dumpEvents(&v.events))

			// synthesize interrupt on Ctrl-C
			if v.events.CountRune(0x03) > 0 {
				log.Printf("Ctrl-C -> sigint")
				raiseSignal(v.sigint, ctrlCSignal)
			}

			// force full redraw on Ctrl-L
			if v.events.CountRune(0x0c) > 0 {
				log.Printf("Ctrl-L -> invalidate")
				v.screen.Invalidate()
				v.RequestFrame(renderDelay)
			}

			// quit on Q
			if n := v.events.CountRune('q', 'Q'); n > 0 {
				log.Printf("Quit -> sigterm")
				raiseSignal(v.sigterm, quitSignal)
			}

			// pass remaining input to client
			log.Printf("handling input %v", dumpEvents(&v.events))
			if err := client.HandleInput(v, v.events); err != nil {
				return err
			}

		case <-v.tick.C:
			log.Printf("tick")
			// clear screen grid
			for i := range v.screen.Rune {
				v.screen.Grid.Rune[i] = 0
				v.screen.Grid.Attr[i] = 0
			}

			// render the client
			// TODO revamp the client contract:
			// - pass it the screen directly...
			// - ...let it decide to (or not) clear the grid
			err := client.Render(v, Grid{v.screen.Grid})

			// flush output, differentially when possible
			if ferr := v.out.Flush(&v.screen); err == nil {
				err = ferr
			}

			if err != nil {
				return err
			}
		}
	}
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

type termEvents struct {
	sigterm  chan os.Signal
	sigint   chan os.Signal
	sigwinch chan os.Signal
	sigio    chan os.Signal
}

func (tev *termEvents) Enter(term *anansi.Term) error {
	tev.sigterm = make(chan os.Signal, 1)
	tev.sigint = make(chan os.Signal, 1)
	tev.sigwinch = make(chan os.Signal, 1)
	tev.sigio = make(chan os.Signal, 1)
	signal.Notify(tev.sigterm, syscall.SIGTERM)
	signal.Notify(tev.sigint, syscall.SIGINT)
	signal.Notify(tev.sigwinch, syscall.SIGWINCH)
	return nil
}

func (tev *termEvents) Exit(term *anansi.Term) error {
	if tev.sigterm != nil {
		signal.Stop(tev.sigterm)
		tev.sigterm = nil
	}
	if tev.sigint != nil {
		signal.Stop(tev.sigint)
		tev.sigint = nil
	}
	if tev.sigwinch != nil {
		signal.Stop(tev.sigwinch)
		tev.sigwinch = nil
	}
	if tev.sigio != nil {
		signal.Stop(tev.sigio)
		tev.sigio = nil
	}
	return nil
}

func raiseSignal(ch chan<- os.Signal, sig os.Signal) {
	select {
	case ch <- sig:
	default:
	}
}
