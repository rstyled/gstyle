package pdg

import (
	"reflect"
	"testing"
)

func operationFlowFixture() *Graph {
	pool := NewBuilder()
	return &Graph{Tables: pool.Tables, Nodes: []Node{{Category: FormalInput}, {Category: Call}, {Category: Call}, {Category: MemorySummary}},
		Edges:      []Edge{{Source: 4, Target: 2, Kind: Data}},
		Extensions: pool.SingletonExtension("gstyle.pdg:operation-flow", `"0.1.0"`),
		ValueFlow:  &NormalFlow{Entry: 1, Blocks: []FlowBlock{{Operations: []Ref{2}, Successors: []Ref{1}, Reachable: true}, {Operations: []Ref{3}}}},
	}
}

func TestSharedProjectionRetainsMemoryAndExcludesUnreachableOperations(t *testing.T) {
	graph := operationFlowFixture()
	if err := validateOperationFlow(graph); err != nil {
		t.Fatal(err)
	}
	want := []bool{true, true, false, true}
	if got := ReachableProjection(graph); !reflect.DeepEqual(got, want) {
		t.Fatalf("projection=%v, want %v", got, want)
	}
	if graph.NormalFlow != nil {
		t.Fatal("shared projection acquired strict normal admission")
	}
	graph.Extensions = nil
	if ReachableProjection(graph) != nil {
		t.Fatal("partial legacy flow removes operations")
	}
}

func TestSharedFlowRejectsInvalidReferencesAndClaims(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Graph)
	}{
		{"entry", func(g *Graph) { g.ValueFlow.Entry = 3 }},
		{"operation", func(g *Graph) { g.ValueFlow.Blocks[0].Operations = []Ref{5} }},
		{"successor", func(g *Graph) { g.ValueFlow.Blocks[0].Successors = []Ref{3} }},
		{"condition", func(g *Graph) { g.ValueFlow.Blocks[0].Condition = 5 }},
		{"reachability", func(g *Graph) { g.ValueFlow.Blocks[0].Reachable = false }},
		{"duplicate", func(g *Graph) { g.ValueFlow.Blocks[1].Operations = []Ref{2} }},
		{"value entity", func(g *Graph) { g.ValueFlow.Blocks[0].Operations = []Ref{4} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			graph := operationFlowFixture()
			tc.change(graph)
			if err := validateOperationFlow(graph); err == nil {
				t.Fatal("invalid operation flow accepted")
			}
		})
	}
}

func TestRegistrationFlowHasSeparateAdmission(t *testing.T) {
	graph := operationFlowFixture()
	pool := NewBuilder()
	graph.Tables = pool.Tables
	graph.Extensions = pool.SingletonExtension("gstyle.pdg:registration-flow", `"0.1.0"`)
	if BodyFlow(graph) != graph.ValueFlow {
		t.Fatal("registration body flow missing")
	}
	if ReachableProjection(graph) != nil {
		t.Fatal("registration flow grants complete ordinary reachability")
	}
	if err := validateOperationFlow(graph); err != nil {
		t.Fatal(err)
	}
	graph.ValueFlow.Blocks[0].Successors = []Ref{3}
	if err := validateOperationFlow(graph); err == nil {
		t.Fatal("registration context bypasses structural validation")
	}
}

func TestExitBodyFlowRejectsInvalidReferences(t *testing.T) {
	graph := operationFlowFixture()
	pool := NewBuilder()
	graph.Tables = pool.Tables
	graph.Extensions = pool.SingletonExtension("gstyle.pdg:exit-body-flow", `"0.1.0"`)
	if ExitBodyFlow(graph) != graph.ValueFlow {
		t.Fatal("exit body missing")
	}
	if BodyFlow(graph) != nil || ReachableProjection(graph) != nil {
		t.Fatal("exit body grants ordinary flow admission")
	}
	if err := validateOperationFlow(graph); err != nil {
		t.Fatal(err)
	}
	graph.ValueFlow.Blocks[0].Successors = []Ref{3}
	if err := validateOperationFlow(graph); err == nil {
		t.Fatal("exit body bypasses structural validation")
	}
}
