package single

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bcicen/ctop/cwidgets"
	ui "github.com/gizak/termui"
)

type Net struct {
	*ui.Sparklines
	rxHist *DiffHist
	txHist *DiffHist
	mu     sync.Mutex
}

func NewNet() *Net {
	net := &Net{Sparklines: ui.NewSparklines(), rxHist: NewDiffHist(60), txHist: NewDiffHist(60)}
	net.BorderLabel = "NET"
	net.Height = 6
	net.Width = defaultPanelWidth
	net.X = 0
	net.Y = 24

	rx := ui.NewSparkline()
	rx.Title = "RX"
	rx.Height = 1
	rx.Data = net.rxHist.Data
	rx.LineColor = ui.ColorGreen

	tx := ui.NewSparkline()
	tx.Title = "TX"
	tx.Height = 1
	tx.Data = net.txHist.Data
	tx.LineColor = ui.ColorYellow

	net.Lines = []ui.Sparkline{rx, tx}
	return net
}

func (w *Net) Update(rx int64, tx int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var rate string

	w.rxHist.Append(int(rx))
	rate = strings.ToLower(cwidgets.ByteFormat(w.rxHist.Val))
	w.Lines[0].Title = fmt.Sprintf("RX [%s/s]", rate)

	w.txHist.Append(int(tx))
	rate = strings.ToLower(cwidgets.ByteFormat(w.txHist.Val))
	w.Lines[1].Title = fmt.Sprintf("TX [%s/s]", rate)
}

func (w *Net) Buffer() ui.Buffer {
	w.mu.Lock()
	defer w.mu.Unlock()
	width := w.InnerBounds().Dx()
	if width <= 0 {
		return w.Block.Buffer()
	}
	w.rxHist.Grow(width)
	w.txHist.Grow(width)
	w.Lines[0].Data = visibleHistory(w.rxHist.Data, width)
	w.Lines[1].Data = visibleHistory(w.txHist.Data, width)
	return w.Sparklines.Buffer()
}
