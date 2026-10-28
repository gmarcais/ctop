package widgets

import (
	"strings"
	"testing"
	"time"

	"github.com/bcicen/ctop/builds"
	"github.com/mattn/go-runewidth"
)

func TestBuildPanelRows(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	rows := []builds.Build{{Name: "app", Builder: "default", CreatedAt: now.Add(-65 * time.Second), TotalSteps: 8, CompletedSteps: 3, CachedSteps: 2}}
	p := NewBuildPanel(rows, "", 120, 6, now)
	if p.Height != 2 || !strings.Contains(p.lines[1], "1m5s") || !strings.Contains(p.lines[1], "3/8 (2 cached)") {
		t.Fatalf("incorrect build rows: %v", p.lines)
	}
	rows[0].TotalSteps = 0
	p = NewBuildPanel(rows, "", 120, 6, now)
	if strings.Contains(p.lines[1], "0/0") || !strings.HasSuffix(p.lines[1], "  -") {
		t.Fatalf("unknown progress should be unavailable: %v", p.lines)
	}
	if p = NewBuildPanel(nil, "", 120, 6, now); p.Height != 0 {
		t.Fatal("empty panel takes screen space")
	}
}

func TestBuildPanelScreenLimits(t *testing.T) {
	rows := make([]builds.Build, 20)
	for _, height := range []int{-1, 0, 1, 2, 3, 5, 10} {
		for _, width := range []int{0, 1, 10, 80} {
			for _, warning := range []string{"", "Build history unavailable"} {
				p := NewBuildPanel(rows, warning, width, height, time.Now())
				if p.Height > 0 && p.Height > height {
					t.Fatalf("panel exceeds height %d: %v", height, p.lines)
				}
				if width > 0 {
					_ = p.Buffer() // tiny terminals must not panic
				}
			}
		}
	}
	p := NewBuildPanel(rows, "", 80, 4, time.Now())
	if p.lines[3] != "+ 18 more active builds" {
		t.Fatalf("incorrect overflow count: %v", p.lines)
	}
}

func TestBuildPanelLiteralAndWideNames(t *testing.T) {
	s := buildField("界界界\n\x1b[red]", 8)
	if runewidth.StringWidth(s) != 8 || strings.ContainsAny(s, "\n\x1b") {
		t.Fatalf("unsafe or incorrectly clipped field: %q", s)
	}
	p := NewBuildPanel([]builds.Build{{Name: "[app](fg-red)", Builder: "default"}}, "", 100, 4, time.Now())
	p.Y = 2
	buf := p.Buffer()
	// Build names must remain literal, including brackets used by termui markup.
	if buf.At(p.X+9, p.Y+1).Ch != '[' {
		t.Fatal("build name interpreted as markup")
	}
}
