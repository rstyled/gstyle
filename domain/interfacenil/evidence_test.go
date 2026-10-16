package interfacenil

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestExplanationMatchesFindAndKeepsGraphIdentity(t *testing.T) {
	graph := interfaceOperatorGraph(t, pdg.Known, "parentheses")
	graphs := []pdg.Graph{{}, graph}
	proofs := Explain(graphs)
	if len(proofs) != 1 {
		t.Fatalf("got %d proofs", len(proofs))
	}
	p := proofs[0]
	if p.Comparison != (ProofSite{Graph: 1, Node: 3}) {
		t.Fatalf("wrong comparison: %+v", p)
	}
	if len(Find(graphs)[1]) != 1 {
		t.Fatal("Find and Explain disagree")
	}
	if len(p.Roots) != 1 || p.Roots[0] != (ProofSite{Graph: 1, Node: 2}) {
		t.Fatalf("wrong roots: %+v", p.Roots)
	}
	if len(p.Steps) != 2 || p.Steps[0].Seed || !p.Steps[1].Seed {
		t.Fatalf("wrong transfer steps: %+v", p.Steps)
	}
	if p.Steps[0].Dependencies[0] != (ProofSite{Graph: 1, Node: 1}) {
		t.Fatal("lost dependency identity")
	}
}

func TestExplanationRejectsMissingProofEvidence(t *testing.T) {
	for _, change := range []string{"capability", "edge", "operator"} {
		t.Run(change, func(t *testing.T) {
			graph := interfaceOperatorGraph(t, pdg.Known, "parentheses")
			switch change {
			case "capability":
				graph.Capabilities = nil
			case "edge":
				graph.Edges = nil
			case "operator":
				graph = interfaceOperatorGraph(t, pdg.Unknown, "parentheses")
			}
			if len(Explain([]pdg.Graph{graph})) != 0 {
				t.Fatal("missing evidence produced a proof")
			}
		})
	}
}
