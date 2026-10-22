package gopatterns

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

// Each independent Boolean doubles the concrete paths. An opaque prefix makes
// exit summaries stop, while ordinary observations must inspect its continuation.
func observationBranchSource(count int) string {
	var parameters, body strings.Builder
	for i := range count {
		fmt.Fprintf(&parameters, "b%d bool,", i)
		fmt.Fprintf(&body, "if b%d { println(%d) };", i, i)
	}
	return fmt.Sprintf(`package example
var foreign func()
func f(%s) {
 foreign()
 %s
 if b0 { panic("selected arm") }
 println("reachable")
}`, parameters.String(), body.String())
}

func TestObservationLimitDiscardsPartialFacts(t *testing.T) {
	for _, count := range []int{4, 20} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			graph := parseIR(t, observationBranchSource(count), pdg.NewBuilder())
			queries, err := callflow.NewQueries(&graph)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			got, err := callflow.OrdinaryObservationsContext(ctx, queries, nil)
			if err != nil {
				t.Fatal(err)
			}
			if count == 4 {
				if got.Limited || len(got.Calls) != count+3 || len(got.Conditions) != 0 {
					t.Fatalf("small traversal lost facts or pruned an unknown condition: %+v", got)
				}
				return
			}
			if !got.Limited || got.Calls != nil || got.Operations != nil || got.Conditions != nil {
				t.Fatalf("limited traversal published partial proof facts: %+v", got)
			}
		})
	}
}

func TestObservationLimitRetainsConservativeGraph(t *testing.T) {
	extraction, _ := proofExtraction(t, observationBranchSource(20))
	graph := continuationGraph(t, extraction, "example.f")
	if err := pdg.Validate(graph); err != nil {
		t.Fatal(err)
	}
	limited := false
	for _, diagnostic := range graph.Diagnostics {
		limited = limited || graph.Tables.Text(diagnostic.Code) == "pdg.observation-work-limit"
	}
	if !limited {
		t.Fatal("missing diagnostic for conservative fallback")
	}
	// The selected panic arm does not make the final println unreachable.
	// A partial traversal must not remove either arm or invent a constant.
	calls := 0
	for _, block := range graph.ValueFlow.Blocks {
		if !block.Reachable {
			continue
		}
		for _, ref := range block.Operations {
			attrs := graph.NodeAttributes(graph.Nodes[ref-1])
			if graph.Tables.Text(attrs.Callee.Value) == "println" {
				calls++
			}
		}
	}
	if calls != 21 {
		t.Fatalf("lost reachable calls after fallback: %d", calls)
	}
}

func BenchmarkOrdinaryObservationsLimited(b *testing.B) {
	graph := parseIR(b, observationBranchSource(20), pdg.NewBuilder())
	queries, err := callflow.NewQueries(&graph)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		got := callflow.OrdinaryObservations(queries, nil)
		if !got.Limited || len(got.Calls)+len(got.Operations)+len(got.Conditions) != 0 {
			b.Fatal("limited traversal published partial proof facts")
		}
	}
}
