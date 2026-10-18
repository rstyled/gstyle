package gopatterns

import (
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func closureNodes(graph *pdg.Graph) []pdg.Ref {
	var refs []pdg.Ref
	for index, node := range graph.Nodes {
		if graph.Tables.Text(graph.NodeAttributes(node).Construct) == "closure" {
			refs = append(refs, pdg.Ref(index+1))
		}
	}
	return refs
}
func TestClosureConstructionDoesNotExecuteBody(t *testing.T) {
	source := `package p;func f()int{x:=1;_=func(){x=9;panic("stop")};return x}`
	graph := parseIR(t, source, pdg.NewBuilder())
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	closures := closureNodes(&graph)
	if len(closures) != 1 {
		t.Fatalf("closure values=%v", closures)
	}
	sites := 0
	for _, block := range graph.ValueFlow.Blocks {
		for _, ref := range block.Operations {
			if ref == closures[0] && block.Reachable {
				sites++
			}
		}
	}
	if sites != 1 {
		t.Fatalf("construction sites=%d, want one", sites)
	}
	got := generalReturnProducers(t, &graph, source)
	if !slices.Equal(got, []string{"x:=1"}) {
		t.Fatalf("body writes escape construction: %v", got)
	}
	for _, node := range graph.Nodes {
		if node.Category == pdg.Call {
			t.Fatal("unexecuted panic becomes an outer call")
		}
	}
	for _, definition := range graph.Definitions {
		if definition.Node == closures[0] {
			t.Fatal("constructing a closure writes its captured variable")
		}
	}
	capture := false
	for _, edge := range graph.Edges {
		if edge.Target == closures[0] && graph.Tables.Text(edge.Subkind) == "capture-address" {
			capture = edge.Location != 0 && edge.Symbol != 0
		}
	}
	if !capture {
		t.Fatal("closure lacks a captured-cell address dependency")
	}
}
func TestClosureCallHasWeakCapturedWritesAndTargetDependency(t *testing.T) {
	source := `package p;func f()int{x:=1;cb:=func(value int){x=value};cb(7);return x}`
	graph := parseIR(t, source, pdg.NewBuilder())
	got := generalReturnProducers(t, &graph, source)
	for _, want := range []string{"x:=1", "cb(7)"} {
		if !slices.Contains(got, want) {
			t.Fatalf("missing possible producer %q: %v", want, got)
		}
	}
	target, argument := false, false
	for _, edge := range graph.Edges {
		if graph.Nodes[edge.Target-1].Category != pdg.Call {
			continue
		}
		switch graph.Tables.Text(edge.Subkind) {
		case "call-target":
			target = edge.Position == 0 && graph.Nodes[edge.Source-1].Category == pdg.Read
		case "source-value":
			if edge.Position == 1 {
				argument = true
			}
		}
	}
	if !target || !argument {
		t.Fatalf("callee/argument dependencies: target=%v argument=%v", target, argument)
	}
}
func TestClosureCaptureScopeAndNestedConstruction(t *testing.T) {
	cases := []struct {
		name, source string
		captures     int
	}{
		{"shadowed parameter", `func f(x int){_=func(x int){_=x}}`, 0},
		{"inner local", `func f(){_=func(){x:=1;_=x}}`, 0},
		{"transitive capture", `func f(x int){_=func(){_=func(){_=x}}}`, 1},
		{"body-only defer", `func f(x int){_=func(){defer func(){_=x}()}}`, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			graph := parseIR(t, "package p;"+tc.source, pdg.NewBuilder())
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			closures := closureNodes(&graph)
			if len(closures) != 1 {
				t.Fatalf("outer closure count=%d", len(closures))
			}
			attrs := graph.NodeAttributes(graph.Nodes[closures[0]-1])
			if len(attrs.Symbols) != tc.captures {
				t.Fatalf("captures=%v want %d", attrs.Symbols, tc.captures)
			}
			if graph.ValueFlow == nil {
				t.Fatal("unexecuted body prevents shared outer flow")
			}
		})
	}
}
func TestClosureGlobalReferencesAreStorageNotLocalCaptures(t *testing.T) {
	source := `package p;var G int;func f()func(){return func(){G++}}`
	graph := parseIR(t, source, pdg.NewBuilder())
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	attrs := graph.NodeAttributes(graph.Nodes[closureNodes(&graph)[0]-1])
	if len(attrs.Symbols) != 0 {
		t.Fatal("package variable becomes an environment capture")
	}
	found := false
	for _, symbol := range graph.Symbols {
		if graph.Tables.Text(symbol.Name.Value) == "G" {
			found = symbol.Role == pdg.GlobalRole
		}
	}
	if !found || len(graph.Locations) == 0 {
		t.Fatal("body-only global has no conservative storage")
	}
}
func TestRepeatedClosureCapturesHaveCanonicalFacts(t *testing.T) {
	graph := parseIR(t, `package p;func f(x int){_=func(){_=x};_=func(){_=x}}`, pdg.NewBuilder())
	if err := pdg.Validate(&graph); err != nil {
		t.Fatal(err)
	}
	markers := 0
	for _, symbol := range graph.Symbols {
		for _, extension := range symbol.Extensions {
			if strings.HasSuffix(graph.Tables.Text(extension.Name), ":captured") {
				markers++
			}
		}
	}
	if markers != 1 {
		t.Fatalf("duplicate capture markers: %d", markers)
	}
}

func TestClosureCallTargetsRetainEvaluatedValues(t *testing.T) {
	cases := []struct{ name, body string }{
		{"immediate", `func(){}()`},
		{"parenthesized", `(func(){})()`},
		{"alias", `cb:=func(){};alias:=cb;alias()`},
		{"global", `Callback()`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			graph := parseIR(t, `package p;var Callback func();func f(){`+tc.body+`}`, pdg.NewBuilder())
			if err := pdg.Validate(&graph); err != nil {
				t.Fatal(err)
			}
			calls, targets := 0, 0
			for _, node := range graph.Nodes {
				if node.Category == pdg.Call {
					calls++
				}
			}
			for _, edge := range graph.Edges {
				if graph.Tables.Text(edge.Subkind) != "call-target" {
					continue
				}
				if edge.Kind != pdg.Data || edge.Position != 0 {
					t.Fatal("callee occupies an argument slot")
				}
				targets++
			}
			if calls != 1 || targets != calls {
				t.Fatalf("calls=%d target dependencies=%d", calls, targets)
			}
		})
	}
}
