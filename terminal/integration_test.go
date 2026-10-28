package terminal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// Guard every view, including new ones, against bypassing the cache or removing
// the resize fallback when it resets event handlers.
func TestApplicationUsesDimensionIntegration(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filename))
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "vendor" || path == filepath.Join(root, "terminal")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		alias := ""
		for _, imp := range f.Imports {
			if imp.Path.Value == `"github.com/gizak/termui"` {
				alias = "termui"
				if imp.Name != nil {
					alias = imp.Name.Name
				}
			}
		}
		if alias == "" {
			return nil
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver := sel.X
			if nested, ok := receiver.(*ast.SelectorExpr); ok && nested.Sel.Name == "DefaultEvtStream" {
				receiver = nested.X
			}
			ident, ok := receiver.(*ast.Ident)
			if !ok || ident.Name != alias {
				return true
			}
			switch sel.Sel.Name {
			case "TermWidth", "TermHeight", "ResetHandlers":
				t.Errorf("%s: use the terminal integration instead of %s", fset.Position(call.Pos()), sel.Sel.Name)
			case "Handle":
				if len(call.Args) > 0 {
					if arg, ok := call.Args[0].(*ast.BasicLit); ok {
						value, _ := strconv.Unquote(arg.Value)
						if value == "/sys/wnd/resize" {
							t.Errorf("%s: register resize through terminal.HandleResize", fset.Position(call.Pos()))
						}
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
