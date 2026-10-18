package gopatterns

import (
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestSharedJoinKeepsReadSitesAndAllInputs(t *testing.T) {
	for _, binding := range []pdg.Definition{{Symbol: 1}, {Location: 1}} {
		pool := pdg.NewBuilder()
		b := &irBuilder{pool: pool, generalBindings: true, graph: pdg.Graph{Tables: pool.Tables}}
		a := &reachingAnalysis{builder: b}
		var inputs []pdg.Ref
		for i := 0; i < 100; i++ {
			node := b.scalarEntity(1, pdg.Assignment, pdg.Attributes{})
			inputs = append(inputs, b.storageDefinition(node, binding.Symbol, binding.Location))
		}
		sites := map[pdg.Ref]bool{}
		for i := 0; i < 100; i++ {
			target := b.scalarEntity(pdg.Ref(i+2), pdg.Read, pdg.Attributes{})
			ref := a.joinedDefinition(1, target, slices.Clone(inputs))
			a.edge(ref, target, 0)
			sites[ref] = true
			definition := b.graph.Definitions[ref-1]
			if definition.Symbol != binding.Symbol || definition.Location != binding.Location {
				t.Fatal("binding changed")
			}
			if b.graph.Nodes[target-1].Span != pdg.Ref(i+2) {
				t.Fatal("read span changed")
			}
		}
		if len(b.graph.Edges) > 300 {
			t.Fatalf("repeated inputs expand to %d edges", len(b.graph.Edges))
		}
		incoming := map[pdg.Ref][]pdg.Ref{}
		for _, edge := range b.graph.Edges {
			incoming[edge.Target] = append(incoming[edge.Target], edge.Definition)
		}
		for site := range sites {
			seen := map[pdg.Ref]bool{}
			leaves := map[pdg.Ref]bool{}
			queue := []pdg.Ref{site}
			for len(queue) > 0 {
				ref := queue[0]
				queue = queue[1:]
				if seen[ref] {
					continue
				}
				seen[ref] = true
				parents := incoming[b.graph.Definitions[ref-1].Node]
				if len(parents) == 0 {
					leaves[ref] = true
					continue
				}
				queue = append(queue, parents...)
			}
			if len(leaves) != len(inputs) {
				t.Fatal("shared set loses or adds an input")
			}
			for _, input := range inputs {
				if !leaves[input] {
					t.Fatal("shared set changes an input")
				}
			}
		}
	}
}

func TestSharedJoinChecksHashBucketContents(t *testing.T) {
	a := &reachingAnalysis{joins: map[uint64][]definitionJoin{7: {
		{binding: 1, inputs: []pdg.Ref{1, 2}, definition: 10},
		{binding: 2, inputs: []pdg.Ref{1, 3}, definition: 11},
	}}}
	if a.findDefinitionJoin(7, 1, []pdg.Ref{1, 3}) != 0 {
		t.Fatal("collision combines different sets")
	}
	if a.findDefinitionJoin(7, 2, []pdg.Ref{1, 3}) != 11 {
		t.Fatal("equal set not found")
	}
}

func TestDefinitionTreeSharesOverlappingSets(t *testing.T) {
	pool := pdg.NewBuilder()
	b := &irBuilder{pool: pool, generalBindings: true, graph: pdg.Graph{Tables: pool.Tables}}
	a := &reachingAnalysis{builder: b}
	var common []pdg.Ref
	for i := 0; i < 100; i++ {
		node := b.scalarEntity(1, pdg.Assignment, pdg.Attributes{})
		common = append(common, b.storageDefinition(node, 1, 0))
	}
	roots := map[pdg.Ref]bool{}
	for i := 0; i < 100; i++ {
		node := b.scalarEntity(2, pdg.Assignment, pdg.Attributes{})
		extra := b.storageDefinition(node, 1, 0)
		inputs := append(slices.Clone(common), extra)
		root := a.sharedDefinition(1, inputs)
		if roots[root] {
			t.Fatal("different input sets share a root")
		}
		roots[root] = true
	}
	if len(b.graph.Edges) > 3000 {
		t.Fatalf("overlapping sets expand to %d edges", len(b.graph.Edges))
	}
}
