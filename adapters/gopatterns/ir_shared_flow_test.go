package gopatterns

import (
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

func TestRuntimeGuardConsumesSemanticSwitchPredicates(t *testing.T) {
	for _, body := range []string{
		`switch p { case nil: return 0 }; return *p`,
		`switch { case p==nil: return 0 }; return *p`,
		`switch p { case nil: return 0; default: return *p }`,
	} {
		graphs := runtimeTestGraphs(t, "package p;func f(p *int)int{"+body+"}")
		graph := &graphs[0].Pdg
		if err := pdg.Validate(graph); err != nil {
			t.Fatal(err)
		}
		report, err := runtimeguard.Analyze(graph)
		if err != nil {
			t.Fatal(err)
		}
		for _, site := range report.Accesses {
			if !site.Value.NonNil {
				t.Fatalf("safe switch path loses pointer guard: %s: %+v", body, site)
			}
		}
		predicates := map[pdg.Ref]bool{}
		for _, block := range graph.ValueFlow.Blocks {
			if block.Condition != 0 {
				predicates[block.Condition] = true
			}
		}
		found := false
		for _, edge := range graph.Edges {
			if edge.Kind != pdg.ControlEdge || graph.Tables.Text(graph.NodeAttributes(graph.Nodes[edge.Source-1]).Construct) != "switch-case" {
				continue
			}
			if !predicates[edge.Source] {
				t.Fatal("value analysis does not use the PDG's switch predicate")
			}
			found = true
		}
		if !found {
			t.Fatal("switch has no semantic dependency")
		}
	}
}

func TestRangeBindingDoesNotWriteOnZeroIterations(t *testing.T) {
	src := `package p;func f(xs []int)int{i:=-1;for i=range xs{};return []int{1}[i]}`
	site := reviewAccess(t, src)
	if site.Value.HasLower && site.Value.Lower >= 0 {
		t.Fatalf("zero-iteration range overwrites initial negative index: %+v", site.Value)
	}
}

func TestRangeBindingWritesOnlyOnSelectedBodyEdge(t *testing.T) {
	graphs := runtimeTestGraphs(t, `package p;func f(xs []int)int{i:=-1;for i=range xs{_=[]int{1}[i]};return i}`)
	graph := &graphs[0].Pdg
	found := false
	for _, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if graph.Tables.Text(attrs.Construct) != "range" {
			continue
		}
		found = true
		if len(attrs.Writes) != 0 || len(attrs.Defines) != 0 {
			t.Fatal("range decision writes a binding before its outcome")
		}
	}
	if !found {
		t.Fatal("missing range decision")
	}
	report, err := runtimeguard.Analyze(graph)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Accesses) != 1 || !report.Accesses[0].Value.HasLower || report.Accesses[0].Value.Lower < 0 {
		t.Fatalf("body does not receive selected range binding: %+v", report)
	}
}

func TestRuntimeSwitchRetainsRealNilAccess(t *testing.T) {
	for _, body := range []string{
		`switch p {case nil:return *p;default:return 0}`,
		`switch {case p==nil:return *p;default:return 0}`,
	} {
		graphs := runtimeTestGraphs(t, "package p;func f(p *int)int{"+body+"}")
		graph := &graphs[0].Pdg
		if err := pdg.Validate(graph); err != nil {
			t.Fatal(err)
		}
		report, err := runtimeguard.Analyze(graph)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Accesses) != 1 || !report.Accesses[0].Value.Nil {
			t.Fatalf("switch loses the real nil access: %+v", report)
		}
	}
}

func TestRuntimeSharedFlowKeepsConstantOutcomeMeaning(t *testing.T) {
	for _, body := range []string{
		`if false{return *p};return 0`,
		`switch {case false:return *p;default:return 0}`,
		`for false {_=*p};return 0`,
	} {
		report := reviewReport(t, "package p;func f(p *int)int{"+body+"}")
		if len(report.Accesses) != 0 {
			t.Fatalf("constant false branch remains executable: %s: %+v", body, report)
		}
	}
}

func TestRuntimeSharedFlowKeepsReachableFalseOutcome(t *testing.T) {
	report := reviewReport(t, `package p;func f(p *int)int{if false{return 0};p=nil;return *p}`)
	if len(report.Accesses) != 1 || !report.Accesses[0].Value.Nil {
		t.Fatalf("false outcome suppresses a reachable nil access: %+v", report)
	}
}

func TestRangeAssignmentTargetEvaluatesOnIterationEdge(t *testing.T) {
	src := `package p;func f(xs []int,p *int){for *p=range xs{}}`
	graphs := runtimeTestGraphs(t, src)
	graph := &graphs[0].Pdg
	var decision, read pdg.Ref
	for index, node := range graph.Nodes {
		attrs := graph.NodeAttributes(node)
		if graph.Tables.Text(attrs.Construct) == "range" {
			decision = pdg.Ref(index + 1)
		}
		if node.Span != 0 {
			span := graph.Spans[node.Span-1]
			if src[span.Start:span.End] == "p" && node.Category == pdg.Read {
				read = pdg.Ref(index + 1)
			}
		}
	}
	if decision == 0 || read == 0 {
		t.Fatal("missing range decision or assignment-target read")
	}
	controlled := false
	for _, edge := range graph.Edges {
		if edge.Kind == pdg.ControlEdge && edge.Source == decision && edge.Target == read && edge.Position == 1 {
			controlled = true
		}
	}
	if !controlled {
		t.Fatal("range assignment target is evaluated before the iterator selects an element")
	}
	sites := 0
	for _, block := range graph.ValueFlow.Blocks {
		for _, operation := range block.Operations {
			if operation == read {
				sites++
			}
		}
	}
	if sites != 1 {
		t.Fatalf("range target has %d operation sites, want one iteration site", sites)
	}

	if err := pdg.Validate(graph); err != nil {
		t.Fatal(err)
	}
}

func TestRangeTargetDoesNotHavePreheaderExecutionConstraint(t *testing.T) {
	src := `package p;func f(xs []int,p *int){for *p=range xs{}}`
	graph := &runtimeTestGraphs(t, src)[0].Pdg
	var decision, target pdg.Ref
	for index, node := range graph.Nodes {
		if graph.Tables.Text(graph.NodeAttributes(node).Construct) == "range" {
			decision = pdg.Ref(index + 1)
		}
		if node.Span != 0 {
			span := graph.Spans[node.Span-1]
			if src[span.Start:span.End] == "*p" {
				target = pdg.Ref(index + 1)
			}
		}
	}
	if decision == 0 || target == 0 {
		t.Fatal("missing source operations")
	}
	for _, edge := range graph.Edges {
		if edge.Kind == pdg.Execution && edge.Source == target && edge.Target == decision {
			t.Fatal("range target falsely must execute before deciding whether to iterate")
		}
	}
}
