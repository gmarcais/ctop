package single

import (
	"fmt"
	"image"
	"math"
	"sync"

	ui "github.com/gizak/termui"
)

const cpuAxisWidth = 7 // five label columns, a space, and the vertical axis

type Cpu struct {
	*ui.Block
	hist FloatHist
	mu   sync.Mutex
}

func NewCpu() *Cpu {
	cpu := &Cpu{Block: ui.NewBlock(), hist: NewFloatHist(55)}
	cpu.BorderLabel = "CPU"
	cpu.Height = 12
	cpu.Width = defaultPanelWidth
	return cpu
}

func (w *Cpu) Update(val int) {
	if val < 0 { // unavailable samples and counter resets are not negative usage
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.hist.Append(float64(val))
}

func (w *Cpu) Buffer() ui.Buffer {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.Width < 2 || w.Height < 2 {
		return ui.NewBuffer()
	}
	buf := w.Block.Buffer()
	inner := w.InnerBounds()
	plot := image.Rect(inner.Min.X+cpuAxisWidth, inner.Min.Y, inner.Max.X, inner.Max.Y)
	if plot.Empty() || plot.Dy() < 2 {
		return buf
	}
	w.hist.Grow(plot.Dx())
	data := visibleHistory(w.hist.Data, plot.Dx())
	top := cpuScale(data)
	axes := ui.ThemeAttr("linechart.axes.fg")
	line := ui.ThemeAttr("linechart.line.fg")
	baseline := plot.Max.Y - 1
	axisX := plot.Min.X - 1
	for y := plot.Min.Y; y <= baseline; y++ {
		buf.Set(axisX, y, ui.Cell{Ch: '│', Fg: axes, Bg: w.Bg})
	}
	buf.Set(axisX, baseline, ui.Cell{Ch: '└', Fg: axes, Bg: w.Bg})
	// Use the bottom Braille dots for the zero line too. A zero-valued sample
	// overlays this exact glyph instead of floating above a separate axis.
	for x := plot.Min.X; x < plot.Max.X; x++ {
		buf.Set(x, baseline, ui.Cell{Ch: '⣀', Fg: axes, Bg: w.Bg})
	}
	ticks := 5
	if plot.Dy() < ticks {
		ticks = plot.Dy()
	}
	for i := 0; i < ticks; i++ {
		value := top * float64(i) / float64(ticks-1)
		y := baseline - int(math.Round(float64(i)*float64(plot.Dy()-1)/float64(ticks-1)))
		label := fmt.Sprintf("%5s", fmt.Sprintf("%g%%", value))
		for x, ch := range label {
			if x < cpuAxisWidth-2 {
				buf.Set(inner.Min.X+x, y, ui.Cell{Ch: ch, Fg: axes, Bg: w.Bg})
			}
		}
	}

	pixelHeight := plot.Dy()*4 - 1
	sampleY := func(value float64) int {
		value = math.Max(0, math.Min(top, value))
		return pixelHeight - int(math.Round(value/top*float64(pixelHeight)))
	}
	// One sample still occupies one terminal column. Interpolate only the
	// connecting stroke on the 2x4 Braille grid; retain the original samples.
	previousY := sampleY(data[0])
	cpuStroke(&buf, plot, 0, previousY, 1, previousY, line, w.Bg)
	for i := 1; i < len(data); i++ {
		y := sampleY(data[i])
		cpuStroke(&buf, plot, 2*i-1, previousY, 2*i+1, y, line, w.Bg)
		previousY = y
	}
	return buf
}

func cpuScale(data []float64) float64 {
	peak := 1.0 // keep idle and low-usage plots nondegenerate
	for _, value := range data {
		peak = math.Max(peak, value)
	}
	magnitude := math.Pow(10, math.Floor(math.Log10(peak)))
	for _, step := range []float64{1, 2, 5, 10} {
		if top := step * magnitude; top >= peak {
			return top
		}
	}
	return peak
}

// cpuStroke draws a connected segment using two horizontal and four vertical
// dots per character. Baseline dots are preserved when a segment shares a cell.
func cpuStroke(buf *ui.Buffer, plot image.Rectangle, x0, y0, x1, y1 int, fg, bg ui.Attribute) {
	dx, dy := int(math.Abs(float64(x1-x0))), -int(math.Abs(float64(y1-y0)))
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	bits := [2][4]rune{{1, 2, 4, 64}, {8, 16, 32, 128}}
	for {
		x, y := plot.Min.X+x0/2, plot.Min.Y+y0/4
		if image.Pt(x, y).In(plot) {
			cell := buf.At(x, y)
			pattern := rune(0)
			if cell.Ch >= '\u2800' && cell.Ch <= '\u28ff' {
				pattern = cell.Ch - '\u2800'
			}
			buf.Set(x, y, ui.Cell{Ch: '\u2800' + (pattern | bits[x0%2][y0%4]), Fg: fg, Bg: bg})
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}
