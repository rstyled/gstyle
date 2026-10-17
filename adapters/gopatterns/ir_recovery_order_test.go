package gopatterns

import (
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"runtime"
	"slices"
	"testing"
)

func TestRecoveryExitContextsAndOperandOrders(t *testing.T) {
	for _, tc := range []struct {
		name, body          string
		ordinary, inherited callflow.Outcomes
	}{
		{"direct", `recover()`, callflow.Returns, callflow.RecoversInherited},
		{"twice", `recover();recover()`, callflow.Returns, callflow.RecoversInherited},
		{"nested ordinary", `absorb()`, callflow.Returns, callflow.Returns},
		{"builtin defer is not a recovery body", `defer recover();panic("fault")`, callflow.Panics, callflow.Panics},
		{"recover own panic", `defer absorb();panic("fault")`, callflow.Returns, callflow.Returns},
		{"panic after recovery", `recover();panic("fault")`, callflow.Panics, callflow.Panics},
		{"goexit after recovery", `recover();runtime.Goexit()`, callflow.ExitsGoroutine, callflow.ExitsGoroutine},
		{"outer guard is not reevaluated after a store", `defer absorb();value:=recover();if value!=nil{value=nil;sink(abortInt(),values[index])}`, callflow.Returns, callflow.RecoversInherited | callflow.Aborts},
		{"logical join leaves sibling unordered", `defer absorb();sink(recover()!=nil&&choose,values[index]);os.Exit(1)`, callflow.Returns | callflow.Aborts, callflow.Returns | callflow.RecoversInherited | callflow.Aborts},
		{"parallel RHS sibling", `defer absorb();_,_=values[index],recover();os.Exit(1)`, callflow.Returns | callflow.Aborts, callflow.Returns | callflow.RecoversInherited | callflow.Aborts},
		{"sibling before selected recovery", `defer absorb();sink(values[index],recover());os.Exit(1)`, callflow.Returns | callflow.Aborts, callflow.Returns | callflow.RecoversInherited | callflow.Aborts},
		{"sibling after selected recovery", `defer absorb();sink(recover(),values[index]);os.Exit(1)`, callflow.Returns | callflow.Aborts, callflow.Returns | callflow.RecoversInherited | callflow.Aborts},
		{"required by earlier invocation", `defer absorb();sink(identity(values[index]),recover());os.Exit(1)`, callflow.Returns | callflow.Aborts, callflow.Returns | callflow.Aborts},
		{"earlier statement fault", `defer absorb();_=values[index];recover();os.Exit(1)`, callflow.Returns | callflow.Aborts, callflow.Returns | callflow.Aborts},
		{"dependent fault only after recovery", `defer absorb();_=recover().(int);os.Exit(1)`, callflow.Returns | callflow.Aborts, callflow.RecoversInherited | callflow.Aborts},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex, _ := proofExtraction(t, `package example;import("os";"runtime");var _=os.Exit;var _=runtime.Goexit;var values []int;var index int;var choose bool;func f(){`+tc.body+`};func absorb(){recover()};func sink(any,any){};func identity(x int)int{return x};func abortInt()int{os.Exit(1);return 0}`)
			graphs := map[string]*pdg.Graph{}
			for i := range ex.Funcs {
				g := &ex.Funcs[i].Pdg
				graphs[g.Tables.Text(g.Function.ID)] = g
			}
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			got := summaries["example.f"]
			if got.Ordinary != tc.ordinary || got.DeferredPanic != tc.inherited {
				t.Fatalf("ordinary=%v inherited=%v want %v/%v", got.Ordinary, got.DeferredPanic, tc.ordinary, tc.inherited)
			}
		})
	}
}

func nativeRecoveryCapture(value *any)  { *value = recover() }
func nativeIndirectRecovery(value *any) { nativeRecoveryCapture(value) }
func nativeRecoveryAbsorb()             { recover() }
func nativeFaultBeforeRecovery() {
	defer nativeRecoveryAbsorb()
	var values []int
	_ = values[0]
	recover()
}
func nativeFaultAfterRecovery() {
	defer nativeRecoveryAbsorb()
	recover()
	var values []int
	_ = values[0]
}
func TestNativeDirectRecoveryEligibilityAndOrder(t *testing.T) {
	for _, direct := range []bool{false, true} {
		var inner, outer any
		func() {
			defer nativeRecoveryCapture(&outer)
			func() {
				if direct {
					defer nativeRecoveryCapture(&inner)
				} else {
					defer nativeIndirectRecovery(&inner)
				}
				panic("initial")
			}()
		}()
		if direct && (inner != "initial" || outer != nil) {
			t.Fatalf("direct inner=%v outer=%v", inner, outer)
		}
		if !direct && (inner != nil || outer != "initial") {
			t.Fatalf("indirect inner=%v outer=%v", inner, outer)
		}
	}
	var builtin any
	func() { defer nativeRecoveryCapture(&builtin); func() { defer recover(); panic("initial") }() }()
	if builtin != "initial" {
		t.Fatalf("deferred builtin consumed panic: %v", builtin)
	}
	for _, before := range []bool{false, true} {
		var outer any
		func() {
			defer nativeRecoveryCapture(&outer)
			func() {
				if before {
					defer nativeFaultBeforeRecovery()
				} else {
					defer nativeFaultAfterRecovery()
				}
				panic("initial")
			}()
		}()
		if (outer == "initial") != before {
			t.Fatalf("before=%v outer=%v", before, outer)
		}
	}
}
func TestNativeRecoveryKeepsGoexitBarrier(t *testing.T) {
	done := make(chan struct{})
	returned := false
	var outer any
	go func() {
		defer close(done)
		defer nativeRecoveryCapture(&outer)
		func() { defer nativeRecoveryAbsorb(); defer panic("new"); runtime.Goexit() }()
		returned = true
	}()
	<-done
	if returned || outer != nil {
		t.Fatalf("Goexit resumed ordinary flow: returned=%v panic=%v", returned, outer)
	}
}

func TestRecoveryRequiresCanonicalInvocationOrder(t *testing.T) {
	graph := parseIR(t, `package p;func f(){recover()}`, pdg.NewBuilder())
	name := graph.Tables.Text(graph.Function.ID)
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if node.Category != pdg.Call {
			continue
		}
		set := graph.Attributes[node.Attributes-1]
		graph.AttributeTables.Extensions[set.Extensions-1] = slices.DeleteFunc(slices.Clone(attrs.Extensions), func(e pdg.Extension) bool { return graph.Tables.Text(e.Name) == "gstyle.go:invocation-order" })
	}
	summaries, err := callflow.ExitEffects(map[string]*pdg.Graph{name: &graph})
	if err != nil {
		t.Fatal(err)
	}
	if summaries[name].DeferredPanic&callflow.UnknownExit == 0 {
		t.Fatal("missing call order replaced by source traversal")
	}
}

func TestDeferredInvocationDoesNotEstablishBodyOrder(t *testing.T) {
	graph := parseIR(t, `package p;func f(){defer noop()};func noop(){}`, pdg.NewBuilder())
	sites, err := deferflow.Sites(&graph)
	if err != nil || len(sites) != 1 {
		t.Fatalf("sites=%v error=%v", sites, err)
	}
	operations, err := callflow.NewOperationEffects(&graph)
	if err != nil {
		t.Fatal(err)
	}
	if _, known := operations.PriorOperations(sites[0].Invocation); known {
		t.Fatal("deferred invocation establishes body prefix")
	}
	if _, known := operations.LaterOperations(sites[0].Invocation); known {
		t.Fatal("deferred invocation establishes body suffix")
	}
}
