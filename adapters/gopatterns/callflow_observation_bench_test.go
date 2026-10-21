package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Independent conditions retain different forward states, but the observation
// result does not need a suffix completion set for each state and predecessor.
func BenchmarkOrdinaryObservationsBranchJoins(b *testing.B) {
	graph := parseIR(b, `package p
func f(a,b,c,d,e,f,g,h bool){
 if a {println("a")};if b {println("b")};if c {println("c")};if d {println("d")}
 if e {println("e")};if f {println("f")};if g {println("g")};if h {println("h")}
}`, pdg.NewBuilder())
	queries, err := callflow.NewQueries(&graph)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		got := callflow.OrdinaryObservations(queries, nil)
		if len(got.Calls) != 8 || len(got.Conditions) != 0 {
			b.Fatalf("lost branches or pruned unknown conditions: %+v", got)
		}
	}
}
