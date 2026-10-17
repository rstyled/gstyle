package gopatterns

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"slices"
	"testing"
)

func TestResourceReleaseFollowsImplicitOperationPanics(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"bounds before close", `_=xs[index];resource.Close()`, false},
		{"bounds after saved close", `defer resource.Close();_=xs[index]`, true},
		{"pointer before close", `_=*pointer;resource.Close()`, false},
		{"pointer after saved close", `defer resource.Close();_=*pointer`, true},
		{"division before close", `_=7/divisor;resource.Close()`, false},
		{"division after saved close", `defer resource.Close();_=7/divisor`, true},
		{"interface comparison before close", `_=value==value;resource.Close()`, false},
		{"interface comparison after saved close", `defer resource.Close();_=value==value`, true},
		{"assertion before close", `_=value.(int);resource.Close()`, false},
		{"assertion after saved close", `defer resource.Close();_=value.(int)`, true},
		{"deferred input before saved close", `defer println(xs[index]);defer resource.Close()`, false},
		{"deferred input after saved close", `defer resource.Close();defer println(xs[index])`, true},
		{"fault before transfer", `_=xs[index];return resource`, false},
		{"fault after release", `resource.Close();_=xs[index]`, true},
		{"short circuit can fault", `if flag && xs[index]>0 {};resource.Close()`, false},
		{"unreachable fault", `if false {_=xs[index]};resource.Close()`, true},
		{"safe scalar operation", `_=divisor+1;resource.Close()`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tail := `;return nil`
			if tc.name == "fault before transfer" {
				tail = ""
			}
			source := `package p;func f(resource *T,xs []int,index,divisor int,pointer *int,value any,flag bool)*T{` + tc.body + tail + `};type T struct{};func(*T)Close(){}`
			graph := parseIR(t, source, pdg.NewBuilder())
			query, err := resourceflow.New(&graph)
			if err != nil {
				t.Fatal(err)
			}
			policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol, TransferReturn: true}
			if got := query.FromEntry(policy); got != tc.want {
				t.Fatalf("release=%v want %v", got, tc.want)
			}
		})
	}
}

func TestOperationEffectsAreCanonicalEvidence(t *testing.T) {
	graph := parseIR(t, `package p;func f(resource *T,xs []int,index int){defer resource.Close();_=xs[index]};type T struct{};func(*T)Close(){}`, pdg.NewBuilder())
	policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}
	prove := func() bool {
		query, err := resourceflow.New(&graph)
		if err != nil {
			t.Fatal(err)
		}
		return query.FromEntry(policy)
	}
	if !prove() {
		t.Fatal("baseline fault does not reach saved close")
	}
	var changed bool
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if attrs.Operator.Status != pdg.Known || graph.Tables.Text(attrs.Operator.Value) != "index" {
			continue
		}
		set := graph.Attributes[node.Attributes-1]
		extensions := graph.AttributeTables.Extensions[set.Extensions-1]
		graph.AttributeTables.Extensions[set.Extensions-1] = slices.DeleteFunc(slices.Clone(extensions), func(extension pdg.Extension) bool {
			return graph.Tables.Text(extension.Name) == "gstyle.go:operation-effects"
		})
		changed = true
	}
	if !changed {
		t.Fatal("missing index operation")
	}
	if prove() {
		t.Fatal("source shape replaces missing operation evidence")
	}
}

func TestOperationEffectReaderDoesNotChangeGraph(t *testing.T) {
	graph := parseIR(t, `package p;func f(xs []int,index int)int{return xs[index]}`, pdg.NewBuilder())
	before, _ := json.Marshal(&graph)
	effects, err := callflow.NewOperationEffects(&graph)
	if err != nil {
		t.Fatal(err)
	}
	var fault bool
	for i := range graph.Nodes {
		if effects.Outcomes(pdg.Ref(i+1))&callflow.Panics != 0 {
			fault = true
		}
	}
	if !fault {
		t.Fatal("index operation has no panic effect")
	}
	after, _ := json.Marshal(&graph)
	if string(before) != string(after) {
		t.Fatal("effect query mutates graph")
	}
	if effects.Outcomes(0) != callflow.UnknownExit || effects.Outcomes(pdg.Ref(len(graph.Nodes)+1)) != callflow.UnknownExit {
		t.Fatal("invalid operation reference establishes completion")
	}
}

