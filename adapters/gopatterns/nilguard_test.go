package gopatterns

import (
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

func nilGuardTestCandidates(t *testing.T, src string) []patterns.Candidate {
	t.Helper()
	var out []patterns.Candidate
	for _, function := range runtimeTestGraphs(t, src) {
		report, err := runtimeguard.Analyze(&function.Pdg)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, nilGuardCandidates(function, report)...)
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
		{"range assignment invalidates allocation", `func Get(q *int) int { if q == nil { return -1 }; p := new(int); for _, p = range []*int{nil} {}; return *p }`, true},
		{"later guard cannot protect earlier read", `func Get(p,q *int) int { if q == nil { return -1 }; n := *p; if p == nil { return -1 }; return n }`, true},
		{"conditional guard does not dominate", `func Get(p,q *int,check bool) int { if q == nil { return -1 }; if check { if p == nil { return -1 } }; return *p }`, true},
		{"copied allocation stays nonnil", `func Get(q *int) int { if q == nil { return -1 }; p := new(int); r := p; return *r }`, false},
		{"allocations join as nonnil", `func Get(q *int,choose bool) int { if q == nil { return -1 }; var p *int; if choose { p = new(int) } else { p = new(int) }; return *p }`, false},
		{"pointer conversion preserves allocation", `type P *int; func Get(q *int) int { if q == nil { return -1 }; p := P(new(int)); return *p }`, false},
		{"short circuit protects dereference", `func Get(p,q *int) int { if q == nil { return -1 }; if p != nil && *p > 0 { return 1 }; return 0 }`, false},
		{"write also dereferences pointer", `func Set(p,q *int) { if q == nil { return }; *p = 1 }`, true},
		{"address dereference cancellation is safe", `func Get(p,q *int) *int { if q == nil { return nil }; return &*p }`, false},
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
