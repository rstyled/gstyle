package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestCanonicalCallExitSummaries(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       callflow.Outcomes
	}{
		{"return", "return", callflow.Returns},
		{"panic", `panic("bad")`, callflow.Panics},
		{"goexit", `runtime.Goexit()`, callflow.ExitsGoroutine},
		{"abort", `os.Exit(1)`, callflow.Aborts},
		{"deferred panic", `defer panic("bad");return`, callflow.Panics},
		{"deferred goexit", `defer runtime.Goexit();return`, callflow.ExitsGoroutine},
		{"deferred abort", `defer os.Exit(1);return`, callflow.Aborts},
		{"loop", `for { println("live") }`, callflow.Diverges},
		{"unknown", `foreign();return`, callflow.UnknownExit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := `package p;import("runtime";"os");func f(x *int){` + tc.body + `};func foreign();var _ = runtime.Goexit;var _ = os.Exit`
			graph := parseIR(t, source, pdg.NewBuilder())
			name := graph.Tables.Text(graph.Function.ID)
			summaries, err := callflow.ExitEffects(map[string]*pdg.Graph{name: &graph})
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries[name].Ordinary; got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
func TestDeferredRegistrationRequiresItsCaptureFact(t *testing.T) {
	graph := parseIR(t, `package p;func f(x *int){defer println("x");return}`, pdg.NewBuilder())
	sites, err := deferflow.Sites(&graph)
	if err != nil || len(sites) != 1 {
		t.Fatalf("sites %v %v", sites, err)
	}
	node := graph.Nodes[sites[0].Registration-1]
	attrs := graph.Attributes[node.Attributes-1]
	extensions := graph.AttributeTables.Extensions[attrs.Extensions-1]
	var kept []pdg.Extension
	for _, extension := range extensions {
		if graph.Tables.Text(extension.Name) != deferflow.Extension {
			kept = append(kept, extension)
		}
	}
	graph.AttributeTables.Extensions[attrs.Extensions-1] = kept
	if _, err := deferflow.Sites(&graph); err == nil {
		t.Fatal("missing capture fact accepted")
	}
	if _, err := callflow.ExitEffects(map[string]*pdg.Graph{graph.Tables.Text(graph.Function.ID): &graph}); err == nil {
		t.Fatal("exit query accepted missing registration fact")
	}
}

func TestExitSummariesComposeLocalBodies(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example
 import "runtime"
 func normal(x *int){return}
 func fails(x *int){panic("bad")}
 func stops(x *int){runtime.Goexit()}
 func chain(x *int){fails(x)}
 func unwind(x *int){defer fails(x);normal(x);return}
 func barrier(x *int){defer fails(x);stops(x)}
 func recursive(x *int){recursive(x)}
 func recovers(x *int){recover();return}
 `)
	graphs := map[string]*pdg.Graph{}
	for i := range extraction.Funcs {
		graph := &extraction.Funcs[i].Pdg
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]callflow.Outcomes{
		"normal": callflow.Returns, "fails": callflow.Panics, "stops": callflow.ExitsGoroutine,
		"chain": callflow.Panics, "unwind": callflow.Panics, "barrier": callflow.PanicDuringGoexit, "recursive": callflow.Diverges,
		"recovers": callflow.Returns,
	} {
		if got := summaries["example."+name].Ordinary; got != want {
			t.Errorf("%s got %v want %v", name, got, want)
		}
	}
	if got := summaries["example.recovers"].DeferredPanic; got != callflow.RecoversInherited {
		t.Fatalf("recovery context got %v", got)
	}
}

func TestScalarCyclePublishesExitBodyFlow(t *testing.T) {
	graph := parseIR(t, `package p;func f(){for{}}`, pdg.NewBuilder())
	if pdg.ExitBodyFlow(&graph) == nil {
		t.Fatal("scalar fixed-point body has no exit context")
	}
	name := graph.Tables.Text(graph.Function.ID)
	summaries, err := callflow.ExitEffects(map[string]*pdg.Graph{name: &graph})
	if err != nil {
		t.Fatal(err)
	}
	if got := summaries[name].Ordinary; got != callflow.Diverges {
		t.Fatalf("pure cycle exit=%v want divergence", got)
	}
}
