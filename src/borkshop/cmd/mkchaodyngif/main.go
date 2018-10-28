package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"math"
	"math/rand"
	"os"
	"time"

	hsluv "github.com/hsluv/hsluv-go"
)

const (
	radius = 32
	size   = radius * 2
	mask   = size - 1
	whole  = 0xffff
	g      = 0x1fffffff
)

var center = image.Point{radius, radius}

type body struct {
	P image.Point
	V image.Point
	E int
}

func grav(a, b image.Point) image.Point {
	p := b.Sub(a)
	dis := int(math.Sqrt(float64(p.X*p.X + p.Y*p.Y)))
	return p.Mul(-g / dis / dis)
}

func bowl(p image.Point) image.Point {
	xx := -0xfff * p.X / whole * p.X / whole * p.X / whole
	yy := -0xfff * p.Y / whole * p.Y / whole * p.Y / whole
	fmt.Printf("XX %v\n", xx)
	return image.Pt(xx, yy)
}

func start() image.Point {
	return image.Pt(rand.Int()&whole, rand.Int()&whole)
}

func newColor(h, s, l int) color.Color {
	r, g, b := hsluv.HsluvToRGB(
		360*float64(h)/float64(whole),
		100*float64(s)/float64(whole),
		100*float64(l)/float64(whole),
	)
	return color.RGBA{
		uint8(r * 0xff),
		uint8(g * 0xff),
		uint8(b * 0xff),
		0xff,
	}
}

var (
	red   = newColor(0, whole, whole/2)
	green = newColor(whole/3, whole, whole/2)
	blue  = newColor(whole*2/3, whole, whole/2)
)

func newPalette() color.Palette {
	pal := make(color.Palette, 0)
	for h := 0; h < 3; h++ {
		for l := 0; l < 16; l++ {
			pal = append(pal, newColor(whole*h/3, whole, whole*l/16))
		}
	}
	return pal
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

const corral = 5

func run() error {
	rand.Seed(time.Now().UnixNano())

	var a, b, c body

	var imgs []*image.Paletted
	var delays []int

	rect := image.Rect(0, 0, size, size)
	pal := newPalette()

	a.P = start()
	b.P = start()
	c.P = start()

	for i := 0; i < 1000; i++ {
		fmt.Printf("AP %10v BP %10v CP %10v -- AV %10v BV %10v CV %10v\n", a.P, b.P, c.P, a.V, b.V, c.V)

		// Compute mutual forces.
		ab := a.V.Add(grav(a.P, b.P))
		bc := a.V.Add(grav(b.P, c.P))
		ca := a.V.Add(grav(c.P, a.P))

		// Apply forces to velocity.
		a.V = a.V.Add(ab).Sub(ca).Sub(a.P.Div(corral))
		b.V = b.V.Add(bc).Sub(ab).Sub(b.P.Div(corral))
		c.V = c.V.Add(ca).Sub(bc).Sub(c.P.Div(corral))

		// Bowl forces
		a.V = a.V.Add(bowl(a.P))
		b.V = b.V.Add(bowl(b.P))
		c.V = c.V.Add(bowl(c.P))

		// Move: apply velocity to position.
		a.P = a.P.Add(a.V)
		b.P = b.P.Add(b.V)
		c.P = c.P.Add(c.V)

		// Project positions into view.
		ap := a.P.Mul(size).Div(10 * whole).Add(center)
		bp := b.P.Mul(size).Div(10 * whole).Add(center)
		cp := c.P.Mul(size).Div(10 * whole).Add(center)
		fmt.Printf("AP %10v BP %10v CP %10v\n", ap, bp, cp)

		img := image.NewPaletted(rect, pal)
		imgs = append(imgs, img)
		delays = append(delays, 14)

		img.Set(ap.X, ap.Y, red)
		img.Set(bp.X, bp.Y, green)
		img.Set(cp.X, cp.Y, blue)
	}

	f, err := os.Create("chaodyn.gif")
	if err != nil {
		return err
	}
	defer f.Close()
	gif.EncodeAll(f, &gif.GIF{
		Image: imgs,
		Delay: delays,
	})

	return nil
}
