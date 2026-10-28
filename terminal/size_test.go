package terminal

import (
	"sync"
	"testing"

	ui "github.com/gizak/termui"
)

func TestCachedReadsAndResize(t *testing.T) {
	c := dimensionCache{width: 120, height: 40}
	refreshes := 0
	refresh := func() { refreshes++ }
	for i := 0; i < 100; i++ {
		if w, h := c.size(); w != 120 || h != 40 {
			t.Fatalf("initial dimensions = %dx%d", w, h)
		}
	}
	for _, size := range [][2]int{{120, 40}, {0, 30}, {80, -1}, {-1, -1}} {
		c.resize(size[0], size[1], refresh)
	}
	if refreshes != 0 {
		t.Fatal("unchanged or invalid dimensions refreshed the terminal")
	}
	if w, h := c.size(); w != 120 || h != 40 {
		t.Fatalf("invalid resize changed dimensions to %dx%d", w, h)
	}
	c.resize(80, 24, refresh)
	c.resize(80, 24, refresh)
	if w, h := c.size(); w != 80 || h != 24 || refreshes != 1 {
		t.Fatalf("resize = %dx%d, refreshes = %d", w, h, refreshes)
	}
}

func TestResizeUpdatesBeforeLayout(t *testing.T) {
	c := dimensionCache{width: 120, height: 40}
	refreshed := false
	called := false
	handler := c.resizeHandler(func(e ui.Event) {
		called = true
		if w, h := c.size(); w != 80 || h != 24 || !refreshed {
			t.Errorf("layout ran before dimensions updated: %dx%d, refreshed=%v", w, h, refreshed)
		}
	}, func() { refreshed = true })
	handler(ui.Event{Data: ui.EvtWnd{Width: 80, Height: 24}})
	if !called {
		t.Fatal("view handler was not called")
	}
}

func TestInitAndResetRetainResizeHandling(t *testing.T) {
	oldStream := ui.DefaultEvtStream
	ui.DefaultEvtStream = ui.NewEvtStream()
	t.Cleanup(func() { ui.DefaultEvtStream = oldStream })
	Init()
	r := ui.TermRect()
	if w, h := Size(); w != r.Dx() || h != r.Dy() {
		t.Fatalf("initial cache differs from termui: %dx%d, %v", w, h, r)
	}
	HandleResize(func(ui.Event) { t.Error("old resize handler survived reset") })
	ui.Handle("/sys/kbd/q", func(ui.Event) {})
	ResetHandlers()
	if len(ui.DefaultEvtStream.Handlers) != 1 || ui.DefaultEvtStream.Handlers["/sys/wnd/resize"] == nil {
		t.Fatal("reset did not preserve only dimension handling")
	}
	// No actual change: the fallback must run without requiring a real terminal.
	ui.DefaultEvtStream.Handlers["/sys/wnd/resize"](ui.Event{Data: ui.EvtWnd{Width: r.Dx(), Height: r.Dy()}})
}

func TestConcurrentSnapshots(t *testing.T) {
	c := dimensionCache{width: 80, height: 40}
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 1; i <= 500; i++ {
				c.resize(i*2, i, func() {})
				w, h := c.size()
				if w != h*2 {
					t.Errorf("inconsistent snapshot: %dx%d", w, h)
				}
			}
		}()
	}
	wg.Wait()
}
