// Package terminal provides side-effect-free terminal dimension reads for ctop.
package terminal

import (
	"sync"

	ui "github.com/gizak/termui"
)

type dimensionCache struct {
	mu            sync.RWMutex
	width, height int
}

var dimensions dimensionCache

// Init captures the dimensions populated by ui.Init and installs resize handling.
// Call it after ui.Init, before constructing any views.
func Init() {
	r := ui.TermRect()
	dimensions.mu.Lock()
	dimensions.width, dimensions.height = r.Dx(), r.Dy()
	dimensions.mu.Unlock()
	HandleResize(nil)
}

// Size returns one consistent cached snapshot. It never touches the terminal.
func Size() (int, int) { return dimensions.size() }

func (c *dimensionCache) size() (int, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.width, c.height
}

func (c *dimensionCache) resize(width, height int, refresh func()) {
	if width <= 0 || height <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.width == width && c.height == height {
		return
	}
	refresh()
	c.width, c.height = width, height
}

// refreshTermui refreshes termui's internal layout rectangle. Its getter calls
// termbox.Sync, so it must only be used for an actual resize, never during layout.
func refreshTermui() { ui.TermWidth() }

func (c *dimensionCache) resizeHandler(handler func(ui.Event), refresh func()) func(ui.Event) {
	return func(e ui.Event) {
		if wnd, ok := e.Data.(ui.EvtWnd); ok {
			c.resize(wnd.Width, wnd.Height, refresh)
		}
		width, _ := c.size()
		if ui.Body != nil {
			ui.Body.Width = width
		}
		if handler != nil {
			handler(e)
		}
	}
}

// HandleResize updates cached dimensions before the view's resize handler runs.
func HandleResize(handler func(ui.Event)) {
	ui.Handle("/sys/wnd/resize", dimensions.resizeHandler(handler, refreshTermui))
}

// ResetHandlers retains dimension updates in views with no custom resize handler.
func ResetHandlers() {
	ui.DefaultEvtStream.ResetHandlers()
	HandleResize(nil)
}
