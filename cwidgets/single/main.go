package single

import (
	"github.com/bcicen/ctop/logging"
	"github.com/bcicen/ctop/models"
	"github.com/bcicen/ctop/terminal"
	ui "github.com/gizak/termui"
)

const (
	defaultPanelWidth = 65 // constructors are sized before the first layout
	minPanelWidth     = 20
)

var log = logging.Init()

type Single struct {
	Info  *Info
	Net   *Net
	Cpu   *Cpu
	Mem   *Mem
	IO    *IO
	Env   *Env
	X, Y  int
	Width int
}

func NewSingle() *Single {
	width, _ := terminal.Size()
	return &Single{
		Info:  NewInfo(),
		Net:   NewNet(),
		Cpu:   NewCpu(),
		Mem:   NewMem(),
		IO:    NewIO(),
		Env:   NewEnv(),
		Width: width,
	}
}

func (e *Single) Up() {
	if e.Y < 0 {
		e.Y++
		e.Align()
	}
}

func (e *Single) Down() {
	_, height := terminal.Size()
	if e.Y > (height - e.GetHeight()) {
		e.Y--
		e.Align()
	}
}

func (e *Single) SetWidth(w int) { e.Width = w }
func (e *Single) SetMeta(m models.Meta) {
	for k, v := range m {
		if k == "[ENV-VAR]" {
			e.Env.Set(v)
		} else {
			e.Info.Set(k, v)
		}
	}
}

func (e *Single) SetMetrics(m models.Metrics) {
	e.Cpu.Update(m.CPUUtil)
	e.Net.Update(m.NetRx, m.NetTx)
	e.Mem.Update(int(m.MemUsage), int(m.MemLimit))
	e.IO.Update(m.IOBytesRead, m.IOBytesWrite)
}

// GetHeight returns total column height
func (e *Single) GetHeight() (h int) {
	h += e.Info.Height
	h += e.Net.Height
	h += e.Cpu.Height
	h += e.Mem.Height
	h += e.IO.Height
	h += e.Env.Height
	return h
}

func (e *Single) Align() {
	_, height := terminal.Size()
	e.align(height)
}

func (e *Single) align(height int) {
	// Preserve scrolling where possible, without leaving blank space below the
	// content after the viewport or metadata changes height.
	minY := height - e.GetHeight()
	if minY > 0 {
		minY = 0
	}
	if e.Y < minY {
		e.Y = minY
	}
	if e.Y > 0 {
		e.Y = 0
	}

	y := e.Y
	for _, i := range e.all() {
		i.SetX(e.X)
		i.SetY(y)
		i.SetWidth(e.Width)
		y += i.GetHeight()
	}

	e.Mem.Align()
	log.Debugf("align: width=%v", e.Width)
}

func (e *Single) Buffer() ui.Buffer {
	buf := ui.NewBuffer()
	if e.Width < minPanelWidth {
		buf.Merge(termSizeError(e.Width).Buffer())
		return buf
	}
	buf.Merge(e.Info.Buffer())
	buf.Merge(e.Cpu.Buffer())
	buf.Merge(e.Mem.Buffer())
	buf.Merge(e.Net.Buffer())
	buf.Merge(e.IO.Buffer())
	buf.Merge(e.Env.Buffer())
	return buf
}

func (e *Single) all() []ui.GridBufferer {
	return []ui.GridBufferer{
		e.Info,
		e.Cpu,
		e.Mem,
		e.Net,
		e.IO,
		e.Env,
	}
}

func termSizeError(width int) *ui.Par {
	p := ui.NewPar("screen too small!")
	p.Height = 1
	p.Width = width
	p.Border = false
	return p
}
