package bindingflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestResolvedBooleanFactBoundaries(t *testing.T) {
	for _, value := range []string{`null`, `{}`, `[]`, `{"primitive":null}`, `{"primitive":true}`, `{"primitive":""}`, `{"type":"bool"}`, `{"primitive":"bool","extra":1}`, `{"primitive":"bool","primitive":"isize"}`, `{"primitive":"isize","primitive":"bool"}`, `{"primitive":"bool"} {}`} {
		if _, err := decodePrimitive(value); err == nil {
			t.Errorf("accepts invalid primitive record %s", value)
		}
	}
}

func TestBooleanFactsUseResolvedTypes(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables, Symbols: []pdg.Symbol{
		{Type: pool.Known("NamedFlag"), Extensions: pool.SingletonExtension("gstyle.go.scalar-normal:resolved-type", `{"primitive":"bool"}`)},
		{Type: pool.Known("bool")},
		{Type: pool.Known("bool"), Extensions: pool.SingletonExtension("gstyle.go.scalar-normal:resolved-type", `{"primitive":"isize"}`)},
	}}
	query := &Query{graph: graph}
	facts, err := query.ValueFacts()
	if err != nil {
		t.Fatal(err)
	}
	if !facts.BooleanSymbols[1] || facts.BooleanSymbols[2] || facts.BooleanSymbols[3] {
		t.Fatalf("Boolean type facts: %v", facts.BooleanSymbols)
	}
	graph.Symbols[0].Extensions = append(graph.Symbols[0].Extensions, pool.SingletonExtension("gstyle.go.scalar-normal:resolved-type", `{"primitive":"bool"}`)...)
	if _, err := query.ValueFacts(); err == nil {
		t.Fatal("duplicate type records are accepted")
	}
}
