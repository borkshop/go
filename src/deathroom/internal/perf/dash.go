package perf

import (
	"fmt"
	"unicode/utf8"

	"deathroom/internal/point"
	"deathroom/internal/view"

	"github.com/jcorbin/anansi/ansi"
	"github.com/jcorbin/anansi/x/platform"
)

// Dash is a summary widget that can be triggered to show a perf dialog.
type Dash struct {
	*Perf
}

// HandleInput processes input for the perf dashboard.
func (da Dash) HandleInput(ctx view.Context, input platform.Events) error {
	if input.CountRune('*')%2 == 1 {
		da.Perf.shouldProfile = !da.Perf.shouldProfile
	}
	return nil
}

// RenderSize calculates the wanted/needed size render the dashboard.
func (da Dash) RenderSize() (wanted, needed point.Point) {
	i := da.lastI()
	lastElapsed := da.Perf.time[i].end.Sub(da.Perf.time[i].start)
	ms := &da.Perf.memStats[i]
	needed.X += utf8.RuneCountInString(fmt.Sprintf("t=%d Δt=%v", da.Perf.round, lastElapsed))
	needed.Y = 1
	wanted = needed
	wanted.X += utf8.RuneCountInString(fmt.Sprintf(" heap=%v/%v", siBytes(ms.HeapAlloc), ms.HeapObjects))
	return wanted, needed
}

// Render the dashboard.
func (da Dash) Render(g view.Grid) {
	i := da.lastI()
	lastElapsed := da.Perf.time[i].end.Sub(da.Perf.time[i].start)
	ms := &da.Perf.memStats[i]
	pt := ansi.Pt(1, 1)
	if i, ok := g.CellOffset(pt); ok {
		g.Rune[i] = da.status()
		g.Attr[i] = 0
	}
	pt.X++
	g.WriteString(pt, "t=%d Δt=%v heap=%v/%v",
		da.Perf.round, lastElapsed,
		siBytes(ms.HeapAlloc), ms.HeapObjects,
	)
}

func (da Dash) lastI() int {
	i := da.Perf.i - 1
	if i < 0 {
		i += numSamples
	}
	return i
}

func (da Dash) status() rune {
	if da.Perf.err != nil {
		return '■'
	}
	if da.Perf.profiling {
		return '◉'
	}
	if da.Perf.shouldProfile {
		return '◎'
	}
	return '○'
}

func siBytes(n uint64) string {
	if n < 1024 {
		return fmt.Sprintf("%vB", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1fKiB", float64(n)/1024.0)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1fMiB", float64(n)/(1024.0*1024.0))
	}
	return fmt.Sprintf("%.1fGiB", float64(n)/(1024.0*1024.0*1024.0))
}
