package errorflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func copyQuery(t *testing.T, size, bits int) *Query {
	t.Helper()
	pool := pdg.NewBuilder()
	graph := &pdg.Graph{Tables: pool.Tables, Symbols: make([]pdg.Symbol, size)}
	var attrs []pdg.Attributes
	add := func(category pdg.Category, attributes pdg.Attributes) pdg.Ref {
		ref := pdg.Ref(len(graph.Nodes) + 1)
		graph.Nodes = append(graph.Nodes, pdg.Node{Category: category, Attributes: ref})
		attrs = append(attrs, attributes)
		return ref
	}
	for i := range size {
		ref := pdg.Ref(i + 1)
		add(pdg.Assignment, pdg.Attributes{Writes: []pdg.Access{{Symbol: ref, Position: 1}}, Defines: []pdg.Ref{ref}})
		graph.Definitions = append(graph.Definitions, pdg.Definition{Node: ref, Symbol: ref})
	}
	for source := range size {
		for target := range size {
			if bits&(1<<(source*size+target)) == 0 {
				continue
			}
			read := add(pdg.Read, pdg.Attributes{Reads: []pdg.Access{{Symbol: pdg.Ref(source + 1)}}})
			graph.Edges = append(graph.Edges,
				pdg.Edge{Source: pdg.Ref(source + 1), Target: read, Kind: pdg.Data, Definition: pdg.Ref(source + 1), Symbol: pdg.Ref(source + 1), Subkind: pool.Text("reaching-definition")},
				pdg.Edge{Source: read, Target: pdg.Ref(target + 1), Kind: pdg.Data, Position: 1, Subkind: pool.Text("source-value")})
		}
	}
	read := add(pdg.Read, pdg.Attributes{Reads: []pdg.Access{{Symbol: pdg.Ref(size)}}})
	sink := add(pdg.Return, pdg.Attributes{})
	graph.Edges = append(graph.Edges,
		pdg.Edge{Source: pdg.Ref(size), Target: read, Kind: pdg.Data, Definition: pdg.Ref(size), Symbol: pdg.Ref(size), Subkind: pool.Text("reaching-definition")},
		pdg.Edge{Source: read, Target: sink, Kind: pdg.Data, Position: 1, Subkind: pool.Text("source-value")})
	if err := pdg.PackAttributes(graph, attrs); err != nil {
		t.Fatal(err)
	}
	query := &Query{graph: graph, definitions: make([][]pdg.Ref, size), values: make([][]pdg.Edge, len(graph.Nodes))}
	query.index()
	return query
}
func copyReachability(size, bits, seed int) bool {
	seen := make([]bool, size)
	queue := []int{seed}
	seen[seed] = true
	for cursor := 0; cursor < len(queue); cursor++ {
		for target := range size {
			if bits&(1<<(queue[cursor]*size+target)) != 0 && !seen[target] {
				seen[target] = true
				queue = append(queue, target)
			}
		}
	}
	return seen[size-1]
}
func TestCanonicalCopiesAgainstReachabilityOracle(t *testing.T) {
	checked := 0
	for size := 1; size <= 3; size++ {
		for bits := 0; bits < 1<<(size*size); bits++ {
			query := copyQuery(t, size, bits)
			for seed := range size {
				got := query.Assignment(pdg.Ref(seed+1), 1)
				want := copyReachability(size, bits, seed)
				if !got.Known || got.Used != want {
					t.Fatalf("size=%d bits=%d seed=%d: result=%+v, want used=%v", size, bits, seed, got, want)
				}
				checked++
			}
		}
	}
	t.Logf("checked %d copy graph/seed cases", checked)
}
