package patterns

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func semanticFlowFixture(t *testing.T) *pdg.Graph {
	t.Helper()
	pool := pdg.NewBuilder()
	source := pool.MatchLabel(pdg.MatchLabel{Kind: pool.Text(string(Call)), Callee: pool.Text("os.Getenv")})
	sink := pool.MatchLabel(pdg.MatchLabel{Kind: pool.Text(string(Call)), Callee: pool.Text("os/exec.Command")})
	graph := &pdg.Graph{Tables: pool.Tables,
		Symbols:     []pdg.Symbol{{ID: pool.Text("binding")}},
		Definitions: []pdg.Definition{{ID: pool.Text("definition"), Node: 1, Symbol: 1}},
		Locations:   []pdg.Location{{ID: pool.Text("storage")}},
		Nodes: []pdg.Node{
			{ID: pool.Text("source"), Category: pdg.Call, MatchLabel: source, Line: 1},
			{ID: pool.Text("assignment"), Category: pdg.Assignment, Line: 2},
			{ID: pool.Text("read"), Category: pdg.Read, Line: 3},
			{ID: pool.Text("sink"), Category: pdg.Call, MatchLabel: sink, Line: 4},
		}, Edges: []pdg.Edge{
			{ID: pool.Text("e1"), Source: 1, Target: 2, Kind: pdg.Data, Subkind: pool.Text("reaching-definition"), Symbol: 1, Definition: 1, Position: 1},
			{ID: pool.Text("e2"), Source: 2, Target: 3, Kind: pdg.Data, Location: 1, Position: 1},
			{ID: pool.Text("e3"), Source: 3, Target: 4, Kind: pdg.Data, Position: 1},
		}}
	return graph
}

func TestCanonicalFlowSurvivesUnlabelledOperations(t *testing.T) {
	graph := semanticFlowFixture(t)
	matching := MiningView(graph)
	if len(matching.Nodes) != 2 || len(matching.Edges) != 0 {
		t.Fatal("clone matching view changed")
	}
	fact := &MiningFacts{ID: "flow", Path: "flow.go", Pdg: matching, Canonical: graph}
	semantic := semanticFact(fact)
	if len(semantic.Pdg.Nodes) != 4 || len(semantic.Pdg.Edges) != 3 {
		t.Fatal("canonical operations or edges lost")
	}
	if semantic.Canonical != graph {
		t.Fatal("canonical evidence copied or lost")
	}
	if semantic.Canonical.Edges[0].Definition != 1 || semantic.Canonical.Edges[1].Location != 1 {
		t.Fatal("dependency identity lost")
	}
	flows := FindTaintFlows([]*MiningFacts{fact})
	if len(flows) != 1 {
		t.Fatalf("want source-to-sink flow through unlabelled nodes, got %v", flows)
	}
	if flows[0].SourceNode != "source" || flows[0].SinkNode != "sink" {
		t.Fatalf("wrong canonical evidence: %+v", flows[0])
	}
	candidates := TaintFlowCandidates(flows, []*MiningFacts{fact})
	if candidates[0].Analysis.Status != "partial" {
		t.Fatal("unknown facts reported as complete")
	}
	if len(fact.Pdg.Nodes) != 2 {
		t.Fatal("input matching graph mutated")
	}
}

func TestCanonicalSlicesRetainFormalInterfaceAndReturns(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables}
	for i := range 25 {
		category := pdg.Computation
		if i == 0 {
			category = pdg.FormalInput
		}
		if i == 23 {
			category = pdg.Return
		}
		if i == 24 {
			category = pdg.FormalOutput
		}
		graph.Nodes = append(graph.Nodes, pdg.Node{ID: pool.Text(fmt.Sprintf("n%d", i)), Category: category, Line: uint32(i + 1)})
		if i > 0 {
			graph.Edges = append(graph.Edges, pdg.Edge{Source: pdg.Ref(i), Target: pdg.Ref(i + 1), Kind: pdg.Data})
		}
	}
	fact := &MiningFacts{ID: "chain", Pdg: MiningView(graph), Canonical: graph}
	facts := []*MiningFacts{fact}
	chops := FindChops(facts)
	if len(chops) != 1 || chops[0].ChopSize != 25 {
		t.Fatalf("formal interface dropped from chop: %v", chops)
	}
	thin := FindThinSlices(facts)
	if len(thin) != 1 || thin[0].SliceSize != 24 || thin[0].SeedNode != "n23" {
		t.Fatalf("return chain lost: %v", thin)
	}
	barriers := FindBarrierSlices(facts)
	if len(barriers) != 1 || barriers[0].SliceSize != 24 {
		t.Fatalf("barrier slice lost: %v", barriers)
	}
}

