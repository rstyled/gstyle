package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestCapturedMarkersRequireUniqueBooleanRecords(t *testing.T) {
	pool := pdg.NewBuilder()
	q := &Query{graph: &pdg.Graph{Tables: pool.Tables}}
	for _, data := range []string{"null", "7", `"true"`, "[]", "{}", "{"} {
		if _, err := q.captureMarker(pool.SingletonExtension("gstyle.memory:captured", data)); err == nil {
			t.Fatalf("invalid capture marker %s accepted", data)
		}
	}
	for _, data := range []string{"true", "false"} {
		marker := pool.SingletonExtension("gstyle.memory:captured", data)
		value, err := q.captureMarker(marker)
		if err != nil || value != (data == "true") {
			t.Fatalf("marker=%v error=%v", value, err)
		}
		if _, err := q.captureMarker(append(marker, marker...)); err == nil {
			t.Fatal("duplicate capture markers accepted")
		}
	}
	q.graph.Symbols = []pdg.Symbol{{Role: pdg.CaptureRole}, {Role: pdg.LocalRole, Extensions: pool.SingletonExtension("gstyle.memory:captured", "true")}, {Role: pdg.LocalRole}}
	symbols, err := q.capturedSymbols()
	if err != nil || len(symbols) != 2 || !symbols[1] || !symbols[2] {
		t.Fatalf("captured symbol inventory=%v %v", symbols, err)
	}
}
