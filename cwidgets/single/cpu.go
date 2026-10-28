package single

import (
	"sync"

	ui "github.com/gizak/termui"
)

type Cpu struct {
	*ui.LineChart
	hist FloatHist
	mu   sync.Mutex
}

func NewCpu() *Cpu {
	cpu := &Cpu{LineChart: ui.NewLineChart(), hist: NewFloatHist(55)}
	cpu.Mode = "dot"
	cpu.BorderLabel = "CPU"
	cpu.Height = 12
	cpu.Width = defaultPanelWidth
	cpu.X = 0
	cpu.DataLabels = cpu.hist.Labels

	// hack to force the default minY scale to 0
	tmpData := []float64{20}
	cpu.Data["CPU"] = tmpData
	_ = cpu.LineChart.Buffer()

	cpu.Data["CPU"] = cpu.hist.Data
	return cpu
}

func (w *Cpu) Update(val int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.hist.Append(float64(val))
}

func (w *Cpu) Buffer() ui.Buffer {
	w.mu.Lock()
	defer w.mu.Unlock()
	width := w.InnerBounds().Dx()
	if width <= 0 {
		return w.Block.Buffer()
	}
	w.hist.Grow(width)
	w.Data["CPU"] = visibleHistory(w.hist.Data, width)
	w.DataLabels = visibleHistory(w.hist.Labels, width)
	return w.LineChart.Buffer()
}
