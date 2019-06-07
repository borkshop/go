// +build js

package main

import (
	"borkshop/stats"
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"syscall/js"
	"time"
)

var (
	document          = js.Global().Get("document")
	window            = js.Global().Get("window")
	ImageData         = js.Global().Get("ImageData")
	Uint8ClampedArray = js.Global().Get("Uint8ClampedArray")
)

const timingWindow = 4 * 60

type imClient interface {
	Update(*imContext) error
}

type imContext struct {
	client imClient

	// timing
	updateTimes  stats.Durations
	clientTimes  stats.Durations
	renderTimes  stats.Durations
	elapsedTimes stats.Durations

	// TODO animation/simulation time
	imInput
	imOutput

	// animation
	lastFrame time.Time
	rafFn     js.Func

	// dom bindings
	canvas    js.Value
	renderCtx js.Value

	infoDetails js.Value
	infoBody    js.Value

	profTiming  bool
	profDetails js.Value
	profTitle   js.Value
	profBody    js.Value

	// run done
	done chan error
}

type keyMod uint8

const (
	altKey keyMod = 1 << iota
	ctrlKey
	metaKey
	shiftKey
)

func readKeyMod(event js.Value) keyMod {
	var mod keyMod
	if event.Get("altKey").Bool() {
		mod |= altKey
	}
	if event.Get("ctrlKey").Bool() {
		mod |= ctrlKey
	}
	if event.Get("shiftKey").Bool() {
		mod |= metaKey
	}
	if event.Get("metaKey").Bool() {
		mod |= shiftKey
	}
	return mod
}

type imInput struct {
	key struct {
		press rune
		mod   keyMod
		// TODO down buttons
	}
	// TODO mouse struct {}
}

type imOutput struct {
	screen *image.RGBA // TODO clarify screen-space vs cell-space
	prof   bytes.Buffer
	info   bytes.Buffer
}

func (ctx *imContext) Run(client imClient) error {
	ctx.client = client
	err := ctx.init()
	defer ctx.release()
	if err == nil {
		err = <-ctx.done
	}
	return err
}

func (ctx *imContext) init() (err error) {
	ctx.canvas, err = getEnvSelector("canvas")
	if err != nil {
		return err
	}

	ctx.infoDetails, err = getEnvSelector("info-details")
	if err != nil {
		return err
	}

	ctx.profDetails, err = getEnvSelector("prof-details")
	if err != nil {
		return err
	}

	ctx.updateTimes = stats.MakeDurations(timingWindow)
	ctx.renderTimes = stats.MakeDurations(timingWindow)
	ctx.clientTimes = stats.MakeDurations(timingWindow)
	ctx.elapsedTimes = stats.MakeDurations(timingWindow)

	ctx.rafFn = js.FuncOf(ctx.onFrame)

	ctx.infoBody = ctx.infoDetails.Call("appendChild", document.Call("createElement", "pre"))
	ctx.profTitle = ctx.profDetails.Call("querySelector", "summary")
	ctx.profBody = ctx.profDetails.Call("appendChild", document.Call("createElement", "pre"))

	// TODO webgl instead
	// TODO initialize cell rendering gl program
	ctx.renderCtx = ctx.canvas.Call("getContext", "2d")

	parent := ctx.canvas.Get("parentNode")
	parent.Call("addEventListener", "keypress", js.FuncOf(ctx.onKeyPress))
	window.Call("addEventListener", "resize", js.FuncOf(ctx.onResize))

	ctx.done = make(chan error)

	ctx.requestFrame()
	ctx.updateSize()

	return nil
}

func (ctx *imContext) requestFrame() {
	js.Global().Call("requestAnimationFrame", ctx.rafFn)
}

func (ctx *imContext) onFrame(this js.Value, args []js.Value) interface{} {
	millisec := args[0].Float()
	sec := int64(millisec / 1000)
	microsec := int64(math.Round(math.Mod(millisec, 1000) * 1000))

	now := time.Unix(sec, microsec*1000)

	var elapsed time.Duration
	if !ctx.lastFrame.IsZero() {
		elapsed = now.Sub(ctx.lastFrame)
		ctx.elapsedTimes.Collect(elapsed)
	}

	ctx.animate(elapsed)
	ctx.requestFrame()
	ctx.lastFrame = now
	return nil
}

