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
		{"named constant is compiler checked", `const slot = 1; func First() string { return [...]string{"a", "b"}[slot] }`, false},
		{"constant expression is compiler checked", `const slot = 0; func First() string { return [...]string{"a", "b"}[slot+1] }`, false},
		{"only lower bound still unsafe", `func String(s int) string { if s < 0 { return "?" }; return [...]string{"a", "b"}[s] }`, true},
		{"only upper bound still unsafe", `func String(s int) string { if s >= 2 { return "?" }; return [...]string{"a", "b"}[s] }`, true},
		{"unsigned upper bound sufficient", `func String(s uint) string { if s >= 2 { return "?" }; return [...]string{"a", "b"}[s] }`, false},
		{"loose upper bound still unsafe", `func String(s int) string { if s < 0 || s >= 3 { return "?" }; return [...]string{"a", "b"}[s] }`, true},
		{"comparison without rejection insufficient", `func String(s int) string { if s < 0 || s >= 2 { println("invalid") }; return [...]string{"a", "b"}[s] }`, true},
		{"guard after access insufficient", `func String(s int) string { x := [...]string{"a", "b"}[s]; if s < 0 || s >= 2 { return "?" }; return x }`, true},
		{"separate guards sufficient", `func String(s int) string { if s < 0 { return "?" }; if s >= 2 { return "?" }; return [...]string{"a", "b"}[s] }`, false},
		{"acceptance branch sufficient", `func String(s int) string { if s >= 0 && s < 2 { return [...]string{"a", "b"}[s] }; return "?" }`, false},
		{"reversed acceptance branch sufficient", `func String(s int) string { if 0 <= s && 2 > s { return [...]string{"a", "b"}[s] }; return "?" }`, false},
		{"equality acceptance sufficient", `func String(s int) string { if s == 1 { return [...]string{"a", "b"}[s] }; return "?" }`, false},
		{"else branch sufficient", `func String(s int) string { if s < 0 || s >= 2 { return "?" } else { return [...]string{"a", "b"}[s] } }`, false},
		{"negated acceptance sufficient", `func String(s int) string { if !(s >= 0 && s < 2) { return "?" }; return [...]string{"a", "b"}[s] }`, false},
		{"reassignment invalidates bounds", `func String(s int) string { if s < 0 || s >= 2 { return "?" }; s = 3; return [...]string{"a", "b"}[s] }`, true},
		{"increment invalidates bounds", `func String(s int) string { if s < 0 || s >= 2 { return "?" }; s++; return [...]string{"a", "b"}[s] }`, true},
		{"address escape invalidates bounds", `func String(s int) string { if s < 0 || s >= 2 { return "?" }; alter(&s); return [...]string{"a", "b"}[s] }; func alter(s *int) { *s = 3 }`, true},
		{"shadowed index not bounded", `func String(s int) string { if s < 0 || s >= 2 { return "?" }; { s := 3; return [...]string{"a", "b"}[s] } }`, true},
		{"keyed slice size retained", `func String(s int) string { if s < 0 || s >= 4 { return "?" }; return []string{3:"b"}[s] }`, false},
		{"named array literal", `type Names [2]string; func String(s int) string { return Names{"a", "b"}[s] }`, true},
		{"unsigned lower bound survives writes", `func String(s uint) string { s = 1; if s >= 2 { return "?" }; return [...]string{"a", "b"}[s] }`, false},
		{"signed type parameter needs lower bound", `func String[T ~int](s T) string { if s >= 2 { return "?" }; return [...]string{"a", "b"}[s] }`, true},
		{"mutation in continuing else invalidates", `func String(s int) string { if s < 0 || s >= 2 { return "?" } else { s = 3 }; return [...]string{"a", "b"}[s] }`, true},
		{"closure mutation after guard stays unknown", `func String(s int) string { alter := func() { s = 3 }; if s < 0 || s >= 2 { return "?" }; alter(); return [...]string{"a", "b"}[s] }`, true},
		{"global mutation after guard stays unknown", `var s int; func alter() { s = 3 }; func String() string { if s < 0 || s >= 2 { return "?" }; alter(); return [...]string{"a", "b"}[s] }`, true},
		{"earlier address escape stays unknown", `func String(s int) string { ptr := &s; if s < 0 || s >= 2 { return "?" }; *ptr = 3; return [...]string{"a", "b"}[s] }`, true},
		{"range write invalidates bounds", `func String(s int) string { if s < 0 || s >= 2 { return "?" }; for s = range [3]int{} {}; return [...]string{"a", "b"}[s] }`, true},
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
