package gopatterns

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

func scopeFixture(t *testing.T, root string, padding bool) []string {
	t.Helper()
	fset := token.NewFileSet()
	if padding {
		fset.AddFile("unrelated.go", -1, 4096)
	}
	source := "package example; var global int; func f(x int, y int) int { { x := y; global = x }; return x + len(\"\") }"
	file, err := parser.ParseFile(fset, root+"/example.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Scopes: map[ast.Node]*types.Scope{}}
	pkg, err := (&types.Config{}).Check("example", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	loaded := &packages.Package{PkgPath: "example", Fset: fset, TypesInfo: info, Types: pkg}
	graph, err := buildGraph(loaded, file.Decls[1].(*ast.FuncDecl), "example.go", pdg.NewBuilder())
	if err != nil {
		t.Fatal(err)
	}
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	var scopes []string
	for _, symbol := range graph.Symbols {
		scopes = append(scopes, graph.Tables.Text(symbol.Scope))
	}
	return scopes
}

func TestScopeIDsIgnoreCheckoutAndFileSetOrder(t *testing.T) {
	first := scopeFixture(t, "/checkout/one", false)
	second := scopeFixture(t, "/different/checkout", true)
	if strings.Join(first, "\n") != strings.Join(second, "\n") {
		t.Fatalf("scope IDs differ: %v / %v", first, second)
	}
	if len(first) < 4 {
		t.Fatalf("missing bindings: %v", first)
	}
	if first[0] != first[1] {
		t.Fatalf("parameters have different scopes: %v", first)
	}
	if first[0] == first[2] {
		t.Fatalf("shadowed binding shares outer scope: %v", first)
	}
	if !strings.HasPrefix(first[0], "scope:example.go:lexical:") {
		t.Fatalf("scope lacks relative source range: %v", first)
	}
	if !strings.Contains(strings.Join(first, "\n"), "scope:package:example") {
		t.Fatalf("missing package scope: %v", first)
	}
}
