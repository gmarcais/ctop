package single

import (
	"reflect"
	"sync"
	"testing"

	ui "github.com/gizak/termui"
)

func TestHistoriesGrowWithoutLosingSamples(t *testing.T) {
	ints := NewIntHist(3)
	floats := NewFloatHist(3)
	for _, v := range []int{1, 2, 3} {
		ints.Append(v)
		floats.Append(float64(v))
	}
	ints.Grow(6)
	floats.Grow(6)
	if !reflect.DeepEqual(ints.Data, []int{0, 0, 0, 1, 2, 3}) || !reflect.DeepEqual(floats.Data, []float64{0, 0, 0, 1, 2, 3}) {
		t.Fatal("growth did not preserve samples at the right edge")
	}
	ints.Grow(2)
	floats.Grow(2)
	if len(ints.Data) != 6 || len(floats.Data) != 6 {
		t.Fatal("narrow viewport discarded retained history")
	}
	ints.Append(4)
	floats.Append(4)
	if !reflect.DeepEqual(visibleHistory(ints.Data, 3), []int{2, 3, 4}) || !reflect.DeepEqual(visibleHistory(floats.Data, 3), []float64{2, 3, 4}) {
		t.Fatal("visible history did not retain the newest samples")
	}
	if ints.Val != 4 || floats.Val != 4 || len(ints.Labels) != 6 || len(floats.Labels) != 6 {
		t.Fatal("current values or labels diverged after growth")
	}
}

func TestNetworkAndIOPlotAtRightEdgeAndFillWideHistory(t *testing.T) {
	net, io := NewNet(), NewIO()
	for _, test := range []struct {
		name   string
		widget *ui.Sparklines
		update func(int64, int64)
		buffer func() ui.Buffer
	}{
		{"network", net.Sparklines, net.Update, net.Buffer},
		{"io", io.Sparklines, io.Update, io.Buffer},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.widget.SetWidth(240)
			test.buffer() // allocate history for a wide viewport before sampling
			test.update(0, 0)
			test.update(10, 20)
			assertSpark := func(buf ui.Buffer, x, y int, color ui.Attribute) {
				cell := buf.At(x, y)
				if cell.Bg != color && !(cell.Fg == color && cell.Ch >= '▁' && cell.Ch <= '█') {
					t.Fatalf("no plotted sample at (%d,%d): %+v", x, y, cell)
				}
			}
			buf := test.buffer()
			inner := test.widget.InnerBounds()
			assertSpark(buf, inner.Max.X-1, inner.Min.Y+1, test.widget.Lines[0].LineColor)
			assertSpark(buf, inner.Max.X-1, inner.Min.Y+3, test.widget.Lines[1].LineColor)
			for n := int64(2); n <= 300; n++ {
				test.update(n*10, n*20)
			}
			for _, width := range []int{240, 40, 180, 80, 240} {
				test.widget.SetWidth(width)
				buf := test.buffer()
				inner := test.widget.InnerBounds()
				for n, line := range test.widget.Lines {
					if len(line.Data) != inner.Dx() {
						t.Fatalf("series %d has %d columns, want %d", n, len(line.Data), inner.Dx())
					}
					assertSpark(buf, inner.Min.X, inner.Min.Y+1+n*2, line.LineColor)
					assertSpark(buf, inner.Max.X-1, inner.Min.Y+1+n*2, line.LineColor)
				}
			}
		})
	}
}

func TestCPUUsesWidePlotAndNewestVisibleSamples(t *testing.T) {
	cpu := NewCpu()
	cpu.Update(10)
	cpu.Update(42)
	for _, width := range []int{180, 40, 240, 80, 180} {
		cpu.SetWidth(width)
		buf := cpu.Buffer()
		inner := cpu.InnerBounds()
		dots := make(map[int]bool)
		for p, cell := range buf.CellMap {
			if cell.Ch == cpu.DotStyle && p.In(inner) {
				dots[p.X] = true
			}
		}
		if !dots[inner.Max.X-1] || len(dots) < width-15 {
			t.Fatalf("CPU plot does not fill width %d: %d plotted columns", width, len(dots))
		}
		series := cpu.Data["CPU"]
		if len(series) != inner.Dx() || series[len(series)-1] != 42 || series[len(series)-2] != 10 {
			t.Fatal("CPU resize lost or misplaced recent samples")
		}
	}
}

func TestChartUpdatesDuringHistoryResize(t *testing.T) {
	cpu, net, io := NewCpu(), NewNet(), NewIO()
	mem := NewMem()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for n := 0; n < 100; n++ {
			cpu.Update(n)
			net.Update(int64(n*10), int64(n*20))
			io.Update(int64(n*30), int64(n*40))
			mem.Update(n, 100)
		}
	}()
	for n := 0; n < 30; n++ {
		width := 40 + n*5
		cpu.SetWidth(width)
		net.SetWidth(width)
		io.SetWidth(width)
		mem.SetWidth(width)
		cpu.Buffer()
		net.Buffer()
		io.Buffer()
		mem.Buffer()
	}
	wg.Wait()
}

func TestMemoryPlotsOneColumnSamplesAcrossAvailableWidth(t *testing.T) {
	mem := NewMem()
	mem.Chart.BarColor[0] = ui.ColorGreen
	mem.SetWidth(180)
	mem.Buffer() // grow history before the first sample
	mem.Update(40, 100)
	buf := mem.Buffer()
	inner := mem.Chart.InnerBounds()
	if cell := buf.At(inner.Max.X-1, inner.Max.Y-2); cell.Bg != mem.Chart.BarColor[0] {
		t.Fatalf("new memory sample is not at the right edge: %+v", cell)
	}
	for n := 0; n < 200; n++ {
		mem.Update(40, 100)
	}
	for _, width := range []int{180, 40, 120, 80, 180} {
		mem.SetWidth(width)
		buf := mem.Buffer()
		inner := mem.Chart.InnerBounds()
		for x := inner.Min.X; x < inner.Max.X; x++ {
			if cell := buf.At(x, inner.Max.Y-2); cell.Bg != mem.Chart.BarColor[0] {
				t.Fatalf("memory history has a gap or truncated sample at column %d, width %d: %+v", x, width, cell)
			}
		}
		if len(mem.Chart.Data[0]) != inner.Dx() || len(mem.Chart.Data[1]) != inner.Dx() {
			t.Fatal("memory series do not match the available width")
		}
	}
	// A single changed value must occupy exactly the newest column, rather than
	// stretching across multiple cells as the old nine-bar chart did.
	mem.Update(100, 100)
	buf = mem.Buffer()
	inner = mem.Chart.InnerBounds()
	y := inner.Min.Y
	used := 0
	for x := inner.Min.X; x < inner.Max.X; x++ {
		if buf.At(x, y).Bg == mem.Chart.BarColor[0] {
			used++
		}
	}
	if used != 1 || buf.At(inner.Max.X-1, y).Bg != mem.Chart.BarColor[0] {
		t.Fatalf("new memory spike occupies %d columns instead of the rightmost column", used)
	}
	if mem.InnerLabel.Text == "" || mem.InnerLabel.Text == "-" {
		t.Fatal("memory summary was lost")
	}
}
