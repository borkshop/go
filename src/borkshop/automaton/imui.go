// +build js

package main

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"syscall/js"
)

var (
	document          = js.Global().Get("document")
	window            = js.Global().Get("window")
	ImageData         = js.Global().Get("ImageData")
	Uint8ClampedArray = js.Global().Get("Uint8ClampedArray")
)

type imClient interface {
	Update(*imContext) error
}

type imContext struct {
	client imClient

	// TODO animation/simulation time
	imInput
	imOutput

	// bindings
	anim        frameAnimator
	canvas      js.Value
	infoOverlay js.Value
	renderCtx   js.Value

	done chan error
}

type imInput struct {
	key struct {
		press rune
		// TODO down buttons
	}
	// TODO mouse struct {}
}

type imOutput struct {
	screen *image.RGBA // TODO clarify screen-space vs cell-space
	info   bytes.Buffer
}

func (ctx *imContext) Run(client imClient) error {
	err := ctx.Init(client)
	defer ctx.Release()
	if err == nil {
		err = ctx.Wait()
	}
	return err
}

func (ctx *imContext) Init(client imClient) (err error) {
	ctx.client = client

	ctx.canvas, err = getEnvSelector("canvas")
	if err != nil {
		return err
	}

	ctx.infoOverlay, err = getEnvSelector("info-overlay")
	if err != nil {
		return err
	}

	// TODO webgl instead
	// TODO initialize cell rendering gl program
	ctx.renderCtx = ctx.canvas.Call("getContext", "2d")

	parent := ctx.canvas.Get("parentNode")
	parent.Call("addEventListener", "keypress", js.FuncOf(ctx.onKeyPress))
	window.Call("addEventListener", "resize", js.FuncOf(ctx.onResize))

	ctx.done = make(chan error)
	ctx.anim.Init(ctx)

	ctx.updateSize()

	return nil
}

func (ctx *imContext) onResize(this js.Value, args []js.Value) interface{} {
	ctx.updateSize()
	ctx.Update(ctx.client)
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
	ctx.Update(ctx.client)
	return nil
}

func (ctx *imContext) animate(now float64) {
	// TODO inject animation/simulation time delta
	ctx.Update(ctx.client)
	ctx.Render()
}

func (ctx *imContext) Release() {
	ctx.anim.Release()

}

func (ctx *imContext) Wait() error {
	return <-ctx.done
}

func (ctx *imContext) Update(client imClient) {
	// clear output so that client may rebuild it
	ctx.imOutput.clear()

	if err := client.Update(ctx); err != nil {
		ctx.done <- err
	}

	// clear one-shot input that's now been processed by the client
	ctx.imInput.clear()
}

func (ctx *imContext) Render() {
	ctx.infoOverlay.Set("innerText", ctx.info.String())

	size := ctx.screen.Rect.Size()
	ar := js.TypedArrayOf(ctx.screen.Pix)
	defer ar.Release()

	// TODO can we just retain this image object between renders?
	img := ImageData.New(Uint8ClampedArray.New(ar), size.X, size.Y)

	ctx.canvas.Set("width", size.X)
	ctx.canvas.Set("height", size.Y)
	ctx.renderCtx.Call("putImageData", img, 0, 0)
}

func (in *imInput) clear() {
	in.key.press = 0
}

func (in *imInput) onKeyPress(this js.Value, args []js.Value) interface{} {
	event := args[0]
	for _, r := range event.Get("key").String() {
		in.key.press = r
		break
	}
	return nil
}

func (out *imOutput) clear() {
	for i := range out.screen.Pix {
		out.screen.Pix[i] = 0
	}
	out.info.Reset()
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