func TestCanonicalCoverageExplainsMissingAndApproximateFacts(t *testing.T) {
	graph := semanticFlowFixture(t)
	fact := &MiningFacts{ID: "flow", Pdg: MiningView(graph), Canonical: graph}
	coverage := semanticCoverage([]*MiningFacts{fact, {ID: "legacy", Pdg: &MiningGraph{}}, nil})
	for _, row := range coverage {
		if row.Partial != 1 || row.Unknown != 1 || row.Unavailable != 1 || row.Complete != 0 {
			t.Fatalf("wrong coverage: %+v", row)
		}
	}
	text := coverageText(coverage)
	if !strings.Contains(text, "no findings does not prove clean flow") {
		t.Fatal("coverage warning absent")
	}
}

func TestSemanticTraversalLeavesMatchingRefinementUnchanged(t *testing.T) {
	graph := semanticFlowFixture(t)
	matching := MiningView(graph)
	before := NewWl(matching).rounds
	fact := &MiningFacts{ID: "flow", Pdg: matching, Canonical: graph}
	FindTaintFlows([]*MiningFacts{fact})
	FindThinSlices([]*MiningFacts{fact})
	if !reflect.DeepEqual(before, NewWl(matching).rounds) {
		t.Fatal("semantic analysis changed clone matching")
	}
}

func TestGoMethodNamesMatchTaintRules(t *testing.T) {
	source := PdgNode{Kind: Call, CalleeID: "net/http.Request.FormValue"}
	sink := PdgNode{Kind: Call, CalleeID: "database/sql.DB.Exec"}
	if !isTaintSource(source) || matchTaintSink(sink) == nil {
		t.Fatal("canonical Go method IDs do not match taint catalog")
	}
}

func TestControlOperationsAreNotThinSliceProducers(t *testing.T) {
	for _, kind := range []NodeKind{Branch, Loop, Iterate, Match, Try} {
		if isProducer(PdgNode{Kind: kind}) {
			t.Fatalf("control operation %s treated as a producer", kind)
		}
	}
}

func TestDependencyCoverageRequiresSupportedEdges(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables}
	for _, name := range []string{"node_extraction", "scalar_reaching_definitions", "returns", "calls", "control_dependencies"} {
		graph.Capabilities = append(graph.Capabilities, pdg.NamedCapability{Name: pool.Text(name), Evidence: pool.Evidence(pdg.Supported, "fixture")})
	}
	fact := &MiningFacts{ID: "supported", Pdg: MiningView(graph), Canonical: graph}
	if got := analysisEvidence(fact, ThinSliceKind).Status; got != "complete" {
		t.Fatalf("supported facts: %s", got)
	}
	graph.Edges = []pdg.Edge{{Kind: pdg.Data, Evidence: pool.Evidence(pdg.Approximated, "fixture")}}
	if got := analysisEvidence(fact, ThinSliceKind).Status; got != "partial" {
		t.Fatalf("approximate data flow: %s", got)
	}
	graph.Edges[0].Kind = pdg.Execution
	if got := analysisEvidence(fact, ThinSliceKind).Status; got != "complete" {
		t.Fatalf("irrelevant execution evidence gated a data-only analysis: %s", got)
	}
}

func TestNativeReportPublishesCoverageWithoutFindings(t *testing.T) {
	pool := pdg.NewBuilder()
	fact := &FuncFacts{ID: "empty", Pdg: &pdg.Graph{Tables: pool.Tables}}
	report := Run([]*FuncFacts{fact}, Options{})
	if len(report.AnalysisCoverage) != 4 {
		t.Fatal("zero findings lost coverage")
	}
	for _, row := range report.AnalysisCoverage {
		if row.Partial != 1 || row.Complete != 0 {
			t.Fatalf("missing capabilities reported as complete: %+v", row)
		}
	}
	if len(report.Candidates) != 0 {
		t.Fatal("coverage creates a fake finding")
	}
}

func TestUnsupportedLoopFlowIsReportedInCoverage(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Control, OriginalKind: pool.Known("*ast.ForStmt")}}}
	for _, name := range []string{"node_extraction", "scalar_reaching_definitions", "returns"} {
		graph.Capabilities = append(graph.Capabilities, pdg.NamedCapability{Name: pool.Text(name), Evidence: pool.Evidence(pdg.Supported, "fixture")})
	}
	fact := &MiningFacts{ID: "loop", Pdg: MiningView(graph), Canonical: graph}
	evidence := analysisEvidence(fact, ThinSliceKind)
	if evidence.Status != "partial" || !strings.Contains(strings.Join(evidence.Missing, " "), "loop_carried_dependencies") {
		t.Fatalf("unsupported loop flow hidden: %+v", evidence)
	}
}
