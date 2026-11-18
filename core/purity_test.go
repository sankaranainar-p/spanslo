package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// R-008: non test code in core/ does no file or network I/O and never reads the wall clock.
func TestCorePurity(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, p := range files {
		if strings.HasSuffix(p, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, p, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			if path == "os" || path == "net" || path == "io/ioutil" ||
				strings.HasPrefix(path, "os/") || strings.HasPrefix(path, "net/") {
				t.Errorf("%s: imports %s", p, path)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok && s.Sel.Name == "Now" {
				if x, ok := s.X.(*ast.Ident); ok && x.Name == "time" {
					t.Errorf("%s: calls time.Now", fset.Position(s.Pos()))
				}
			}
			return true
		})
	}
}
