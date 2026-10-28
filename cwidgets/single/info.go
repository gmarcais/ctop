package single

import (
	"strings"

	ui "github.com/gizak/termui"
)

var displayInfo = []string{"id", "name", "image", "ports", "IPs", "state", "created", "uptime", "health"}

type Info struct {
	*ui.Table
	data map[string]string
}

func NewInfo() *Info {
	p := ui.NewTable()
	p.Height = 4
	p.Width = defaultPanelWidth
	p.FgColor = ui.ThemeAttr("par.text.fg")
	p.Separator = false
	i := &Info{p, make(map[string]string)}
	return i
}

func (w *Info) Set(k, v string) {
	w.data[k] = v

	// rebuild rows
	w.Rows = [][]string{}
	for _, k := range displayInfo {
		if v, ok := w.data[k]; ok {
			w.Rows = append(w.Rows, mkInfoRows(k, v)...)
		}
	}

	w.Height = len(w.Rows) + 2
}

func (w *Info) Buffer() ui.Buffer { return tableBuffer(w.Table) }

// termui tables can draw long values over their borders. Clip the table body
// separately so narrower layouts keep their panel edges intact.
func tableBuffer(table *ui.Table) ui.Buffer {
	buf := table.Buffer()
	inner := table.InnerBounds()
	for p := range buf.CellMap {
		if !p.In(inner) {
			delete(buf.CellMap, p)
		}
	}
	for p, cell := range table.Block.Buffer().CellMap {
		if !p.In(inner) {
			buf.Set(p.X, p.Y, cell)
		}
	}
	return buf
}

// Build row(s) from a key and value string
func mkInfoRows(k, v string) (rows [][]string) {
	lines := strings.Split(v, "\n")

	// initial row with field name
	rows = append(rows, []string{k, lines[0]})

	// append any additional lines in separate row
	if len(lines) > 1 {
		for _, line := range lines[1:] {
			if line != "" {
				rows = append(rows, []string{"", line})
			}
		}
	}

	return rows
}
