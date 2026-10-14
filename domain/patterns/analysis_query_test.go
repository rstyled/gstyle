package patterns

import (
	"math"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestCoverageRejectsUnavailableEvidence(t *testing.T) {
	for _, graph := range []*pdg.Graph{nil, {}, {Tables: &pdg.Tables{}}} {
		if got := evidenceStatus(graph, 1); got != pdg.Unknown {
			t.Fatalf("missing table status = %v", got)
		}
		if graphCapability(graph, "node_extraction") != pdg.Unknown {
			t.Fatal("missing capability is supported")
		}
	}
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables, Edges: []pdg.Edge{{Kind: pdg.Data}}}
	for _, ref := range []pdg.Ref{0, 1, math.MaxUint32} {
		graph.Edges[0].Evidence = ref
		if supportedEdges(graph, TaintFlowKind) {
			t.Fatalf("bad evidence ref %d certifies an edge", ref)
		}
	}
}

func TestCoverageDistinguishesCapabilityAndEdgePolicies(t *testing.T) {
	pool := pdg.NewBuilder()
	for _, status := range []pdg.Status{pdg.Unknown, pdg.Known, pdg.Unsupported, pdg.NotApplicable, pdg.Supported, pdg.Approximated} {
		ref := pool.Evidence(status, "fixture")
		graph := &pdg.Graph{Tables: pool.Tables, Edges: []pdg.Edge{{Kind: pdg.Data, Evidence: ref}}}
		want := status == pdg.Known || status == pdg.Supported
		if got := supportedEdges(graph, TaintFlowKind); got != want {
			t.Fatalf("status %v: edge support %v, want %v", status, got, want)
		}
	}
	if !supportedCapability(pdg.NotApplicable) {
		t.Fatal("explicitly inapplicable capability is a gap")
	}
	if supportedEdges(nil, TaintFlowKind) || supportedEdges(&pdg.Graph{}, TaintFlowKind) {
		t.Fatal("missing graph is complete")
	}
}

func TestMalformedDependencyEvidenceIsPartialForEverySemanticPattern(t *testing.T) {
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables}
	for _, name := range []string{"node_extraction", "scalar_reaching_definitions", "returns", "calls", "control_dependencies"} {
		graph.Capabilities = append(graph.Capabilities, pdg.NamedCapability{Name: pool.Text(name), Evidence: pool.Evidence(pdg.Supported, "fixture")})
	}
	fact := &MiningFacts{ID: "evidence", Pdg: &MiningGraph{}, Canonical: graph}
	for _, kind := range semanticAnalysisKinds {
		if got := analysisEvidence(fact, kind).Status; got != "complete" {
			t.Fatalf("initial %s: %s", kind, got)
		}
		graph.Edges = []pdg.Edge{{Kind: pdg.Data, Evidence: math.MaxUint32}}
		if got := analysisEvidence(fact, kind).Status; got != "partial" {
			t.Fatalf("invalid %s: %s", kind, got)
		}
		graph.Edges = nil
	}
}
