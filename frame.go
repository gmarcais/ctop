package main

import (
	"image"

	ui "github.com/gizak/termui"
)

// frameComposer remembers the last layout's footprint so disappearing rows can
// be replaced with blank cells in the next frame, without clearing the screen.
type frameComposer struct {
	area image.Rectangle
}

func (c *frameComposer) capture(clear bool, viewport image.Rectangle, parts ...ui.Bufferer) *renderFrame {
	f := newRenderFrame(clear, parts...)
	current := f.buffer.Area.Intersect(viewport)
	previous := c.area.Intersect(viewport)
	c.area = current
	if !clear {
		blank := ui.Cell{Ch: ' ', Fg: ui.ColorDefault, Bg: ui.ThemeAttr("bg")}
		for y := previous.Min.Y; y < previous.Max.Y; y++ {
			for x := previous.Min.X; x < previous.Max.X; x++ {
				p := image.Pt(x, y)
				if _, exists := f.buffer.CellMap[p]; !exists {
					f.buffer.Set(x, y, blank)
				}
			}
		}
		f.buffer.SetArea(current.Union(previous))
	} else {
		f.buffer.SetArea(current)
	}
	return f
}

// renderFrame captures layout before submitting it to termui's asynchronous
// renderer. Later widget changes cannot alter a frame that is already queued.
type renderFrame struct {
	buffer ui.Buffer
	clear  func()
}

func newRenderFrame(clear bool, parts ...ui.Bufferer) *renderFrame {
	f := &renderFrame{buffer: ui.NewBuffer()}
	for _, part := range parts {
		buf := part.Buffer()
		for p, cell := range buf.CellMap {
			// Respect each component's clipping before combining their areas.
			if p.In(buf.Area) {
				f.buffer.Set(p.X, p.Y, cell)
			}
		}
		f.buffer.SetArea(f.buffer.Area.Union(buf.Area))
	}
	if clear {
		f.clear = ui.Clear
	}
	return f
}

func (f *renderFrame) Buffer() ui.Buffer {
	// Clear the back buffer on the render goroutine, immediately before the
	// completed frame is drawn and flushed, rather than between queued frames.
	if f.clear != nil {
		f.clear()
	}
	return f.buffer
}
