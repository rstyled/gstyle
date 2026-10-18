package callflow

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

// Reachability to an external writer is an independent graph property. Enumerate
// every three-body call graph, including cycles, and every external-write set.
func TestMemoryPreservationAgainstWriterReachabilityOracle(t *testing.T) {
	for edges := 0; edges < 512; edges++ {
		for writers := 0; writers < 8; writers++ {
			bodies := map[string]*exitBody{}
			summaries := ExitSummaries{}
			for source := 0; source < 3; source++ {
				name := fmt.Sprint("f", source)
				graph := &pdg.Graph{}
				calls := &CallSites{targets: map[pdg.Ref]callTargets{}}
				block := pdg.FlowBlock{Reachable: true}
				var targets []string
				for target := 0; target < 3; target++ {
					if edges&(1<<(source*3+target)) != 0 {
						targets = append(targets, fmt.Sprint("f", target))
					}
				}
				if writers&(1<<source) != 0 {
					targets = append(targets, "external.writer")
				}
				for _, target := range targets {
					graph.Nodes = append(graph.Nodes, pdg.Node{Category: pdg.Call})
					ref := pdg.Ref(len(graph.Nodes))
					block.Operations = append(block.Operations, ref)
					calls.targets[ref] = callTargets{known: true, names: []string{target}}
				}
				bodies[name] = &exitBody{graph: graph, calls: calls, flow: &pdg.NormalFlow{Blocks: []pdg.FlowBlock{block}}}
				summaries[name] = ExitSummary{PreservesCallerMemory: true}
			}
			solveMemoryPreservation(bodies, summaries)
			for source := 0; source < 3; source++ {
				seen := map[int]bool{}
				queue := []int{source}
				want := true
				for len(queue) > 0 {
					current := queue[0]
					queue = queue[1:]
					if seen[current] {
						continue
					}
					seen[current] = true
					if writers&(1<<current) != 0 {
						want = false
					}
					for target := 0; target < 3; target++ {
						if edges&(1<<(current*3+target)) != 0 {
							queue = append(queue, target)
						}
					}
				}
				if got := summaries[fmt.Sprint("f", source)].PreservesCallerMemory; got != want {
					t.Fatalf("edges=%09b writers=%03b source=%d got %v want %v", edges, writers, source, got, want)
				}
			}
		}
	}
}

func TestMemoryPreservationDoesNotTrustAbsentEvidence(t *testing.T) {
	if memoryBodyKnown(nil) {
		t.Fatal("nil body has a memory proof")
	}
	if memoryBodyKnown(&exitBody{}) {
		t.Fatal("empty body has a memory proof")
	}
	calls := &CallSites{targets: map[pdg.Ref]callTargets{
		1: {known: true, names: []string{"example.noop"}},
		2: {known: true, names: []string{"len"}},
		3: {known: true, names: []string{"example.len"}},
		4: {known: true, names: []string{"example.noop", "example.writer"}},
	}}
	summaries := ExitSummaries{"example.noop": {PreservesCallerMemory: true}, "example.writer": {}}
	for ref, want := range map[pdg.Ref]bool{0: false, 1: true, 2: true, 3: false, 4: false} {
		if got := calls.PreservesCallerMemory(summaries, ref); got != want {
			t.Fatalf("ref %v preserves=%v want %v", ref, got, want)
		}
	}
}
