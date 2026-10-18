package gopatterns

import "golang.org/x/tools/go/cfg"

// For each candidate z, start at {z} and add predecessors only after every
// successor must reach z. This least fixed point includes maximal infinite
// paths: a cycle avoiding z never enters the set. Mutual postdominance is legal.
// Storage is O(V²) bits; work is O(V(V+E)). No executable exit is invented.
type controlPostSets [][]uint64

func maximalPostDominators(flow *cfg.CFG, dom *executionDominators) controlPostSets {
	sets := make(controlPostSets, len(flow.Blocks))
	for _, index := range dom.order {
		sets[index] = make([]uint64, (len(flow.Blocks)+63)/64)
	}
	work := postDominanceWork{remaining: make([]int, len(flow.Blocks)), seen: make([]bool, len(flow.Blocks)), queue: make([]int, 0, len(flow.Blocks))}
	for _, target := range dom.order {
		must := work.mustReach(flow, dom, target)
		for _, index := range must {
			sets[index][target/64] |= uint64(1) << (target % 64)
		}
	}
	return sets
}

// Reuse per-candidate work storage; only the published bitsets grow with V².
type postDominanceWork struct {
	remaining []int
	seen      []bool
	queue     []int
}

func (work *postDominanceWork) mustReach(flow *cfg.CFG, dom *executionDominators, target int) []int {
	for index, block := range flow.Blocks {
		work.remaining[index] = len(block.Succs)
		work.seen[index] = dom.idom[index] < 0
	}
	work.seen[target] = true
	work.queue = append(work.queue[:0], target)
	for at := 0; at < len(work.queue); at++ {
		for _, previous := range dom.predecessors[work.queue[at]] {
			if work.seen[previous] {
				continue
			}
			work.remaining[previous]--
			if work.remaining[previous] != 0 {
				continue
			}
			work.seen[previous] = true
			work.queue = append(work.queue, previous)
		}
	}
	return work.queue
}

func (sets controlPostSets) contains(from, target int) bool {
	return len(sets[from]) != 0 && sets[from][target/64]&(uint64(1)<<(target%64)) != 0
}
