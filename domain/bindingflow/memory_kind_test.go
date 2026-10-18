package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestImmutableMemoryKindRequiresUnambiguousCompilerFact(t *testing.T) {
	for _, tc := range []struct {
		name  string
		facts []string
		want  bool
	}{
		{"missing", nil, false}, {"immutable", []string{`"immutable"`}, true}, {"pointer", []string{`"pointer"`}, false}, {"unknown", []string{`"unknown"`}, false}, {"malformed", []string{`immutable`}, false}, {"conflicting", []string{`"immutable"`, `"pointer"`}, false}, {"duplicate", []string{`"immutable"`, `"immutable"`}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := pdg.NewBuilder()
			var extensions []pdg.Extension
			for _, fact := range tc.facts {
				extensions = append(extensions, pdg.Extension{Name: pool.Text("gstyle.memory:kind"), Value: pool.Text(fact)})
			}
			if got := immutableMemoryKind(pool.Tables, extensions); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestMemoryKindRetainsExactCompilerKind(t *testing.T) {
	for _, kind := range []string{"function", "interface", "immutable", "pointer"} {
		pool := pdg.NewBuilder()
		facts := []pdg.Extension{{Name: pool.Text("gstyle.memory:kind"), Value: pool.Text(`"` + kind + `"`)}}
		if got := sourceMemoryKind(pool.Tables, facts); got != kind {
			t.Fatalf("got %q want %q", got, kind)
		}
		if got := sourceMemoryKind(pool.Tables, append(facts, facts...)); got != "" {
			t.Fatalf("duplicate fact established %q", got)
		}
	}
	var absent *Query
	for _, q := range []*Query{absent, {}} {
		for _, ref := range []pdg.Ref{0, 1} {
			if q.MemoryKind(ref) != "" || q.ImmutableValue(ref) || q.ImmutableSymbol(ref) {
				t.Fatal("missing graph established memory kind")
			}
		}
	}
}
