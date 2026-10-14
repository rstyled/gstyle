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

func negativeIndexTestCandidates(t *testing.T, src string) []patterns.Candidate {
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
			out = append(out, negativeIndexCandidates(ppkg, fn, "test.go")...)
		}
	}
	return out
}

func TestNegativeIndex(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		find       bool
	}{
		{"rejection guard", `func At(s []int, i int) int { if i >= len(s) { return -1 }; return s[i] }`, true},
		{"reversed rejection guard", `func At(s []int, i int) int { if len(s) <= i { return -1 }; return s[i] }`, true},
		{"parenthesized rejection", `func At(s []int, i int) int { if (i >= len(s)) { return -1 }; return s[i] }`, true},
		{"combined rejection", `func At(s []int, i int) int { if i < 0 || i >= len(s) { return -1 }; return s[i] }`, false},
		{"unsigned array index cannot be negative", `func At(s [2]int, i uint) int { if i < uint(len(s)) { return s[i] }; return -1 }`, false},
		{"named unsigned index cannot be negative", `type Index uint; func At(s [2]int, i Index) int { if i < Index(len(s)) { return s[i] }; return -1 }`, false},
		{"shadowed length is not builtin", `func At(s []int, i int) int { len := func([]int) int { return 1 }; if i < len(s) { return s[i] }; return -1 }`, false},
		{"missing lower bound", `func At(s []int, i int) int { if i < len(s) { return s[i] }; return -1 }`, true},
		{"reversed comparison", `func At(s []int, i int) int { if len(s) > i { return s[i] }; return -1 }`, true},
		{"combined guard present", `func At(s []int, i int) int { if i >= 0 && i < len(s) { return s[i] }; return -1 }`, false},
		{"separate rejection", `func At(s []int, i int) int { if i < 0 { return -1 }; if i < len(s) { return s[i] }; return -1 }`, false},
		{"no bound check at all", `func At(s []int, i int) int { return s[i] }`, false},
		{"no index use", `func Len(s []int, i int) int { if i < len(s) { return len(s) }; return -1 }`, false},
		{"non-integer index", `func First(s []string, f float64) float64 { if f < float64(len(s)) { return f }; return -1 }`, false},
		{"guard without return", `func At(s []int, i int) int { x := 0; if i < len(s) { x = s[i] }; return x }`, false},
		{"different index guarded", `func At(s []int, i, j int) int { if j < len(s) { return s[j] }; if i < len(s) { return s[i] }; return -1 }`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package example\n" + tc.body + "\n"
			candidates := negativeIndexTestCandidates(t, src)
			if (len(candidates) != 0) != tc.find {
				t.Fatalf("got %d candidates, want finding=%v", len(candidates), tc.find)
			}
			if !tc.find {
				return
			}
			c := candidates[0]
			if c.Kind != patterns.NegativeIndex {
				t.Errorf("kind=%q want negative_index", c.Kind)
			}
			if !strings.Contains(c.Observation, "no lower-bound check") {
				t.Errorf("observation missing lower-bound wording: %s", c.Observation)
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

func TestNegativeIndexOneFindingPerBinding(t *testing.T) {
	src := `package example
 func At(s []int, i int) int {
  if i >= len(s) { return -1 }
  if i < len(s) { return s[i] }
  return -1
 }`
	for range 20 {
		got := negativeIndexTestCandidates(t, src)
		if len(got) != 1 {
			t.Fatalf("got %d findings, want one per binding", len(got))
		}
	}
}
