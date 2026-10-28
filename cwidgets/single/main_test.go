package single

import (
	"strings"
	"testing"

	"github.com/bcicen/ctop/models"
	ui "github.com/gizak/termui"
)

func TestSinglePanelsFollowWidth(t *testing.T) {
	s := NewSingle()
	s.X = 3
	s.SetMeta(models.NewMeta("name", "test", "image", "test:latest", "[ENV-VAR]", "KEY=value"))
	s.SetMetrics(models.Metrics{CPUUtil: 10, MemUsage: 10, MemLimit: 100})
	for _, width := range []int{180, 80, 40, 120, minPanelWidth} {
		s.SetWidth(width)
		s.align(40)
		for n, panelWidth := range []int{s.Info.Width, s.Cpu.Width, s.Mem.Width, s.Net.Width, s.IO.Width, s.Env.Width} {
			if panelWidth != width {
				t.Errorf("panel %d width = %d, want %d", n, panelWidth, width)
			}
		}
		for n, x := range []int{s.Info.X, s.Cpu.X, s.Mem.X, s.Net.X, s.IO.X, s.Env.X} {
			if x != s.X {
				t.Errorf("panel %d x = %d, want %d", n, x, s.X)
			}
		}
		if s.Mem.Chart.X != s.X+1 || s.Mem.Chart.Width != width-2 || s.Mem.InnerLabel.Width != width-2 {
			t.Fatal("memory chart children did not follow panel geometry")
		}
		buf := s.Buffer()
		if got := buf.Area.Dx(); got != width {
			t.Errorf("rendered width = %d, want %d", got, width)
		}
	}
}

func TestSingleWidthsAreIndependentAndRecoverFromNarrowViewport(t *testing.T) {
	a, b := NewSingle(), NewSingle()
	a.SetWidth(180)
	a.align(30)
	b.SetWidth(80)
	b.align(30)
	if a.Info.Width != 180 || b.Info.Width != 80 {
		t.Fatal("single views share mutable width state")
	}
	for _, width := range []int{0, 1, minPanelWidth - 1, 120, 40} {
		b.SetWidth(width)
		b.align(30)
		buf := b.Buffer()
		if width == minPanelWidth-1 {
			var text strings.Builder
			for x := 0; x < width; x++ {
				text.WriteRune(buf.At(x, 0).Ch)
			}
			if !strings.Contains(text.String(), "screen too small!") {
				t.Fatalf("missing narrow-screen message: %q", text.String())
			}
		}
		if width >= minPanelWidth && buf.Area.Dx() != width {
			t.Fatalf("normal layout did not recover at width %d", width)
		}
	}
}

func TestSingleScrollClampsWhenViewportChanges(t *testing.T) {
	s := NewSingle()
	s.SetWidth(100)
	s.Y = -1000
	s.align(20)
	if want := 20 - s.GetHeight(); s.Y != want {
		t.Fatalf("scroll = %d, want %d", s.Y, want)
	}
	s.align(1000)
	if s.Y != 0 || s.Info.Y != 0 {
		t.Fatal("scroll did not reset when the full view fits")
	}
	s.Y = 10
	s.align(20)
	if s.Y != 0 {
		t.Fatal("positive scroll offset survived layout")
	}
}

func TestNarrowTablesPreserveBordersAndValues(t *testing.T) {
	s := NewSingle()
	value := strings.Repeat("long-value-", 20)
	s.SetMeta(models.NewMeta("image", value, "[ENV-VAR]", "KEY="+value))
	for _, width := range []int{40, 180, 20, 120} {
		s.SetWidth(width)
		s.align(40)
		for _, buf := range []ui.Buffer{s.Info.Buffer(), s.Env.Buffer()} {
			if buf.At(width-1, buf.Area.Min.Y+1).Ch != '│' {
				t.Fatalf("table value overwrote border at width %d", width)
			}
			for p := range buf.CellMap {
				if !p.In(buf.Area) {
					t.Fatalf("table drew outside bounds at %v", p)
				}
			}
		}
	}
	if s.Info.data["image"] != value || s.Env.data["KEY"] != value {
		t.Fatal("resizing truncated stored table values")
	}
}