func nativeOperationRelease(action func(), saved bool) (closed, panicked bool) {
	func() {
		defer func() { panicked = recover() != nil }()
		if saved {
			defer func() { closed = true }()
		}
		action()
		if !saved {
			closed = true
		}
	}()
	return closed, panicked
}
func TestImplicitFaultReleaseMatchesNativeGo(t *testing.T) {
	var xs []int
	var pointer *int
	divisor := 0
	var value any = "wrong type"
	var incomparable any = []int{1}
	for _, tc := range []struct {
		name, source string
		action       func()
	}{
		{"bounds", `_=xs[index]`, func() { _ = xs[1] }},
		{"pointer", `_=*pointer`, func() { _ = *pointer }},
		{"division", `_=7/divisor`, func() { _ = 7 / divisor }},
		{"assertion", `_=value.(int)`, func() { _ = value.(int) }},
		{"interface comparison", `_=value==value`, func() { _ = incomparable == incomparable }},
	} {
		for _, saved := range []bool{false, true} {
			t.Run(tc.name+fmt.Sprint(saved), func(t *testing.T) {
				closed, panicked := nativeOperationRelease(tc.action, saved)
				if !panicked {
					t.Fatal("native counterexample does not panic")
				}
				body := tc.source + `;resource.Close()`
				if saved {
					body = `defer resource.Close();` + tc.source
				}
				graph := parseIR(t, `package p;func f(resource *T,xs []int,index,divisor int,pointer *int,value any){`+body+`};type T struct{};func(*T)Close(){}`, pdg.NewBuilder())
				query, err := resourceflow.New(&graph)
				if err != nil {
					t.Fatal(err)
				}
				if got := query.FromEntry(resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}); got != closed {
					t.Fatalf("PDG release=%v native closed=%v", got, closed)
				}
			})
		}
	}
}
func TestOwnedResourceNilRefinementDoesNotHideNestedFaults(t *testing.T) {
	for _, tc := range []struct {
		declaration, operation string
		want                   bool
	}{
		{`type T struct{Number int}`, `_=resource.Number`, true},
		{`type T struct{Nested *T;Number int}`, `_=resource.Nested.Number`, false},
		{`type T struct{*U};type U struct{Number int}`, `_=resource.Number`, false},
	} {
		graph := parseIR(t, `package p;func f(resource *T){`+tc.operation+`;resource.Close()};`+tc.declaration+`;func(*T)Close(){}`, pdg.NewBuilder())
		query, err := resourceflow.New(&graph)
		if err != nil {
			t.Fatal(err)
		}
		if got := query.FromEntry(resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}); got != tc.want {
			t.Fatalf("%s: release=%v want %v", tc.declaration, got, tc.want)
		}
	}
}

func TestUnspecifiedOperandOrderCannotProveEarlierRelease(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       bool
	}{
		{"close written first", `consume(resource.Close(),xs[index])`, false},
		{"fault written first", `consumeFirst(xs[index],resource.Close())`, false},
		{"saved close covers either order", `defer resource.Close();consume(resource.Close(),xs[index])`, true},
		{"later statement cannot preempt", `resource.Close();_=xs[index]`, true},
		{"dependent fault cannot preempt", `_=after(resource.Close())[index]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			graph := parseIR(t, `package p;func f(resource *T,xs []int,index int){`+tc.body+`};type T struct{};func(*T)Close()error{return nil};func consume(error,int){};func consumeFirst(int,error){};func after(error)[]int{return nil}`, pdg.NewBuilder())
			query, err := resourceflow.New(&graph)
			if err != nil {
				t.Fatal(err)
			}
			if got := query.FromEntry(resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}); got != tc.want {
				t.Fatalf("release=%v want %v", got, tc.want)
			}
		})
	}
}

func TestMissingSiblingRegionCannotProveCallOrder(t *testing.T) {
	graph := parseIR(t, `package p;func f(resource *T,xs []int,index int){consume(resource.Close(),xs[index])};type T struct{};func(*T)Close()error{return nil};func consume(error,int){}`, pdg.NewBuilder())
	changed := false
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if graph.Tables.Text(attrs.Operator.Value) != "index" {
			continue
		}
		set := graph.Attributes[node.Attributes-1]
		graph.AttributeTables.Extensions[set.Extensions-1] = slices.DeleteFunc(slices.Clone(attrs.Extensions), func(extension pdg.Extension) bool {
			return graph.Tables.Text(extension.Name) == "gstyle.go:evaluation-region"
		})
		changed = true
	}
	if !changed {
		t.Fatal("missing index operation")
	}
	query, err := resourceflow.New(&graph)
	if err != nil {
		t.Fatal(err)
	}
	if query.FromEntry(resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}) {
		t.Fatal("missing sibling region invents must-before order")
	}
}
func TestLocalExitSummaryRetainsUnsequencedPanic(t *testing.T) {
	extraction, _ := proofExtraction(t, `package example
 import "runtime"
 func local(xs []int,index int){consume(stops(),xs[index])}
 func stops()int{runtime.Goexit();return 0}
 func consume(int,int){}
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
	if got := summaries["example.local"].Ordinary; got != callflow.Panics|callflow.ExitsGoroutine {
		t.Fatalf("unsequenced sibling fault erased: %v", got)
	}
}

func TestConditionalNilFaultRequiresCanonicalInput(t *testing.T) {
	graph := parseIR(t, `package p;func f(){_=7}`, pdg.NewBuilder())
	changed := false
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if attrs.Literal.Status != pdg.Known {
			continue
		}
		set := graph.Attributes[node.Attributes-1]
		extensions := slices.Clone(attrs.Extensions)
		for i, extension := range extensions {
			if graph.Tables.Text(extension.Name) != "gstyle.go:operation-effects" {
				continue
			}
			graph.Tables.Strings = append(graph.Tables.Strings, `{"nil_input":true}`)
			extensions[i].Value = pdg.Text(len(graph.Tables.Strings))
			changed = true
		}
		if set.Extensions != 0 {
			graph.AttributeTables.Extensions[set.Extensions-1] = extensions
		}
	}
	if !changed {
		t.Fatal("missing literal effect to replace")
	}
	if _, err := callflow.NewOperationEffects(&graph); err == nil {
		t.Fatal("nil fault with no canonical input accepted")
	}
}

func TestEvaluationRegionRequiresAdmittedExecutionContext(t *testing.T) {
	graph := parseIR(t, `package p;func f(){go work()};func work(){}`, pdg.NewBuilder())
	if pdg.ExitBodyFlow(&graph) != nil {
		t.Fatal("concurrent body unexpectedly has admitted sequential flow")
	}
	operations, err := callflow.NewOperationEffects(&graph)
	if err != nil {
		t.Fatal(err)
	}
	for i, node := range graph.Nodes {
		if node.Category != pdg.Call {
			continue
		}
		if _, known := operations.PriorOperations(pdg.Ref(i + 1)); known {
			t.Fatal("region tag alone establishes execution context")
		}
	}
}
