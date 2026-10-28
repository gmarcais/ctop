package main

import ui "github.com/gizak/termui"

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
