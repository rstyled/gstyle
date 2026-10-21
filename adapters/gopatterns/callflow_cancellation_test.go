package gopatterns

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCallflowCancellationReturnsNoResults(t *testing.T) {
	graph := parseIR(t, `package p;func f(){println("ok")}`, pdg.NewBuilder())
	graphs := map[string]*pdg.Graph{graph.Tables.Text(graph.Function.ID): &graph}
	queries, err := callflow.NewQueries(&graph)
	if err != nil {
		t.Fatal(err)
	}
	program := callflow.NewProgram()
	if err := program.Add(&graph); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, solve := range []func(context.Context, map[string]*pdg.Graph) (callflow.ExitSummaries, error){callflow.ExitEffectsContext, program.ExitEffectsContext} {
		result, err := solve(ctx, graphs)
		if !errors.Is(err, context.Canceled) || result != nil {
			t.Fatalf("got %v, %v", result, err)
		}
	}
	observed, err := callflow.OrdinaryObservationsContext(ctx, queries, nil)
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(observed, callflow.Observations{}) {
		t.Fatalf("got %+v, %v", observed, err)
	}
	// A canceled solve must not poison the reusable program or its indexes.
	want, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	got, err := program.ExitEffectsContext(context.Background(), graphs)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("retry: got %+v, %v; want %+v", got, err, want)
	}
}
