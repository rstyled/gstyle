package gopatterns

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestSingleCellSourceProofs(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		single     bool
	}{
		{"ordinary local", `cell:=false;_=cell`, true},
		{"loop body", `for choose{cell:=false;_=cell}`, false},
		{"loop header", `for cell:=false;cell;{}`, false},
		{"range declaration", `for _,cell:=range []bool{false}{_=cell}`, false},
		{"range assignment retains one cell", `cell:=false;for _,cell=range []bool{false}{_=cell}`, true},
		{"outer cell across loop", `cell:=false;for choose{cell=true};_=cell`, true},
		{"goto may repeat", `again:cell:=false;_=cell;if choose{goto again}`, false},
		{"unrelated goto conservative", `cell:=false;goto done;done:_=cell`, false},
		{"nested closure loop does not repeat outer cell", `cell:=false;_=func(){for choose{_=cell}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var choose bool;func f(){`+tc.body+`}`)
			graph := continuationGraph(t, extraction, "example.f")
			query, err := bindingflow.New(graph)
			if err != nil {
				t.Fatal(err)
			}
			facts, err := query.ValueFacts()
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for i, symbol := range graph.Symbols {
				if graph.Tables.Text(symbol.Name.Value) != "cell" {
					continue
				}
				found = true
				if got := facts.SingleCellSymbols[pdg.Ref(i+1)]; got != tc.single {
					t.Fatalf("single cell=%v want %v", got, tc.single)
				}
			}
			if !found {
				t.Fatal("missing cell symbol")
			}
		})
	}
}
