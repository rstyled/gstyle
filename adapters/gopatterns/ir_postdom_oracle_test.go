package gopatterns

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

// Enumerate every forward-edge DAG through six blocks. The oracle intersects
// concrete terminal paths; it does not reverse the graph or use dominators.
func TestPostDominatorsAndControlMatchTerminalPaths(t *testing.T) {
	graphs := 0
	for size := 1; size <= 6; size++ {
		pairs := [][2]int{}
		for from := 0; from < size; from++ {
			for to := from + 1; to < size; to++ {
				pairs = append(pairs, [2]int{from, to})
			}
		}
		for mask := 0; mask < 1<<len(pairs); mask++ {
			flow := &cfg.CFG{Blocks: make([]*cfg.Block, size)}
			for i := range flow.Blocks {
				flow.Blocks[i] = &cfg.Block{Index: int32(i)}
			}
			for bit, pair := range pairs {
				if mask&(1<<bit) != 0 {
					flow.Blocks[pair[0]].Succs = append(flow.Blocks[pair[0]].Succs, flow.Blocks[pair[1]])
				}
			}
			checkPostdomOracle(t, flow)
			graphs++
		}
	}
	if graphs != 33867 {
		t.Fatalf("graphs=%d", graphs)
	}
}

func terminalPathIntersection(block *cfg.Block, path uint64, intersection *uint64) {
	path |= 1 << block.Index
	if len(block.Succs) == 0 {
		*intersection &= path
		return
	}
	for _, next := range block.Succs {
		terminalPathIntersection(next, path, intersection)
	}
}

func checkPostdomOracle(t *testing.T, flow *cfg.CFG) {
	t.Helper()
	dom := newExecutionDominators(flow)
	post := normalPostDominators(flow, dom)
	expected := make([]uint64, len(flow.Blocks))
	for _, index := range dom.order {
		expected[index] = ^uint64(0)
		terminalPathIntersection(flow.Blocks[index], 0, &expected[index])
		for _, target := range dom.order {
			if got, want := executionDominates(post, target+1, index+1), expected[index]&(1<<target) != 0; got != want {
				t.Fatalf("postdom %d,%d=%v,want %v; CFG=%v", target, index, got, want, flow.Blocks)
			}
		}
	}
	pool := pdg.NewBuilder()
	builder := &irBuilder{pool: pool, graph: pdg.Graph{Tables: pool.Tables}}
	analysis := normalFlowAnalysis{builder: builder, post: post, operations: make([][]pdg.Ref, len(flow.Blocks))}
	for i := range analysis.operations {
		analysis.operations[i] = []pdg.Ref{pdg.Ref(i + 1)}
	}
	analysis.control(flow, dom)
	got, want := map[[3]int]bool{}, map[[3]int]bool{}
	for _, edge := range builder.graph.Edges {
		got[[3]int{int(edge.Source) - 1, int(edge.Position) - 1, int(edge.Target) - 1}] = true
	}
	for _, from := range dom.order {
		for arm, next := range flow.Blocks[from].Succs {
			for _, target := range dom.order {
				if expected[next.Index]&(1<<target) != 0 && expected[from]&(1<<target) == 0 {
					want[[3]int{from, arm, target}] = true
				}
			}
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("control got %v,want %v", got, want)
	}
}

// A maximal path avoids z iff it can end at a real terminal or revisit a
// vertex without encountering z. This search uses neither fixed points nor
// the producer's reverse/predecessor representation.
func avoidingMaximalPath(flow *cfg.CFG, start, z int, active []bool) bool {
	if start == z {
		return false
	}
	if active[start] || len(flow.Blocks[start].Succs) == 0 {
		return true
	}
	active[start] = true
	defer func() { active[start] = false }()
	for _, next := range flow.Blocks[start].Succs {
		if avoidingMaximalPath(flow, int(next.Index), z, active) {
			return true
		}
	}
	return false
}

func TestMaximalPostDominanceAndControlExhaustive(t *testing.T) {
	graphs := 0
	for size := 1; size <= 4; size++ {
		for mask := 0; mask < 1<<(size*size); mask++ {
			flow := &cfg.CFG{Blocks: make([]*cfg.Block, size)}
			for index := range flow.Blocks {
				flow.Blocks[index] = &cfg.Block{Index: int32(index)}
			}
			for from := range flow.Blocks {
				for to := range flow.Blocks {
					if mask&(1<<(from*size+to)) != 0 {
						flow.Blocks[from].Succs = append(flow.Blocks[from].Succs, flow.Blocks[to])
					}
				}
			}
			checkMaximalOracle(t, flow)
			graphs++
		}
	}
	if graphs != 66066 {
		t.Fatalf("graphs=%d", graphs)
	}
}

func checkMaximalOracle(t *testing.T, flow *cfg.CFG) {
	t.Helper()
	dom := newExecutionDominators(flow)
	sets := maximalPostDominators(flow, dom)
	expected := make([][]bool, len(flow.Blocks))
	for _, from := range dom.order {
		expected[from] = make([]bool, len(flow.Blocks))
		for _, target := range dom.order {
			want := !avoidingMaximalPath(flow, from, target, make([]bool, len(flow.Blocks)))
			expected[from][target] = want
			if got := sets.contains(from, target); got != want {
				t.Fatalf("post(%d,%d)=%v,want %v CFG=%v", from, target, got, want, flow.Blocks)
			}
		}
	}
	pool := pdg.NewBuilder()
	builder := &irBuilder{pool: pool, graph: pdg.Graph{Tables: pool.Tables}}
	analysis := normalFlowAnalysis{builder: builder, operations: make([][]pdg.Ref, len(flow.Blocks))}
	for index := range analysis.operations {
		analysis.operations[index] = []pdg.Ref{pdg.Ref(index + 1)}
	}
	analysis.maximalControl(flow, dom, sets, nil)
	got, want := map[[3]int]bool{}, map[[3]int]bool{}
	for _, edge := range builder.graph.Edges {
		key := [3]int{int(edge.Source) - 1, int(edge.Position) - 1, int(edge.Target) - 1}
		if got[key] {
			t.Fatalf("duplicate edge %v", key)
		}
		got[key] = true
	}
	for _, from := range dom.order {
		for arm, next := range flow.Blocks[from].Succs {
			for _, target := range dom.order {
				if expected[next.Index][target] && (target == from || !expected[from][target]) {
					want[[3]int{from, arm, target}] = true
				}
			}
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("control got %v,want %v CFG=%v", got, want, flow.Blocks)
	}
}
