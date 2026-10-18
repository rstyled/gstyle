package pdgmemory

import (
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

type fixture struct {
	graph pdg.Graph
	attrs []pdg.Attributes
	pool  *pdg.Builder
}

func newFixture(size int) *fixture {
	pool := pdg.NewBuilder()
	f := &fixture{graph: pdg.Graph{Tables: pool.Tables}, pool: pool}
	for index := range size {
		f.graph.Symbols = append(f.graph.Symbols, pdg.Symbol{ID: pool.Text("pointer"), Role: pdg.LocalRole, Extensions: pool.SingletonExtension("gstyle.memory:reference", "true")})
		_ = index
	}
	f.graph.Symbols = append(f.graph.Symbols, pdg.Symbol{ID: pool.Text("storage"), Role: pdg.LocalRole})
	return f
}
func (f *fixture) node(category pdg.Category, attrs pdg.Attributes) pdg.Ref {
	ref := pdg.Ref(len(f.graph.Nodes) + 1)
	f.graph.Nodes = append(f.graph.Nodes, pdg.Node{Category: category, Attributes: ref})
	f.attrs = append(f.attrs, attrs)
	return ref
}
func (f *fixture) read(symbol pdg.Ref, reference bool) pdg.Ref {
	attrs := pdg.Attributes{Reads: []pdg.Access{{Symbol: symbol}}}
	if reference {
		attrs.Extensions = f.pool.SingletonExtension("gstyle.memory:reference", "true")
	}
	return f.node(pdg.Read, attrs)
}
func (f *fixture) operand(source, target pdg.Ref) {
	f.graph.Edges = append(f.graph.Edges, pdg.Edge{Source: source, Target: target, Kind: pdg.Data, Position: 1, Subkind: f.pool.Text("source-value")})
}
func (f *fixture) assignment(symbol, source pdg.Ref) {
	node := f.node(pdg.Assignment, pdg.Attributes{Writes: []pdg.Access{{Symbol: symbol, Position: 1}}})
	f.operand(source, node)
}
func (f *fixture) lookup(node pdg.Node) pdg.Attributes { return f.attrs[node.Attributes-1] }
func (f *fixture) address(source pdg.Ref) pdg.Ref {
	result := f.node(pdg.Computation, pdg.Attributes{Operator: f.pool.Known("&"), Extensions: f.pool.SingletonExtension("gstyle.memory:reference", "true")})
	f.operand(source, result)
	return result
}
func (f *fixture) load(source pdg.Ref) pdg.Ref {
	result := f.node(pdg.Read, pdg.Attributes{Operator: f.pool.Known("dereference"), Reads: []pdg.Access{{UnresolvedReason: f.pool.Text("unknown storage")}}})
	f.operand(source, result)
	return result
}

// Independent reachability on variable-copy graphs predicts the address seed's
// propagation. The oracle does not execute the PDG alias constraints.
func TestCopyAliasesAgainstReachabilityOracle(t *testing.T) {
	cases := 0
	for size := 1; size <= 3; size++ {
		for bits := 0; bits < 1<<(size*size); bits++ {
			for seed := range size {
				f := newFixture(size)
				f.assignment(pdg.Ref(seed+1), f.address(f.read(pdg.Ref(size+1), false)))
				for from := range size {
					for to := range size {
						if bits&(1<<(from*size+to)) != 0 {
							f.assignment(pdg.Ref(to+1), f.read(pdg.Ref(from+1), true))
						}
					}
				}
				loads := make([]pdg.Ref, size)
				for index := range size {
					loads[index] = f.load(f.read(pdg.Ref(index+1), true))
				}
				model, err := Analyze(&f.graph, f.lookup)
				if err != nil {
					t.Fatal(err)
				}
				reachable := copyReachable(size, bits, seed)
				for index, load := range loads {
					got := model.Targets(load)
					want := []Place(nil)
					if reachable[index] {
						want = []Place{{Symbol: pdg.Ref(size + 1)}}
					}
					if !slices.Equal(got, want) {
						t.Fatalf("size=%d graph=%x seed=%d pointer=%d got=%v want=%v", size, bits, seed, index, got, want)
					}
				}
				cases++
			}
		}
	}
	t.Logf("%d alias copy/seed cases agree with independent reachability", cases)
}
func copyReachable(size, bits, seed int) []bool {
	seen := make([]bool, size)
	seen[seed] = true
	queue := []int{seed}
	for cursor := 0; cursor < len(queue); cursor++ {
		for target := range size {
			if bits&(1<<(queue[cursor]*size+target)) != 0 && !seen[target] {
				seen[target] = true
				queue = append(queue, target)
			}
		}
	}
	return seen
}

func TestUnknownPointerIncludesAddressedStorage(t *testing.T) {
	f := newFixture(2)
	f.graph.Symbols[0].Role = pdg.ParameterRole
	f.assignment(2, f.address(f.read(3, false)))
	load := f.load(f.read(1, true))
	model, err := Analyze(&f.graph, f.lookup)
	if err != nil {
		t.Fatal(err)
	}
	targets := model.Targets(load)
	for _, want := range []Place{{}, {Symbol: 3}} {
		if !slices.Contains(targets, want) {
			t.Fatalf("unknown pointer omits %v: %v", want, targets)
		}
	}
}
func TestOpaqueCallUsesResolvedIdentity(t *testing.T) {
	f := newFixture(0)
	for _, name := range []string{"len", "example.len", "new", "example.new"} {
		attrs := pdg.Attributes{Callee: f.pool.Known(name)}
		want := name == "example.len" || name == "example.new"
		if got := OpaqueCall(&f.graph, pdg.Node{Category: pdg.Call}, attrs); got != want {
			t.Fatalf("%s opaque=%v want=%v", name, got, want)
		}
	}
}
func TestAliasBoundaryValidation(t *testing.T) {
	if _, err := Analyze(nil, nil); err == nil {
		t.Fatal("missing graph accepted")
	}
	f := newFixture(0)
	node := f.node(pdg.Read, pdg.Attributes{})
	f.operand(node+1, node)
	if _, err := Analyze(&f.graph, f.lookup); err == nil {
		t.Fatal("invalid operand reference accepted")
	}
	f.graph.Edges = nil
	f.attrs[0].Reads = []pdg.Access{{Symbol: 2}}
	if _, err := Analyze(&f.graph, f.lookup); err == nil {
		t.Fatal("invalid binding accepted")
	}
}

func TestClosureCapturesCreateAddressableCells(t *testing.T) {
	f := newFixture(1)
	f.node(pdg.Other, pdg.Attributes{Operator: f.pool.Known("closure"), Symbols: []pdg.Ref{1}})
	model, err := Analyze(&f.graph, f.lookup)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(model.Places(), Place{Symbol: 1}) {
		t.Fatal("captured cell is absent from may-alias storage")
	}
	f.attrs[0].Symbols = []pdg.Ref{pdg.Ref(len(f.graph.Symbols) + 1)}
	if _, err := Analyze(&f.graph, f.lookup); err == nil {
		t.Fatal("invalid capture reference accepted")
	}
}

func TestCapturedBindingsAreExternalAddressableStorage(t *testing.T) {
	f := newFixture(1)
	f.graph.Symbols[0].Role = pdg.CaptureRole
	f.graph.Symbols[1].Role = pdg.CaptureRole
	load := f.load(f.read(1, true))
	model, err := Analyze(&f.graph, f.lookup)
	if err != nil {
		t.Fatal(err)
	}
	for _, symbol := range []pdg.Ref{1, 2} {
		if !slices.Contains(model.Places(), Place{Symbol: symbol}) {
			t.Fatalf("captured cell %d has no storage", symbol)
		}
	}
	if !slices.Contains(model.Targets(load), Place{}) {
		t.Fatal("captured pointer loses its unknown external referent")
	}
}
