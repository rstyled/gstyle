package patterns

import (
	"slices"
	"testing"
)

func TestAuditChopSorted(t *testing.T) {
	g := &MiningGraph{Nodes: make([]PdgNode, 12)}
	for i := 0; i < 11; i++ {
		g.Edges = append(g.Edges, PdgEdge{From: i, To: i + 1, Kind: Data})
	}
	for range 20 {
		got := Chop(g, 0, 11)
		if !slices.IsSorted(got) {
			t.Fatalf("Chop promises sorted nodes, got %v", got)
		}
	}
}
