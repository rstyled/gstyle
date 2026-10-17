package outputflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strconv"
	"testing"
)

// The oracle enumerates paths, rejecting a dead end, unsafe vertex or repeated
// vertex on the current path. It does not use the producer's backward fixed point.
func oracleCompletion(next [][]int, node int, unsafe bool, nilReturn bool, path []bool) bool {
	if node == len(next)-1 {
		return nilReturn
	}
	if unsafe && node == 0 {
		return false
	}
	if path[node] || len(next[node]) == 0 {
		return false
	}
	path[node] = true
	defer func() { path[node] = false }()
	for _, target := range next[node] {
		if !oracleCompletion(next, target, unsafe, nilReturn, path) {
			return false
		}
	}
	return true
}
func completionFixture(t *testing.T, next [][]int, unsafe, nilReturn bool) *SuccessFlow {
	t.Helper()
	pool := pdg.NewBuilder()
	n := len(next)
	graph := &pdg.Graph{Tables: pool.Tables, Nodes: make([]pdg.Node, n+2), ValueFlow: &pdg.NormalFlow{Entry: 1, Blocks: make([]pdg.FlowBlock, n+1)}}
	attrs := make([]pdg.Attributes, n+2)
	graph.Nodes[0].Category = pdg.Call
	graph.ValueFlow.Blocks[0] = pdg.FlowBlock{Operations: []pdg.Ref{1}, Successors: []pdg.Ref{2}, Reachable: true}
	for i, targets := range next {
		category := pdg.Computation
		if unsafe && i == 0 {
			category = pdg.Call
		}
		if i == n-1 {
			category = pdg.Return
		}
		graph.Nodes[i+1].Category = category
		attrs[i+1].Extensions = pool.SingletonExtension("gstyle.runtime:normal-completion", "true")
		block := pdg.FlowBlock{Operations: []pdg.Ref{pdg.Ref(i + 2)}}
		for _, target := range targets {
			block.Successors = append(block.Successors, pdg.Ref(target+2))
		}
		graph.ValueFlow.Blocks[i+1] = block
	}
	graph.Nodes[n+1].Category = pdg.Computation
	if nilReturn {
		attrs[n+1].Extensions = pool.SingletonExtension("gstyle.runtime:nil", "true")
	}
	graph.Edges = []pdg.Edge{{Source: pdg.Ref(n + 2), Target: pdg.Ref(n + 1), Kind: pdg.Data, Position: 1, Subkind: pool.Text("source-value")}}
	for index := range graph.Nodes {
		graph.Nodes[index].Attributes = pdg.Ref(index + 1)
	}
	if err := pdg.PackAttributes(graph, attrs); err != nil {
		t.Fatal(err)
	}
	graph.ValueFlow.RefreshReachability()
	completeHeader(graph, pool)
	inputs, err := bindingflow.New(graph)
	if err != nil {
		t.Fatal(err)
	}
	proof := &SuccessFlow{graph: graph, inputs: inputs, slot: 1, sites: make([]operationSite, len(graph.Nodes)), good: make([]bool, len(graph.ValueFlow.Blocks))}
	if err := proof.index(); err != nil {
		t.Fatal(err)
	}
	proof.solve()
	return proof
}
func TestSuccessFlowAgainstIndependentPathOracle(t *testing.T) {
	checked := 0
	for size := 1; size <= 3; size++ {
		for bits := 0; bits < 1<<((size-1)*size); bits++ {
			next := make([][]int, size)
			for source := 0; source < size-1; source++ {
				for target := 0; target < size; target++ {
					if bits&(1<<(source*size+target)) != 0 {
						next[source] = append(next[source], target)
					}
				}
			}
			for _, unsafe := range []bool{false, true} {
				for _, nilReturn := range []bool{false, true} {
					proof := completionFixture(t, next, unsafe, nilReturn)
					got := proof.After(1)
					want := oracleCompletion(next, 0, unsafe, nilReturn, make([]bool, size))
					if got != want {
						t.Fatalf("size=%d bits=%d unsafe=%v nil=%v: got %v want %v", size, bits, unsafe, nilReturn, got, want)
					}
					checked++
				}
			}
		}
	}
	t.Logf("checked %d graph/effect/return layouts", checked)
}

func completeHeader(g *pdg.Graph, b *pdg.Builder) {
	evidence := b.Evidence(pdg.Approximated, "Synthetic path oracle")
	origin := b.Origin(pdg.Analysis, "Synthetic path oracle")
	g.Profile = b.Policy("test/profile", "1")
	g.Producer = pdg.Producer{Name: b.Text("test"), Version: b.Text("1"), Language: b.Text("go"), LanguageVersion: b.Known("1.26"), Configuration: b.Text("{}")}
	g.Sources = []pdg.Source{{ID: b.Text("source"), Path: b.Text("fixture.go"), ContentIdentity: b.Missing(pdg.Unknown, "Synthetic")}}
	g.Function = pdg.Function{ID: b.Text("f"), Name: b.Known("f"), QualifiedName: b.Known("p.f"), Span: pdg.Span{Source: 1}, Interface: evidence, ReferenceCount: pdg.CountFact{Status: pdg.Known}}
	g.Conformance = pdg.Conformance{Status: pdg.NonConformant, Reasons: []pdg.Text{b.Text("Synthetic")}}
	for _, name := range pdg.RequiredCapabilities {
		g.Capabilities = append(g.Capabilities, pdg.NamedCapability{Name: b.Text(name), Evidence: evidence})
	}
	for i := range g.Nodes {
		g.Nodes[i].ID = b.Text("n" + strconv.Itoa(i))
		g.Nodes[i].Synthetic = true
		g.Nodes[i].OriginalKind = b.Known("synthetic")
		g.Nodes[i].Evidence = evidence
		g.Nodes[i].Provenance = origin
	}
	for i := range g.Edges {
		g.Edges[i].ID = b.Text("e" + strconv.Itoa(i))
		g.Edges[i].Evidence = evidence
		g.Edges[i].Provenance = origin
	}
}
