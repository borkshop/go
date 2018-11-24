package hud

import (
	"fmt"
	"unicode/utf8"

	"deathroom/internal/moremath"
	"deathroom/internal/point"
	"deathroom/internal/view"

	"github.com/jcorbin/anansi/ansi"
)

// Logs represents a renderable buffer of log messages.
type Logs struct {
	Buffer   []string
	Align    view.Align
	Min, Max int
}

// Init initializes the log buffer and metadata, allocating the given capacity.
func (logs *Logs) Init(logCap int) {
	logs.Align = view.AlignTop | view.AlignCenter
	logs.Min = 5
	logs.Max = 10
	logs.Buffer = make([]string, 0, logCap)
}

// RenderSize returns the desired and necessary sizes for rendering.
func (logs Logs) RenderSize() (wanted, needed point.Point) {
	needed.X = 1
	needed.Y = moremath.MinInt(len(logs.Buffer), logs.Min)
	wanted.X = 1
	wanted.Y = moremath.MinInt(len(logs.Buffer), logs.Max)
	for i := range logs.Buffer {
		if n := utf8.RuneCountInString(logs.Buffer[i]); n > wanted.X {
			wanted.X = n
		}
	}
	if needed.Y > wanted.Y {
		needed.Y = wanted.Y
	}
	return wanted, needed
}

// Render renders the log buffer.
func (logs Logs) Render(g view.Grid) {
	off := len(logs.Buffer) - g.Bounds().Dy()
	if off < 0 {
		off = 0
	}
	for i, pt := off, ansi.Pt(1, 1); i < len(logs.Buffer); i, pt = i+1, ansi.Pt(1, pt.Y+1) {
		g.WriteString(pt, logs.Buffer[i])
	}
}

// Log formats and appends a log message to the buffer, discarding the oldest
// message if full.
func (logs *Logs) Log(mess string, args ...interface{}) {
	mess = fmt.Sprintf(mess, args...)
	if len(logs.Buffer) < cap(logs.Buffer) {
		logs.Buffer = append(logs.Buffer, mess)
	} else {
		copy(logs.Buffer, logs.Buffer[1:])
		logs.Buffer[len(logs.Buffer)-1] = mess
	}
}
