package gopatterns

import (
	"context"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryPreservationComposesCallEffects(t *testing.T) {
	for _, tc := range []struct {
		name, body, helpers string
		want                callflow.Outcomes
		preserves           bool
	}{
		{"empty body", `noop();fn()`, `func noop(){}`, callflow.Returns, true},
		{"private assignment", `noop();fn()`, `func noop(){value:=false;value=true;_=value}`, callflow.Returns, true},
		{"parameter assignment", `noop(true);fn()`, `func noop(value bool){value=false;_=value}`, callflow.Returns, true},
		{"transitive body", `noop();fn()`, `func noop(){helper()};func helper(){}`, callflow.Returns, true},
		{"recursive component", `noop(false);fn()`, `func noop(flag bool){if flag{helper(flag)}};func helper(flag bool){if flag{noop(flag)}}`, callflow.Returns, true},
		{"captured reader", `read:=func()bool{return flag};_=read();fn()`, `func noop(){}`, callflow.Returns, true},
		{"deferred reader", `noop();fn()`, `func noop(){value:=false;defer func(){_=value}()}`, callflow.Returns, true},
		{"global writer", `noop();fn()`, `func noop(){global=true}`, callflow.Returns | callflow.Panics, false},
		{"transitive writer", `noop();fn()`, `func noop(){helper()};func helper(){global=true}`, callflow.Returns | callflow.Panics, false},
		{"recursive writer excluded by argument", `noop(false);fn()`, `func noop(value bool){if value{helper(value)}};func helper(value bool){global=true;if value{noop(value)}}`, callflow.Returns, false},
		{"recursive writer reached by argument", `noop(true);fn()`, `func noop(value bool){if value{helper(false)}};func helper(value bool){global=true;if value{noop(value)}}`, callflow.Returns | callflow.Panics, false},
		{"deferred writer", `noop();fn()`, `func noop(){defer func(){global=true}()}`, callflow.Returns | callflow.Panics, false},
		{"indirect writer", `noop(&flag);fn()`, `func noop(value *bool){*value=true}`, callflow.Returns | callflow.Panics, false},
		{"captured writer", `write:=func(){flag=true};write();fn()`, `func noop(){global=true}`, callflow.Panics, false},
		{"private indirect store stays conservative", `noop();fn()`, `func noop(){value:=false;p:=&value;*p=true}`, callflow.Returns | callflow.Panics, false},
		{"opaque target", `noop();fn()`, `var noop func()`, callflow.UnknownExit, false},
		{"body with opaque call", `noop();fn()`, `func noop(){foreign()};var foreign func()`, callflow.UnknownExit, false},
		{"builtin len", `_=len("x");noop();fn()`, `func noop(){}`, callflow.Returns, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;var global bool;func f(){flag:=false;fn:=func(){if flag{panic("unreachable")}};`+tc.body+`};`+tc.helpers)
			graphs := map[string]*pdg.Graph{}
			for _, graph := range extraction.Callables {
				graphs[graph.Tables.Text(graph.Function.ID)] = graph
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			if got := summaries["example.f"].Ordinary; got != tc.want {
				t.Fatalf("exit=%v want %v", got, tc.want)
			}
			if got := summaries["example.noop"].PreservesCallerMemory; got != tc.preserves {
				t.Fatalf("memory preserved=%v want %v", got, tc.preserves)
			}
		})
	}
}

func TestMemoryPreservationNeedsActiveBodyCatalog(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func noop(){};func f(){flag:=false;fn:=func(){if flag{panic("maybe")}};noop();if flag{flag=false};fn()}`)
	graphs := map[string]*pdg.Graph{}
	for _, graph := range extraction.Callables {
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	delete(graphs, "example.noop")
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	graph := graphs["example.f"]
	queries, err := callflow.NewQueries(graph)
	if err != nil {
		t.Fatal(err)
	}
	observed := callflow.OrdinaryObservations(queries, summaries)
	if len(observed.Conditions) != 0 {
		t.Fatal("missing body preserves a constant condition after its call")
	}
	found := false
	for ref, outcome := range observed.Calls {
		attrs := graph.NodeAttributes(graph.Nodes[ref-1])
		if graph.Tables.Text(attrs.Callee.Value) == "example.noop" {
			continue
		}
		found = true
		if outcome != callflow.Returns|callflow.Panics {
			t.Fatalf("incomplete catalog observes exit=%v", outcome)
		}
	}
	if !found {
		t.Fatal("missing closure after opaque prefix")
	}
}

func TestMemoryPreservationAcrossDependencyRevisions(t *testing.T) {
	root := crossPackageModule(t, `package stop;func Now(flag bool){}`)
	caller := filepath.Join(root, "caller.go")
	source := `package example;import "example/middle";func f(){flag:=false;fn:=func(){if flag{panic("bad")}};middle.Stop(true);fn()}`
	if err := os.WriteFile(caller, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	cache := &FixExtractor{}
	first, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, first, callflow.Returns)
	files := map[string][]byte{filepath.Join(root, "stop/stop.go"): []byte(`package stop;var global bool;func Now(flag bool){global=true}`)}
	second, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: first.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, second, callflow.Returns|callflow.Panics)
	files[filepath.Join(root, "stop/stop.go")] = []byte(`package stop;func Now(flag bool){}`)
	third, err := cache.Extract(context.Background(), root, []string{"."}, SourceOverlay{Files: files, Disk: second.Sources.EvidenceOnly()})
	if err != nil {
		t.Fatal(err)
	}
	checkMemoryRevision(t, third, callflow.Returns)
}

func checkMemoryRevision(t *testing.T, extraction *Extraction, want callflow.Outcomes) {
	t.Helper()
	program := callflow.NewProgram()
	var caller *pdg.Graph
	for _, graph := range extraction.Callables {
		if graph.Tables.Text(graph.Function.ID) == "example.f" {
			caller = graph
		}
		if err := program.Add(graph); err != nil {
			t.Fatal(err)
		}
	}
	if caller == nil {
		t.Fatal("missing caller")
	}
	summaries, err := program.ExitEffects(map[string]*pdg.Graph{"example.f": caller})
	if err != nil {
		t.Fatal(err)
	}
	if got := summaries["example.f"].Ordinary; got != want {
		t.Fatalf("revision exit=%v want %v", got, want)
	}
}

func TestPreservedMemoryConditionsRefineSharedGraph(t *testing.T) {
	for _, tc := range []struct {
		name, call, helpers string
		reachable           bool
	}{
		{"reader", `noop()`, `func noop(){}`, false},
		{"transitive reader", `noop()`, `func noop(){helper()};func helper(){}`, false},
		{"writer", `noop(&flag)`, `func noop(p *bool){*p=true}`, true},
		{"unknown", `noop()`, `var noop func()`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;func f(){flag:=false;_=&flag;`+tc.call+`;if flag{println("arm")}};`+tc.helpers)
			graph := continuationGraph(t, extraction, "example.f")
			found, reachable := false, false
			for _, block := range graph.ValueFlow.Blocks {
				for _, ref := range block.Operations {
					attrs := graph.NodeAttributes(graph.Nodes[ref-1])
					if graph.Tables.Text(attrs.Callee.Value) != "println" {
						continue
					}
					found = true
					reachable = reachable || block.Reachable
					if !block.Reachable {
						assertNoResultExecutionEdges(t, graph, ref)
					}
				}
			}
			if !found {
				t.Fatal("missing source operation")
			}
			if reachable != tc.reachable {
				t.Fatalf("arm reachable=%v want %v", reachable, tc.reachable)
			}
		})
	}
}
