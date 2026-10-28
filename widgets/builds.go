package widgets

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/bcicen/ctop/builds"
	ui "github.com/gizak/termui"
	"github.com/mattn/go-runewidth"
)

// BuildPanel displays read-only build rows independently of container selection.
type BuildPanel struct {
	*ui.Block
	lines []string
}

func NewBuildPanel(rows []builds.Build, warning string, width, maxHeight int, now time.Time) *BuildPanel {
	p := &BuildPanel{Block: ui.NewBlock()}
	p.Border = false
	p.X = 1
	p.Width = width
	if width <= 0 || maxHeight < 2 || (len(rows) == 0 && warning == "") {
		p.Height = 0
		return p
	}
	p.lines = append(p.lines, fmt.Sprintf("ACTIVE BUILDS (%d)  NAME / BUILDER / ELAPSED / STEPS (CACHED)", len(rows)))
	available := maxHeight - 1
	if warning != "" {
		available--
	}
	visible := len(rows)
	if visible > available {
		visible = available - 1 // reserve a line for the overflow count
	}
	if visible < 0 {
		visible = 0
	}
	for _, b := range rows[:visible] {
		name := b.Name
		if name == "" {
			name = b.Ref
		}
		elapsed := "-"
		if !b.CreatedAt.IsZero() {
			d := now.Sub(b.CreatedAt)
			if d < 0 {
				d = 0
			}
			elapsed = d.Truncate(time.Second).String()
		}
		steps := "-"
		if b.TotalSteps > 0 {
			steps = fmt.Sprintf("%d/%d (%d cached)", b.CompletedSteps, b.TotalSteps, b.CachedSteps)
		}
		// Keep the name flexible while retaining the other fields on normal terminals.
		nameWidth := width - 58
		if nameWidth < 12 {
			nameWidth = 12
		}
		p.lines = append(p.lines, fmt.Sprintf("RUNNING  %s  %s  %8s  %s", buildField(name, nameWidth), buildField(b.Builder, 18), elapsed, steps))
	}
	if visible < len(rows) && available > 0 {
		p.lines = append(p.lines, fmt.Sprintf("+ %d more active builds", len(rows)-visible))
	}
	if warning != "" {
		p.lines = append(p.lines, warning)
	}
	p.Height = len(p.lines)
	return p
}

// buildField strips terminal controls and clips using display width, not bytes.
func buildField(s string, width int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
	s = runewidth.Truncate(s, width, "…")
	return s + strings.Repeat(" ", width-runewidth.StringWidth(s))
}

func (p *BuildPanel) Buffer() ui.Buffer {
	buf := p.Block.Buffer()
	for i, line := range p.lines {
		fg := ui.ThemeAttr("par.text.fg")
		if i == 0 {
			fg = ui.ThemeAttr("label.fg")
		}
		// Render literal cells so build names cannot inject termui markup.
		x := p.X
		for _, r := range buildField(line, p.Width) {
			w := runewidth.RuneWidth(r)
			if w == 0 {
				continue
			}
			buf.Set(x, p.Y+i, ui.Cell{Ch: r, Fg: fg, Bg: ui.ThemeAttr("par.text.bg")})
			x += w
		}
	}
	return buf
}
