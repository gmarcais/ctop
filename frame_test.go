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

func TestFrameErasesRemovedRowsAndPreservesMovedContent(t *testing.T) {
	var composer frameComposer
	viewport := image.Rect(0, 0, 12, 10)
	part := func(y int, text string) *framePart {
		buf := ui.NewBuffer()
		buf.SetArea(image.Rect(0, y, len(text), y+1))
		for x, ch := range text {
			buf.Set(x, y, ui.Cell{Ch: ch})
		}
		return &framePart{buffer: buf}
	}
	// Two build rows above the container table.
	composer.capture(false, viewport, part(0, "BUILDS"), part(1, "build-a"), part(2, "build-b"), part(4, "container"))
	// One build ends and the container table moves up one row.
	one := composer.capture(false, viewport, part(0, "BUILDS"), part(1, "build-b"), part(3, "container"))
	if one.clear != nil || one.buffer.At(0, 2).Ch != ' ' || one.buffer.At(0, 4).Ch != ' ' {
		t.Fatal("removed build or old container position was not erased in the frame")
	}
	if one.buffer.At(0, 3).Ch != 'c' || one.buffer.At(6, 1).Ch != 'b' {
		t.Fatal("erasure overwrote surviving build or moved container")
	}
	// The last build ends, including removal of its section heading.
	zero := composer.capture(false, viewport, part(0, "container"))
	if zero.clear != nil || zero.buffer.At(0, 1).Ch != ' ' || zero.buffer.At(0, 3).Ch != ' ' {
		t.Fatal("last build removal did not erase the section and displaced table")
	}
	if zero.buffer.At(0, 0).Ch != 'c' {
		t.Fatal("last build removal erased the replacement content")
	}
	// Each queued frame owns its erasure cells independently of later layouts.
	if one.buffer.At(0, 3).Ch != 'c' {
		t.Fatal("later frame mutated an earlier captured frame")
	}
}

func TestFrameRemovalIsClippedAfterResize(t *testing.T) {
	var composer frameComposer
	buf := ui.NewBuffer()
	buf.SetArea(image.Rect(0, 0, 20, 10))
	buf.Set(15, 8, ui.Cell{Ch: 'B'})
	composer.capture(false, image.Rect(0, 0, 20, 10), &framePart{buffer: buf})
	viewport := image.Rect(0, 0, 8, 4)
	f := composer.capture(false, viewport)
	if f.buffer.Area != viewport {
		t.Fatalf("erasure area = %v, want %v", f.buffer.Area, viewport)
	}
	for p, cell := range f.buffer.CellMap {
		if !p.In(viewport) || cell.Ch != ' ' {
			t.Fatalf("invalid erasure cell at %v: %+v", p, cell)
		}
	}
	// The next frame need not retain the old, larger erasure footprint.
	if next := composer.capture(false, viewport); !next.buffer.Area.Empty() {
		t.Fatalf("empty layout retained the erased footprint: %v", next.buffer.Area)
	}
}

func TestFrameErasesShorterTextAndKeepsOverlay(t *testing.T) {
	var composer frameComposer
	viewport := image.Rect(0, 0, 20, 10)
	buf := ui.NewBuffer()
	buf.SetArea(image.Rect(0, 0, 20, 3))
	buf.Set(18, 0, ui.Cell{Ch: 'X'})
	composer.capture(false, viewport, &framePart{buffer: buf})
	overlay := ui.NewBuffer()
	overlay.SetArea(image.Rect(0, 2, 5, 3))
	overlay.Set(0, 2, ui.Cell{Ch: 'F'})
	f := composer.capture(false, viewport, &framePart{buffer: overlay})
	if f.buffer.At(18, 0).Ch != ' ' || f.buffer.At(0, 2).Ch != 'F' {
		t.Fatal("old text was not erased or the overlay was overwritten")
	}
}
