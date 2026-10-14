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

func nilGuardTestCandidates(t *testing.T, src string) []patterns.Candidate {
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
	ppkg := &packages.Package{Fset: fset, TypesInfo: info, Types: pkg}
	var out []patterns.Candidate
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			out = append(out, nilGuardCandidates(ppkg, fn, "test.go")...)
		}
	}
	return out
}

func TestNilGuard(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		find       bool
	}{
		{"implicit field dereference", `type Item struct { N int }; func Get(p, q *Item) int { if q == nil { return -1 }; return p.N }`, true},
		{"parenthesized field dereference", `type Item struct { N int }; func Get(p, q *Item) int { if q == nil { return -1 }; return (p).N }`, true},
		{"new local is nonnil", `func Get(q *int) int { if q == nil { return -1 }; p := new(int); return *p }`, false},
		{"address local is nonnil", `func Get(q *int) int { if q == nil { return -1 }; n := 1; p := &n; return *p }`, false},
		{"reassigned local stays unknown", `func Get(q *int) int { if q == nil { return -1 }; p := new(int); p = nil; return *p }`, true},
		{"escaped local stays unknown", `func Get(q *int) int { if q == nil { return -1 }; p := new(int); overwrite(&p); return *p }; func overwrite(p **int) { *p = nil }`, true},
		{"closure written local stays unknown", `func Get(q *int) int { if q == nil { return -1 }; p := new(int); f := func() { p = nil }; f(); return *p }`, true},
		{"branch allocation does not dominate", `func Get(q *int) int { if q == nil { return -1 }; var p *int; if *q > 0 { p = new(int) }; return *p }`, true},
		{"shadowed new is not allocation", `func Get(q *int) int { if q == nil { return -1 }; new := func(int) *int { return nil }; p := new(1); return *p }`, true},
		{"nil receiver method may be safe", `type Item struct{}; func (p *Item) Value() int { return 0 }; func Get(p, q *Item) int { if q == nil { return -1 }; return p.Value() }`, false},
		{"partial return is not a guard", `func Get(p, q *int) int { if q == nil { if p == nil { return -1 } }; return *p }`, false},
		{"wrong variable guarded", `func Get(p, q *int) int { if q == nil { return -1 }; return *p }`, true},
		{"same variable guarded", `func Get(p *int) int { if p == nil { return -1 }; return *p }`, false},
		{"no guard at all", `func Get(p *int) int { return *p }`, false},
		{"error check is not a guard", `func Parse(text string) (int, error) { var err error; if err != nil { return 42, err }; return 12, nil }`, false},
		{"guard without return", `func Get(p, q *int) int { if q == nil { println("nil") }; return *p }`, false},
		{"reversed nil comparison", `func Get(p, q *int) int { if nil == q { return -1 }; return *p }`, true},
		{"multiple pointers one unguarded", `func Sum(a, b *int) int { if a == nil { return 0 }; return *a + *b }`, true},
		{"deref inside guard else stays guarded", `func Get(p *int) int { if p == nil { return -1 } else { return *p } }`, false},
		{"nil map index is not a deref", `func Get(m map[string]int, k string) int { var q *int; if q == nil { return -1 }; return m[k] }`, false},
		{"closure deref not attributed", `func Get(p, q *int) int { if q == nil { return -1 }; f := func() int { return *p }; return f() }`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package example\n" + tc.body + "\n"
			candidates := nilGuardTestCandidates(t, src)
			if (len(candidates) != 0) != tc.find {
				t.Fatalf("got %d candidates, want finding=%v", len(candidates), tc.find)
			}
			if !tc.find {
				return
			}
			c := candidates[0]
			if c.Kind != patterns.NilGuard {
				t.Errorf("kind=%q want nil_guard", c.Kind)
			}
			if !strings.Contains(c.Observation, "without a guard") {
				t.Errorf("observation missing guard wording: %s", c.Observation)
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
