package main

import (
	"image"
	"testing"

	ui "github.com/gizak/termui"
)

type framePart struct {
	buffer ui.Buffer
	reads  int
}

func (p *framePart) Buffer() ui.Buffer {
	p.reads++
	return p.buffer
}

func TestFrameCapturesAllSectionsAndOverlay(t *testing.T) {
	parts := make([]*framePart, 4)
	for n := range parts {
		buf := ui.NewBuffer()
		buf.SetArea(image.Rect(0, n, 10, n+1))
		buf.Set(0, n, ui.Cell{Ch: rune('A' + n)})
		parts[n] = &framePart{buffer: buf}
	}
	// The last component is an overlay covering part of the container row.
	parts[3].buffer.SetArea(image.Rect(0, 2, 10, 4))
	parts[3].buffer.Set(0, 2, ui.Cell{Ch: 'X'})
	f := newRenderFrame(false, parts[0], parts[1], parts[2], parts[3])
	for _, p := range parts {
		p.buffer.Set(0, 0, ui.Cell{Ch: '!'})
		for point := range p.buffer.CellMap {
			p.buffer.Set(point.X, point.Y, ui.Cell{Ch: '!'})
		}
	}
	buf := f.Buffer()
	for y, want := range []rune{'A', 'B', 'X', 'D'} {
		if got := buf.At(0, y).Ch; got != want {
			t.Errorf("row %d = %q, want %q", y, got, want)
		}
	}
	for _, p := range parts {
		if p.reads != 1 {
			t.Error("queued frame reread a mutable widget")
		}
	}
}

func TestFrameClipsComponentsBeforeCombining(t *testing.T) {
	a, b := ui.NewBuffer(), ui.NewBuffer()
	a.SetArea(image.Rect(0, 0, 2, 1))
	a.Set(5, 0, ui.Cell{Ch: '!'}) // outside a, but inside the combined frame
	b.SetArea(image.Rect(0, 1, 10, 2))
	b.Set(5, 1, ui.Cell{Ch: 'B'})
	buf := newRenderFrame(false, &framePart{buffer: a}, &framePart{buffer: b}).Buffer()
	if buf.At(5, 0).Ch != 0 || buf.At(5, 1).Ch != 'B' {
		t.Fatal("component clipping was lost when the frame was combined")
	}
}

func TestFrameDefersClearUntilRendering(t *testing.T) {
	f := newRenderFrame(true)
	if f.clear == nil {
		t.Fatal("clear request was lost")
	}
	clears := 0
	f.clear = func() { clears++ }
	if clears != 0 {
		t.Fatal("frame construction cleared the screen")
	}
	f.Buffer()
	if clears != 1 {
		t.Fatal("renderer did not clear before drawing the frame")
	}
	if newRenderFrame(false).clear != nil {
		t.Fatal("ordinary frame requests a clear")
	}
}
