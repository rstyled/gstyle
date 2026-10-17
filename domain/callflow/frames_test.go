package callflow

import (
	"fmt"
	"testing"
)

// The oracle walks reachable functions once. It does not use summary iteration,
// map order or the solver's joins, so cycles cannot hide a seeded observation.
func reachableFrameOracle(graph map[string]frameDescriptor, root string) FrameEffect {
	result := FrameEffect{}
	pending := []string{root}
	seen := map[string]bool{}
	for len(pending) > 0 {
		current := pending[0]
		pending = pending[1:]
		if seen[current] {
			continue
		}
		seen[current] = true
		descriptor := graph[current]
		if descriptor.local.Observes {
			result.Observes = true
		}
		if descriptor.local.Unknown {
			result.Unknown = true
		}
		pending = append(pending, descriptor.targets...)
	}
	return result
}
func TestFrameFixedPointMatchesIndependentReachability(t *testing.T) {
	checked := 0
	for size := 1; size <= 3; size++ {
		for edges := 0; edges < 1<<(size*size); edges++ {
			for seed := 0; seed < size; seed++ {
				for _, unknown := range []bool{false, true} {
					graph := map[string]frameDescriptor{}
					for i := 0; i < size; i++ {
						descriptor := frameDescriptor{local: FrameEffect{Observes: i == seed, Unknown: unknown && i == (seed+1)%size}}
						for j := 0; j < size; j++ {
							if edges&(1<<(i*size+j)) != 0 {
								descriptor.targets = append(descriptor.targets, fmt.Sprint(j))
							}
						}
						graph[fmt.Sprint(i)] = descriptor
					}
					got := solveFrames(graph)
					for root := range graph {
						if want := reachableFrameOracle(graph, root); got[root] != want {
							t.Fatalf("size=%d edges=%d seed=%d root=%s got=%+v want=%+v", size, edges, seed, root, got[root], want)
						}
						checked++
					}
				}
			}
		}
	}
	t.Logf("checked %d small cyclic reachability cases", checked)
}
func TestExternalFrameContractsStayExplicit(t *testing.T) {
	for _, name := range []string{"recover", "panic", "runtime.Caller", "runtime.Callers", "runtime.Stack", "runtime/debug.Stack", "runtime/debug.PrintStack"} {
		if got := targetFrameEffect(name, map[string]FrameEffect{name: {}}); !got.Observes || got.Unknown {
			t.Fatalf("known observation can be replaced by a body: %s %+v", name, got)
		}
	}
	for _, name := range []string{"append", "len", "errors.New", "strings.TrimSpace", "path.Clean"} {
		if got := externalFrameEffect(name); got != (FrameEffect{}) {
			t.Fatalf("reviewed contract=%s %+v", name, got)
		}
	}
	for _, name := range []string{"p.recover", "p.len", "p.append", "fmt.Println", "strings.TrimFunc", "reflect.Value.Call", ""} {
		if got := externalFrameEffect(name); !got.Unknown || got.Observes {
			t.Fatalf("unknown external effect=%s %+v", name, got)
		}
	}
}
