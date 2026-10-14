package gopatterns

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"golang.org/x/tools/go/packages"
)

func literalIndexTestCandidates(t *testing.T, src string) []patterns.Candidate {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}, Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Scopes: map[ast.Node]*types.Scope{}}
	pkg, err := (&types.Config{GoVersion: "go1.26"}).Check("example", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	ppkg := &packages.Package{Fset: fset, TypesInfo: info, Types: pkg}
	var out []patterns.Candidate
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			out = append(out, literalIndexCandidates(ppkg, fn, "test.go")...)
		}
	}
	return out
}

func TestLiteralIndex(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		find       bool
	}{
		{"table indexed by variable", `func String(s int) string { return [...]string{"a", "b"}[s] }`, true},
		{"slice literal with unrelated bound stays silent", `func At(s []int, i int) string { names := []string{"x", "y"}; if i < len(s) { return names[i] }; return "" }`, false},
		{"constant index is safe", `func First() string { return [...]string{"a", "b"}[0] }`, false},
		{"bounded elsewhere stays silent", `func String(s int) string { if s < 0 || s > 1 { return "?" }; return [...]string{"a", "b"}[s] }`, false},
		{"map literal is not an index", `func Get(m map[int]string, k int) string { return m[k] }`, false},
		{"no table at all", `func Get(p *int) int { return *p }`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package example\n" + tc.body + "\n"
			candidates := literalIndexTestCandidates(t, src)
			if (len(candidates) != 0) != tc.find {
				t.Fatalf("got %d candidates, want finding=%v", len(candidates), tc.find)
			}
			if !tc.find {
				return
			}
			c := candidates[0]
			if c.Kind != patterns.LiteralIndex {
				t.Errorf("kind=%q want literal_index", c.Kind)
			}
			if !strings.Contains(c.Observation, "unchecked variable") {
				t.Errorf("observation missing wording: %s", c.Observation)
			}
			if len(c.CounterEvidence) == 0 {
				t.Error("detection-only finding must carry counter-evidence")
			}
			if c.FixSpec != nil {
				t.Error("detection-only finding must not carry a fix spec")
			}
		})
	}
}
