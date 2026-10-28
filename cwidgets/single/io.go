package single

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bcicen/ctop/cwidgets"
	ui "github.com/gizak/termui"
)

type IO struct {
	*ui.Sparklines
	readHist  *DiffHist
	writeHist *DiffHist
	mu        sync.Mutex
}

func NewIO() *IO {
	io := &IO{Sparklines: ui.NewSparklines(), readHist: NewDiffHist(60), writeHist: NewDiffHist(60)}
	io.BorderLabel = "IO"
	io.Height = 6
	io.Width = defaultPanelWidth
	io.X = 0
	io.Y = 24

	read := ui.NewSparkline()
	read.Title = "READ"
	read.Height = 1
	read.Data = io.readHist.Data
	read.LineColor = ui.ColorGreen

	write := ui.NewSparkline()
	write.Title = "WRITE"
	write.Height = 1
	write.Data = io.writeHist.Data
	write.LineColor = ui.ColorYellow

	io.Lines = []ui.Sparkline{read, write}
	return io
}

func (w *IO) Update(read int64, write int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var rate string

	w.readHist.Append(int(read))
	rate = strings.ToLower(cwidgets.ByteFormatShort(w.readHist.Val))
	w.Lines[0].Title = fmt.Sprintf("read [%s/s]", rate)

	w.writeHist.Append(int(write))
	rate = strings.ToLower(cwidgets.ByteFormatShort(w.writeHist.Val))
	w.Lines[1].Title = fmt.Sprintf("write [%s/s]", rate)
}

func (w *IO) Buffer() ui.Buffer {
	w.mu.Lock()
	defer w.mu.Unlock()
	width := w.InnerBounds().Dx()
	if width <= 0 {
		return w.Block.Buffer()
	}
	w.readHist.Grow(width)
	w.writeHist.Grow(width)
	w.Lines[0].Data = visibleHistory(w.readHist.Data, width)
	w.Lines[1].Data = visibleHistory(w.writeHist.Data, width)
	return w.Sparklines.Buffer()
}
