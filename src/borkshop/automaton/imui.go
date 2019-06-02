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

	input  rune
	screen *image.RGBA
	info   bytes.Buffer

	// bindings
	anim        frameAnimator
	canvas      js.Value
	infoOverlay js.Value
	renderCtx   js.Value

	done chan error
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

	// TODO observe keydown/up instead of presses directly
	parent.Call("addEventListener", "keypress", js.FuncOf(ctx.onKeyPress))

	// TODO proper grid size calc
	size := image.Pt(
		parent.Get("clientWidth").Int(),
		parent.Get("clientHeight").Int(),
	)
	ctx.screen = image.NewRGBA(image.Rect(0, 0, size.X, size.Y))
	ctx.done = make(chan error)

	ctx.anim.Init(ctx)

	return nil
}

func (ctx *imContext) onKeyPress(this js.Value, args []js.Value) interface{} {
	event := args[0]
	ctx.clearInput()
	for _, r := range event.Get("key").String() {
		ctx.input = r
		break
	}
	ctx.Update(ctx.client)
	ctx.Render()
	return nil
}

func (ctx *imContext) animate(now float64) {
	ctx.clearInput()
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
	ctx.clearOutput()
	if err := client.Update(ctx); err != nil {
		ctx.done <- err
	}
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

func (ctx *imContext) infof(mess string, args ...interface{}) {
	_, _ = fmt.Fprintf(&ctx.info, mess, args...)
}

func (ctx *imContext) clearInput() {
	ctx.input = 0
}

func (ctx *imContext) clearOutput() {
	for i := range ctx.screen.Pix {
		ctx.screen.Pix[i] = 0
	}
	ctx.info.Reset()
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
