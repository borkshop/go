// +build js

package main

import (
	"borkshop/stats"
	"log"
	"math"
	"syscall/js"
	"time"
)

type animator interface {
	animate(now float64)
}

type animatorFunc func(now float64)

func (af animatorFunc) animate(now float64) { af(now) }

type frameAnimator struct {
	last float64
	animator
	fn js.Func

	rafTimes    stats.Durations
	clientTimes stats.Durations
}

func (anim *frameAnimator) Init(client animator) {
	const timingWindow = 240
	const reportEvery = 0
	anim.rafTimes.Init(timingWindow, reportEvery, func(db *stats.Durations, i int) {
		log.Printf("average frame ∂: %v", db.Average())
	})
	anim.clientTimes.Init(timingWindow, reportEvery, func(db *stats.Durations, i int) {
		log.Printf("average client elapsed: %v", db.Average())
	})
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
		now := args[0].Float()
		elapsed := now - anim.last
		elapsedT := time.Duration(math.Round(elapsed*1000)) * time.Microsecond
		anim.rafTimes.Collect(elapsedT)

		t0 := time.Now()
		anim.animate(now) // TODO pass elapsedT
		t1 := time.Now()
		anim.clientTimes.Collect(t1.Sub(t0))

		anim.request()
		anim.last = now
	}
	return nil
}

func (anim *frameAnimator) Release() {
	anim.animator = nil
	anim.fn.Release()
}
