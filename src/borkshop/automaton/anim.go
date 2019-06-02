package main

// +build js

import (
	"syscall/js"
)

type animator interface {
	animate(now float64)
}

type animatorFunc func(now float64)

func (af animatorFunc) animate(now float64) { af(now) }

type frameAnimator struct {
	animator
	fn js.Func
}

func (anim *frameAnimator) Init(client animator) {
	anim.animator = client
	anim.request()
}

func (anim *frameAnimator) InitFunc(af func(now float64)) {
	anim.Init(animatorFunc(af))
}

func (anim *frameAnimator) request() {
	if !anim.fn.Truthy() {
		anim.fn = js.FuncOf(anim.callback)
	}
	js.Global().Call("requestAnimationFrame", anim.fn)
}

func (anim *frameAnimator) callback(this js.Value, args []js.Value) interface{} {
	if anim.animator != nil {
		anim.animate(args[0].Float())
		anim.request()
	}
	return nil
}

func (anim *frameAnimator) Release() {
	anim.animator = nil
	anim.fn.Release()
}
