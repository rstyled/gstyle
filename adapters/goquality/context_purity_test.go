package goquality

import (
	"context"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestRecursiveOwnedWorkingMap(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
func walk(seen map[int]bool, n int) { if seen[n] {return}; seen[n]=true; if n>0 { walk(seen,n-1) } }
func Owned() { walk(make(map[int]bool),4) }
func Shared(seen map[int]bool) { walk(seen,4) }
func Mixed(seen map[int]bool) { walk(make(map[int]bool),4); walk(seen,4) }
func ReverseMixed(seen map[int]bool) { walk(seen,4); walk(make(map[int]bool),4) }
func noisy(seen map[int]bool,n int) {seen[n]=true; if n>0 {noisy(seen,n-1)}; println(n)}
func Effectful() {noisy(make(map[int]bool),4)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	got := resultNamed(t, report, "Owned")
	if !got.Complete || len(got.Effects) != 0 {
		t.Fatalf("owned working data: %+v", got)
	}
	for _, name := range []string{"Shared", "Mixed", "ReverseMixed", "Effectful"} {
		got := resultNamed(t, report, name)
		if got.Complete || len(got.Effects) == 0 {
			t.Fatalf("%s loses evidence: %+v", name, got)
		}
	}
}
