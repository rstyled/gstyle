package patterns

import (
	"slices"
	"testing"
)

func TestNeighborhoodTraversalOrderAndWorkspaceReuse(t *testing.T) {
	g := Graph{inc: make([][]Nb, 6), out: make([][]Nb, 6)}
	for _, edge := range [][2]int{{1, 0}, {0, 1}, {0, 2}, {0, 2}, {1, 3}, {2, 3}, {3, 4}, {2, 2}} {
		from, to := edge[0], edge[1]
		g.inc[to] = append(g.inc[to], Nb{node: uint64(from)})
		g.out[from] = append(g.out[from], Nb{node: uint64(to)})
	}
	ws := newNeighborhoodWLWorkspace(6)
	for _, tc := range []struct {
		start, hops int
		want        []int
	}{
		{0, 0, []int{0}},
		{0, 1, []int{0, 1, 2}},
		{0, 2, []int{0, 1, 2, 3}},
		{0, 3, []int{0, 1, 2, 3, 4}},
		{4, 1, []int{4, 3}},
		{4, 2, []int{4, 3, 1, 2}},
		{5, 2, []int{5}},
		{0, 2, []int{0, 1, 2, 3}},
	} {
		if got := ws.markKHop(&g, tc.start, tc.hops); !slices.Equal(got, tc.want) {
			t.Fatalf("start=%d hops=%d: got %v want %v", tc.start, tc.hops, got, tc.want)
		}
	}
}

func TestNeighborhoodTraversalReusesWarmedBuffer(t *testing.T) {
	const nodes = 2048
	g := Graph{inc: make([][]Nb, nodes), out: make([][]Nb, nodes)}
	for node := 1; node < nodes; node++ {
		g.out[0] = append(g.out[0], Nb{node: uint64(node)})
	}
	ws := newNeighborhoodWLWorkspace(nodes)
	ws.markKHop(&g, 0, 2)
	var reached int
	allocs := testing.AllocsPerRun(100, func() {
		reached = len(ws.markKHop(&g, 0, 2))
	})
	if reached != nodes || allocs != 0 {
		t.Fatalf("warmed full traversal: reached=%d want=%d allocations=%g want=0", reached, nodes, allocs)
	}
}
