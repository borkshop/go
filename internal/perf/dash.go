package perf

import (
	"fmt"
	"unicode/utf8"

	"github.com/jcorbin/execs/internal/point"
	"github.com/jcorbin/execs/internal/view"
)

// Dash is a summary widget that can be triggered to show a perf dialog.
type Dash struct {
	*Perf
}

// HandleKey handles key input for the perf dashboard.
func (da Dash) HandleKey(k view.KeyEvent) bool {
	switch k.Ch {
	case '*':
		da.Perf.shouldProfile = !da.Perf.shouldProfile
		return true
	}
	return false
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
	x := 0
	g.Set(x, 0, da.status(), 0, 0)
	x++
	g.WriteString(x, 0, "t=%d Δt=%v heap=%v/%v",
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
