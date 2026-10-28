package single

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	ui "github.com/gizak/termui"
)

func cpuLabel(buf ui.Buffer, x, y int) string {
	var label strings.Builder
	for col := 0; col < cpuAxisWidth-2; col++ {
		label.WriteRune(buf.At(x+col, y).Ch)
	}
	return strings.TrimSpace(label.String())
}

func TestCPUIdleSamplesShareZeroAxis(t *testing.T) {
	cpu := NewCpu()
	cpu.SetWidth(100)
	cpu.SetY(7)
	buf := cpu.Buffer()
	inner := cpu.InnerBounds()
	baseline := inner.Max.Y - 1
	if label := cpuLabel(buf, inner.Min.X, baseline); label != "0%" {
		t.Fatalf("zero label = %q", label)
	}
	for x := inner.Min.X + cpuAxisWidth; x < inner.Max.X; x++ {
		if got := buf.At(x, baseline).Ch; got != '⣀' {
			t.Fatalf("idle sample at column %d is not on the zero axis: %q", x, got)
		}
		for y := inner.Min.Y; y < baseline; y++ {
			if ch := buf.At(x, y).Ch; ch >= '\u2801' && ch <= '\u28ff' {
				t.Fatalf("idle plot floats above zero at (%d,%d)", x, y)
			}
		}
	}
}

func TestCPUScaleRelabelsWithoutNegativeTicks(t *testing.T) {
	cpu := NewCpu()
	cpu.SetWidth(100)
	cpu.Buffer()
	for _, step := range []struct{ value, top int }{{100, 100}, {1, 1}, {42, 50}, {0, 1}} {
		for n := 0; n < len(cpu.hist.Data); n++ {
			cpu.Update(step.value)
		}
		buf := cpu.Buffer()
		inner := cpu.InnerBounds()
		if got, want := cpuLabel(buf, inner.Min.X, inner.Min.Y), strconv.Itoa(step.top)+"%"; got != want {
			t.Fatalf("usage %d: top label = %q, want %q", step.value, got, want)
		}
		for y := inner.Min.Y; y < inner.Max.Y; y++ {
			label := cpuLabel(buf, inner.Min.X, y)
			if label == "" {
				continue
			}
			value, err := strconv.ParseFloat(strings.TrimSuffix(label, "%"), 64)
			if err != nil || value < 0 || value > float64(step.top) {
				t.Fatalf("invalid axis label %q for top %d", label, step.top)
			}
		}
	}
	before := append([]float64(nil), cpu.hist.Data...)
	cpu.Update(-1)
	cpu.Update(-100)
	if !reflect.DeepEqual(cpu.hist.Data, before) {
		t.Fatal("invalid negative CPU usage changed the history")
	}
}

func TestCPUShowsSubcellDetailAboveBaseline(t *testing.T) {
	cpu := NewCpu()
	cpu.SetWidth(100)
	cpu.Buffer()
	cpu.Update(100)
	for n := 0; n < len(cpu.hist.Data)-2; n++ {
		cpu.Update(5)
	}
	buf := cpu.Buffer()
	inner := cpu.InnerBounds()
	cell := buf.At(inner.Max.X-1, inner.Max.Y-1)
	if cell.Ch < '\u2800' || cell.Ch > '\u28ff' {
		t.Fatalf("low usage is not a Braille cell: %q", cell.Ch)
	}
	pattern := cell.Ch - '\u2800'
	if pattern&0xc0 != 0xc0 || pattern&^rune(0xc0) == 0 {
		t.Fatalf("5%% usage should show detail above the shared zero line: %q", cell.Ch)
	}
}

func TestCPULatestPeakIsAtRightEdge(t *testing.T) {
	cpu := NewCpu()
	cpu.SetWidth(180)
	cpu.Buffer()
	cpu.Update(100)
	buf := cpu.Buffer()
	inner := cpu.InnerBounds()
	cell := buf.At(inner.Max.X-1, inner.Min.Y)
	if cell.Ch < '\u2800' || cell.Ch > '\u28ff' || (cell.Ch-'\u2800')&8 == 0 {
		t.Fatalf("newest peak is missing from the top-right plot position: %q", cell.Ch)
	}
	for p := range buf.CellMap {
		if !p.In(buf.Area) {
			t.Fatalf("CPU plot drew outside its panel at %v", p)
		}
	}
}

func TestCPUTinyDimensionsDoNotPanic(t *testing.T) {
	cpu := NewCpu()
	for _, width := range []int{0, 1, 5, 10, 20} {
		for _, height := range []int{0, 1, 2, 3, 12} {
			cpu.SetWidth(width)
			cpu.Height = height
			cpu.Buffer()
		}
	}
}
