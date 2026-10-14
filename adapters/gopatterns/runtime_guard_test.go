package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"os"
	"path/filepath"
	"testing"
)

func runtimeTestGraphs(t *testing.T, src string) []FuncPdg {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Scopes: map[ast.Node]*types.Scope{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	pkg, err := (&types.Config{GoVersion: "go1.26"}).Check("example", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	source := &packages.Package{Fset: fset, TypesInfo: info, Types: pkg, PkgPath: "example"}
	pool := pdg.NewBuilder()
	var out []FuncPdg
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		graph, err := buildGraph(source, fn, "test.go", pool)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, FuncPdg{Name: patterns.FuncID(info.Defs[fn.Name].(*types.Func)), Path: "test.go", Pdg: graph})
	}
	return out
}

func TestRuntimeGuardCachedExtraction(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":    "module example\n\ngo 1.26\n",
		"sample.go": `package example; func Get(p,q *int) int { if q == nil { return 0 }; return *p }`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cache := &FixExtractor{}
	for range 2 {
		extraction, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
		if err != nil {
			t.Fatal(err)
		}
		if len(extraction.NilGuards) != 1 {
			t.Fatalf("cached extraction has %d nil guards", len(extraction.NilGuards))
		}
	}
}
