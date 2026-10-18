package gopatterns

import (
	"go/ast"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

// The oracle searches a definition's paths until a write kills it. It has no
// predecessor unions, worklist states, transfer iteration, or producer CFG logic.
func TestReachingFixedPointAgainstPathOracle(t *testing.T) {
	cases := 0
	for size := 1; size <= 4; size++ {
		for bits := 0; bits < 1<<(size*size); bits++ {
			flow := &cfg.CFG{Blocks: make([]*cfg.Block, size)}
			for i := range flow.Blocks {
				flow.Blocks[i] = &cfg.Block{Index: int32(i)}
			}
			for from := range size {
				for to := range size {
					if bits&(1<<(from*size+to)) != 0 {
						flow.Blocks[from].Succs = append(flow.Blocks[from].Succs, flow.Blocks[to])
					}
				}
			}
			for layout := range 3 {
				a, writes := reachingOracleFixture(flow, layout)
				dom := newExecutionDominators(flow)
				a.predecessors = dom.predecessors
				a.fixedPoint(flow, dom)
				want := reachingPathOracle(flow, writes)
				for block := range size {
					for symbol := pdg.Ref(1); symbol <= 2; symbol++ {
						if !slices.Equal(a.outputs[block][symbol], want[block][symbol]) {
							t.Fatalf("size=%d graph=%x layout=%d block=%d symbol=%d: got %v want %v", size, bits, layout, block, symbol, a.outputs[block][symbol], want[block][symbol])
						}
					}
				}
				cases++
			}
		}
	}
	t.Logf("%d graph/definition layouts agree with independent path search", cases)
}

func reachingOracleFixture(flow *cfg.CFG, layout int) (*reachingAnalysis, []pdg.Ref) {
	b := &irBuilder{nodes: map[ast.Node]pdg.Ref{}}
	for symbol := pdg.Ref(1); symbol <= 2; symbol++ {
		b.graph.Nodes = append(b.graph.Nodes, pdg.Node{Category: pdg.FormalInput, Attributes: symbol})
		b.attributes = append(b.attributes, pdg.Attributes{})
		b.graph.Definitions = append(b.graph.Definitions, pdg.Definition{Node: symbol, Symbol: symbol})
	}
	writes := make([]pdg.Ref, len(flow.Blocks))
	for index, block := range flow.Blocks {
		symbol := pdg.Ref((index + layout) % 3)
		writes[index] = symbol
		ref := pdg.Ref(len(b.graph.Nodes) + 1)
		b.graph.Nodes = append(b.graph.Nodes, pdg.Node{Category: pdg.Assignment, Attributes: ref})
		attrs := pdg.Attributes{}
		if symbol != 0 {
			b.graph.Definitions = append(b.graph.Definitions, pdg.Definition{Node: ref, Symbol: symbol})
			attrs.Defines = []pdg.Ref{pdg.Ref(len(b.graph.Definitions))}
		}
		b.attributes = append(b.attributes, attrs)
		marker := &ast.EmptyStmt{}
		b.nodes[marker] = ref
		block.Nodes = []ast.Node{marker}
	}
	return &reachingAnalysis{builder: b, outputs: make([]reachingState, len(flow.Blocks))}, writes
}

func reachingPathOracle(flow *cfg.CFG, writes []pdg.Ref) []reachingState {
	result := make([]reachingState, len(flow.Blocks))
	reachable := reachingOracleReachable(flow)
	definition := pdg.Ref(0)
	for origin := -2; origin < len(flow.Blocks); origin++ {
		symbol := pdg.Ref(origin + 3)
		if origin >= 0 {
			symbol = writes[origin]
		}
		if symbol == 0 {
			continue
		}
		definition++
		if origin >= 0 && !reachable[origin] {
			continue
		}
		visited := make([]bool, len(flow.Blocks))
		var visit func(int)
		visit = func(index int) {
			if visited[index] || writes[index] == symbol {
				return
			}
			visited[index] = true
			for _, next := range flow.Blocks[index].Succs {
				visit(int(next.Index))
			}
		}
		if origin < 0 {
			visit(0)
		} else {
			visited[origin] = true
			for _, next := range flow.Blocks[origin].Succs {
				visit(int(next.Index))
			}
		}
		for index, present := range visited {
			if !present {
				continue
			}
			if result[index] == nil {
				result[index] = reachingState{}
			}
			result[index][symbol] = append(result[index][symbol], definition)
		}
	}
	return result
}

func TestReachingEntryBackEdgeRetainsInitialAndCarriedValues(t *testing.T) {
	flow := &cfg.CFG{Blocks: []*cfg.Block{{Index: 0}, {Index: 1}}}
	flow.Blocks[0].Succs = []*cfg.Block{flow.Blocks[1]}
	flow.Blocks[1].Succs = []*cfg.Block{flow.Blocks[0]}
	a, _ := reachingOracleFixture(flow, 0)
	a.predecessors = newExecutionDominators(flow).predecessors
	a.fixedPoint(flow, newExecutionDominators(flow))
	// Entry has no write. The first input and the second block's write both reach it.
	if !reflect.DeepEqual(a.outputs[0][1], []pdg.Ref{1, 3}) {
		t.Fatalf("entry loses initial or loop value: %v", a.outputs[0])
	}
}

func reachingOracleReachable(flow *cfg.CFG) []bool {
	seen := make([]bool, len(flow.Blocks))
	queue := []int{0}
	seen[0] = true
	for index := 0; index < len(queue); index++ {
		for _, next := range flow.Blocks[queue[index]].Succs {
			if seen[next.Index] {
				continue
			}
			seen[next.Index] = true
			queue = append(queue, int(next.Index))
		}
	}
	return seen
}

func TestExtendedReachingResourceFailureIsExplicit(t *testing.T) {
	flow := &cfg.CFG{Blocks: make([]*cfg.Block, controlVertexLimit+1)}
	for index := range flow.Blocks {
		flow.Blocks[index] = &cfg.Block{Index: int32(index)}
	}
	builder := &irBuilder{pool: pdg.NewBuilder(), extendedScalar: true}
	if _, err := builder.reachingCFG(&ast.BlockStmt{}, flow); err == nil {
		t.Fatal("oversized scalar flow silently accepted")
	}
}