func (ctx *imContext) onResize(this js.Value, args []js.Value) interface{} {
	ctx.updateSize()
	ctx.Update()
	return nil
}

func (ctx *imContext) updateSize() {
	parent := ctx.canvas.Get("parentNode")
	size := image.Pt(
		parent.Get("clientWidth").Int(),
		parent.Get("clientHeight").Int(),
	)

	// TODO decouple grid size from screen size

	// TODO reuse prior capacity when possible
	ctx.screen = image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
}

func (ctx *imContext) onKeyPress(this js.Value, args []js.Value) interface{} {
	ctx.imInput.onKeyPress(this, args)
	ctx.Update()
	return nil
}

func (ctx *imContext) animate(elapsed time.Duration) {
	// TODO inject elapsed time to derive animation/simulation step
	ctx.Update()
	ctx.Render()
}

func (ctx *imContext) release() {
	ctx.rafFn.Release()
}

func (ctx *imContext) Update() {
	defer ctx.updateTimes.Measure()()

	// clear output so that client may rebuild it
	ctx.clearOutput()

	if ctx.key.press == 'p' && ctx.key.mod == ctrlKey {
		ctx.clearInput()
		ctx.profTiming = !ctx.profTiming
	}

	if ctx.profTiming {
		ctx.proff("µ update: %v\n", ctx.updateTimes.Average())
		ctx.proff("µ client: %v\n", ctx.clientTimes.Average())
		ctx.proff("µ render: %v\n", ctx.renderTimes.Average())
		ctx.proff("µ 𝝙frame: %v\n", ctx.elapsedTimes.Average())
	}

	ctx.updateClient()

	// clear one-shot input that's now been processed by the client
	ctx.clearInput()
}

func (ctx *imContext) updateClient() {
	defer ctx.clientTimes.Measure()()
	if err := ctx.client.Update(ctx); err != nil {
		ctx.done <- err
	}
}

func (ctx *imContext) Render() {
	defer ctx.renderTimes.Measure()()

	// update profiling details
	if ctx.prof.Len() == 0 {
		ctx.profDetails.Get("style").Set("display", "none")
		ctx.profTitle.Set("innerText", "")
		ctx.profBody.Set("innerText", "")
	} else {
		ctx.profDetails.Get("style").Set("display", "")
		if ctx.profDetails.Get("open").Bool() {
			ctx.profTitle.Set("innerText", "")
			ctx.profBody.Set("innerText", ctx.prof.String())
		} else {
			b := ctx.prof.Bytes()
			if i := bytes.IndexByte(b, '\n'); i > 0 {
				b = b[:i]
			}
			ctx.profTitle.Set("innerText", string(b))
			ctx.profBody.Set("innerText", "")
		}
	}

	// update simulation info details
	ctx.infoBody.Set("innerText", ctx.info.String())

	// render the world grid
	size := ctx.screen.Rect.Size()
	ar := js.TypedArrayOf(ctx.screen.Pix)
	defer ar.Release()

	// TODO can we just retain this image object between renders?
	img := ImageData.New(Uint8ClampedArray.New(ar), size.X, size.Y)

	ctx.renderCtx.Call("putImageData", img, 0, 0)
}

func (in *imInput) clearInput() {
	in.key.press = 0
}

func (in *imInput) onKeyPress(this js.Value, args []js.Value) interface{} {
	event := args[0]
	in.key.mod = readKeyMod(event)
	for _, r := range event.Get("key").String() {
		in.key.press = r
		break
	}
	return nil
}

func (out *imOutput) clearOutput() {
	for i := range out.screen.Pix {
		out.screen.Pix[i] = 0
	}
	out.info.Reset()
	out.prof.Reset()
}

func (out *imContext) proff(mess string, args ...interface{}) {
	_, _ = fmt.Fprintf(&out.prof, mess, args...)
}

func (out *imContext) infof(mess string, args ...interface{}) {
	_, _ = fmt.Fprintf(&out.info, mess, args...)
}

func getEnvSelector(name string) (js.Value, error) {
	selector := os.Getenv(name)
	if selector == "" {
		return js.Value{}, fmt.Errorf("no $%s given", name)
	}
	el := document.Call("querySelector", os.Getenv(name))
	if !el.Truthy() {
		return js.Value{}, fmt.Errorf("no element selected by $%s=%q", name, selector)
	}
	return el, nil
}
