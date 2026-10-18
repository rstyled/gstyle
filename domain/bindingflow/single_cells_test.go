package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestSingleCellsNeedUniqueBooleanEvidence(t *testing.T) {
	pool := pdg.NewBuilder()
	q := &Query{graph: &pdg.Graph{Tables: pool.Tables}}
	for _, data := range []string{"null", "0", `"true"`, "{}", "[]", "{"} {
		q.graph.Symbols = []pdg.Symbol{{Role: pdg.LocalRole, Extensions: pool.SingletonExtension("gstyle.memory:single-cell", data)}}
		if _, err := q.singleCellSymbols(); err == nil {
			t.Fatalf("invalid marker %s accepted", data)
		}
	}
	for _, data := range []string{"true", "false"} {
		marker := pool.SingletonExtension("gstyle.memory:single-cell", data)
		q.graph.Symbols = []pdg.Symbol{{Role: pdg.LocalRole, Extensions: marker}}
		values, err := q.singleCellSymbols()
		if err != nil || values[1] != (data == "true") {
			t.Fatalf("marker=%s values=%v err=%v", data, values, err)
		}
		q.graph.Symbols[0].Extensions = append(marker, marker...)
		if _, err := q.singleCellSymbols(); err == nil {
			t.Fatal("duplicate markers accepted")
		}
	}
	q.graph.Symbols = []pdg.Symbol{{Role: pdg.CaptureRole}, {Role: pdg.LocalRole}}
	values, err := q.singleCellSymbols()
	if err != nil || len(values) != 0 {
		t.Fatalf("missing proof invents cells: %v %v", values, err)
	}
}
