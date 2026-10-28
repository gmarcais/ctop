package single

import (
	"fmt"
	"sync"

	"github.com/bcicen/ctop/cwidgets"
	ui "github.com/gizak/termui"
)

type Mem struct {
	*ui.Block
	Chart      *ui.MBarChart
	InnerLabel *ui.Par
	valHist    *IntHist
	limitHist  *IntHist
	mu         sync.Mutex
}

func NewMem() *Mem {
	mem := &Mem{
		Block:      ui.NewBlock(),
		Chart:      newMemChart(),
		InnerLabel: newMemLabel(),
		valHist:    NewIntHist(9),
		limitHist:  NewIntHist(9),
	}
	mem.Height = 13
	mem.Width = defaultPanelWidth
	mem.BorderLabel = "MEM"

	mem.Chart.Data[0] = mem.valHist.Data
	mem.Chart.Data[1] = mem.limitHist.Data
	mem.Chart.DataLabels = mem.valHist.Labels

	return mem
}

func (w *Mem) Align() {
	y := w.Y + 1
	w.InnerLabel.SetX(w.X + 1)
	w.InnerLabel.SetY(y)
	w.InnerLabel.SetWidth(w.Width - 2)
	w.Chart.SetX(w.X + 1)
	w.Chart.SetY(y + w.InnerLabel.Height)

	w.Chart.Height = w.Height - w.InnerLabel.Height - 2
	w.Chart.SetWidth(w.Width - 2)
}

func (w *Mem) Buffer() ui.Buffer {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.Align()
	width := w.Chart.Width
	if width <= 0 {
		return w.Block.Buffer()
	}
	w.valHist.Grow(width)
	w.limitHist.Grow(width)
	w.Chart.Data[0] = visibleHistory(w.valHist.Data, width)
	w.Chart.Data[1] = visibleHistory(w.limitHist.Data, width)
	w.Chart.DataLabels = visibleHistory(w.valHist.Labels, width)
	buf := ui.NewBuffer()
	buf.Merge(w.Block.Buffer())
	buf.Merge(w.InnerLabel.Buffer())
	buf.Merge(w.Chart.Buffer())
	return buf
}

func newMemLabel() *ui.Par {
	p := ui.NewPar("-")
	p.X = 1
	p.Border = false
	p.Height = 1
	p.Width = 20
	return p
}

func newMemChart() *ui.MBarChart {
	mbar := ui.NewMBarChart()
	mbar.X = 1
	mbar.Border = false
	mbar.BarGap = 0
	mbar.BarWidth = 1

	mbar.BarColor[1] = ui.ColorBlack
	mbar.NumColor[1] = ui.ColorBlack

	// The summary label provides the values; individual one-column samples
	// cannot accommodate readable numeric labels.
	mbar.NumFmt = func(int) string { return "" }
	//mbar.ShowScale = true
	return mbar
}

func (w *Mem) Update(val int, limit int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.valHist.Append(val)
	w.limitHist.Append(limit - val)
	w.InnerLabel.Text = fmt.Sprintf("%v / %v", cwidgets.ByteFormatShort(val), cwidgets.ByteFormatShort(limit))
	//w.Data[0] = w.hist.data
}
