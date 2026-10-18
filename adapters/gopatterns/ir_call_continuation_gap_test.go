package gopatterns

import (
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCanonicalCallContinuations(t *testing.T) {
	for _, tc := range []struct {
		name, body, declarations string
		reachable                bool
	}{
		{"panic", `stop();println("after")`, `func stop(){panic("stop")}`, false},
		{"mixed nonreturning exits", `stop(choose);println("after")`, `func stop(flag bool){if flag{panic("stop")};for{}}`, false},
		{"divergence", `stop();println("after")`, `func stop(){for{}}`, false},
		{"recursion", `stop();println("after")`, `func stop(){stop()}`, false},
		{"goexit", `stop();println("after")`, `func stop(){runtime.Goexit()}`, false},
		{"abort", `stop();println("after")`, `func stop(){os.Exit(1)}`, false},
		{"returns", `stop();println("after")`, `func stop(){}`, true},
		{"unknown", `target();println("after")`, `var target func()`, true},
		{"may return", `stop(choose);println("after")`, `func stop(flag bool){if flag{panic("stop")}}`, true},
		{"unknown prefix still reaches nonreturning target", `foreign();stop();println("after")`, `var foreign func();func stop(){panic("stop")}`, false},
		{"unknown prefix may return on other arm", `flag:=true;if choose{flag=false;foreign()};stop(flag);println("after")`, `var foreign func();func stop(flag bool){if flag{panic("stop")}}`, true},
		{"assigned argument", `flag:=true;stop(flag);println("after")`, `func stop(flag bool){if flag{panic("stop")}}`, false},
		{"overwritten argument", `flag:=false;flag=true;stop(flag);println("after")`, `func stop(flag bool){if flag{panic("stop")}}`, false},
		{"both branch assignments stop", `flag:=false;if choose{flag=true}else{flag=true};stop(flag);println("after")`, `func stop(flag bool){if flag{panic("stop")}}`, false},
		{"one branch returns", `flag:=false;if choose{flag=true};stop(flag);println("after")`, `func stop(flag bool){if flag{panic("stop")}}`, true},
		{"indirect overwrite", `flag:=true;pointer:=&flag;*pointer=false;stop(flag);println("after")`, `func stop(flag bool){if flag{panic("stop")}}`, true},
		{"unknown call write", `flag:=true;change(&flag);stop(flag);println("after")`, `func change(pointer *bool){*pointer=false};func stop(flag bool){if flag{panic("stop")}}`, true},
		{"loop may return", `flag:=false;for choose{flag=true};stop(flag);println("after")`, `func stop(flag bool){if flag{panic("stop")}}`, true},
		{"known argument", `stop(true);println("after")`, `func stop(flag bool){if flag{panic("stop")}}`, false},
		{"saved function", `target:=stop;target();println("after")`, `func stop(){panic("stop")}`, false},
		{"nested call", `_ = stop()+1;println("after")`, `func stop()int{panic("stop")}`, false},
		{"defer registration", `defer stop();println("after")`, `func stop(){panic("stop")}`, true},
		{"callee recovery", `stop();println("after")`, `func stop(){defer func(){recover()}();panic("stop")}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extraction, _ := proofExtraction(t, `package example;import("runtime";"os");var _=runtime.Goexit;var _=os.Exit;var choose bool;func f(){`+tc.body+`};`+tc.declarations)
			graph := continuationGraph(t, extraction, "example.f")
			if err := pdg.Validate(graph); err != nil {
				t.Fatal(err)
			}
			found, reachable := false, false
			for _, block := range graph.ValueFlow.Blocks {
				for _, ref := range block.Operations {
					attrs := graph.NodeAttributes(graph.Nodes[ref-1])
					if graph.Tables.Text(attrs.Callee.Value) != "println" {
						continue
					}
					found = true
					reachable = reachable || block.Reachable
				}
			}
			if !found {
				t.Fatal("missing println source operation")
			}
			if !reachable {
				for _, edge := range graph.Edges {
					attrs := graph.NodeAttributes(graph.Nodes[edge.Target-1])
					if (edge.Kind == pdg.Execution || edge.Kind == pdg.ControlEdge) && graph.Tables.Text(attrs.Callee.Value) == "println" {
						t.Fatal("unreachable operation retains execution/control dependency")
					}
				}
			}
			if reachable != tc.reachable {
				t.Fatalf("after call reachable=%v want %v", reachable, tc.reachable)
			}
		})
	}
}

func continuationGraph(t *testing.T, extraction *Extraction, name string) *pdg.Graph {
	t.Helper()
	for i := range extraction.Funcs {
		graph := &extraction.Funcs[i].Pdg
		if graph.Tables.Text(graph.Function.ID) == name {
			return graph
		}
	}
	t.Fatalf("missing graph %s", name)
	return nil
}

func TestCallContinuationRetainsExitSummary(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func stop(){panic("stop")};func f(){stop();println("after")}`)
	graphs := map[string]*pdg.Graph{}
	for i := range extraction.Funcs {
		graph := &extraction.Funcs[i].Pdg
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	exits, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"example.stop", "example.f"} {
		if exits[name].Ordinary != callflow.Panics {
			t.Errorf("%s: %+v", name, exits[name])
		}
	}
}

func TestCallContinuationRecomputesDependencies(t *testing.T) {
	source := `package example;func stop(){panic("stop")};func f(choose bool)int{x:=0;if choose{x=1;stop()};return x}`
	extraction, _ := proofExtraction(t, source)
	graph := continuationGraph(t, extraction, "example.f")
	var written, read, returned pdg.Ref
	for i, node := range graph.Nodes {
		if node.Span == 0 {
			continue
		}
		span := graph.Spans[node.Span-1]
		text := source[span.Start:span.End]
		ref := pdg.Ref(i + 1)
		if text == "x=1" {
			written = ref
		}
		if text == "return x" {
			returned = ref
		}
		if text == "x" && int(span.Start) == strings.LastIndex(source, "return x")+len("return ") {
			read = ref
		}
	}
	if written == 0 || read == 0 || returned == 0 {
		t.Fatal("missing source operations")
	}
	seen := map[pdg.Ref]bool{written: true}
	queue := []pdg.Ref{written}
	for i := 0; i < len(queue); i++ {
		for _, edge := range graph.Edges {
			if edge.Kind != pdg.Data || edge.Source != queue[i] || seen[edge.Target] {
				continue
			}
			seen[edge.Target] = true
			queue = append(queue, edge.Target)
		}
	}
	if seen[read] {
		t.Fatal("definition from nonreturning arm reaches the return read")
	}
	controlled := false
	for _, edge := range graph.Edges {
		if edge.Kind == pdg.ControlEdge && edge.Target == returned && graph.Tables.Text(edge.Outcome) == "false" {
			controlled = true
		}
	}
	if !controlled {
		t.Fatal("return loses dependence on the false branch")
	}
}

func TestCallContinuationPreservesDivergenceContext(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func stop(){for{}};func f(){stop();println("after")}`)
	graph := continuationGraph(t, extraction, "example.f")
	cyclic := false
	for i, block := range graph.ValueFlow.Blocks {
		for _, next := range block.Successors {
			if next == pdg.Ref(i+1) {
				cyclic = true
			}
		}
	}
	if !cyclic {
		t.Fatal("proven divergence becomes a finite exit")
	}
	if pdg.ExitBodyFlow(graph) == graph.ValueFlow {
		t.Fatal("exit solver loses its original body context")
	}
	graph.ValueFlow.Blocks[0].Reachable = false
	if err := pdg.Validate(graph); err == nil {
		t.Fatal("invalid refined reachability is accepted beside valid exit context")
	}
}

func TestCallContinuationUnsupportedBodyStaysPartial(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example;func stop(){panic("stop")};func f(){go stop();stop()}`)
	graph := continuationGraph(t, extraction, "example.f")
	if pdg.BodyFlow(graph) != nil {
		t.Fatal("concurrent body gains unsupported sequential-flow claim")
	}
}
