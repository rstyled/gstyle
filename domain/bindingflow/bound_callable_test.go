package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestBoundCallableSetsOwnCellsAndRetainEnvironmentAlternatives(t *testing.T) {
	target := Callable{Name: "body", Revision: "rev", Cells: []CallableCell{{Symbol: 2, Address: "b"}, {Symbol: 1, Address: "a"}}}
	set := NewBoundCallableSet(target)
	target.Cells[0].Address = "changed"
	targets := set.Targets()
	targets[0].Cells[0].Address = "also changed"
	if got := set.Targets()[0].Cells; len(got) != 2 || got[0].Address != "a" || got[1].Address != "b" {
		t.Fatal(got)
	}
	alternative := Callable{Name: "body", Revision: "rev", Cells: []CallableCell{{Symbol: 1, Address: "other"}}}
	combined := NewBoundCallableSet(append(set.Targets(), alternative)...)
	if len(combined.Targets()) != 2 || len(combined.Names()) != 1 {
		t.Fatal("code equality merges distinct environments")
	}
	for _, invalid := range []Callable{
		{Name: "body", Cells: []CallableCell{{Symbol: 1, Address: "a"}}},
		{Name: "body", Revision: "rev", Cells: []CallableCell{{Symbol: 0, Address: "a"}}},
		{Name: "body", Revision: "rev", Cells: []CallableCell{{Symbol: 1, Address: "a"}, {Symbol: 1, Address: "b"}}},
		{Name: "body", Revision: "rev", Cells: []CallableCell{{Symbol: pdg.Ref(1), Address: ""}}},
		{Name: "body", Revision: "rev", Cells: []CallableCell{{Symbol: 1, Address: string([]byte{255})}}},
	} {
		if NewBoundCallableSet(invalid).Known() {
			t.Fatalf("invalid environment: %+v", invalid)
		}
	}
}
