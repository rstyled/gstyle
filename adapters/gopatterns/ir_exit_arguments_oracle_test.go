package gopatterns

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/callflow"
)

// Each concrete state either returns, panics, or calls the next function with
// false/true. A direct walk detects cycles independently of summary solving.
func concreteArgumentExit(actions []int, function, flag int) callflow.Outcomes {
	seen := map[int]bool{}
	for {
		state := function*2 + flag
		if seen[state] {
			return callflow.Diverges
		}
		seen[state] = true
		action := actions[state]
		if action == 0 {
			return callflow.Returns
		}
		if action == 1 {
			return callflow.Panics
		}
		function = (function + 1) % (len(actions) / 2)
		flag = action - 2
	}
}
func TestExitArgumentsAgainstConcreteStateOracle(t *testing.T) {
	observations := 0
	for count := 1; count <= 2; count++ {
		for code := 0; code < 1<<(4*count); code++ {
			actions := make([]int, 2*count)
			for i := range actions {
				actions[i] = (code >> (2 * i)) & 3
			}
			var source strings.Builder
			for function := range count {
				fmt.Fprintf(&source, "func fn%d(flag bool){if flag{%s}else{%s}};", function, argumentOracleAction(actions[function*2+1], (function+1)%count), argumentOracleAction(actions[function*2], (function+1)%count))
				for flag := range 2 {
					fmt.Fprintf(&source, "func probe%d_%d(){fn%d(%t)};", function, flag, function, flag == 1)
				}
			}
			graphs, name := argumentExitGraphs(t, source.String())
			summaries, err := callflow.ExitEffects(graphs)
			if err != nil {
				t.Fatal(err)
			}
			prefix := strings.TrimSuffix(name, ".f")
			for function := range count {
				for flag := range 2 {
					got := summaries[fmt.Sprintf("%s.probe%d_%d", prefix, function, flag)]
					want := concreteArgumentExit(actions, function, flag)
					if got.Ordinary != want || got.DeferredPanic != want {
						t.Fatalf("actions=%v start=%d/%d got=%v/%v want=%v", actions, function, flag, got.Ordinary, got.DeferredPanic, want)
					}
					observations++
				}
			}
		}
	}
	if observations != 1056 {
		t.Fatalf("observations=%d", observations)
	}
}
func argumentOracleAction(action, next int) string {
	switch action {
	case 0:
		return "return"
	case 1:
		return `panic("stop")`
	default:
		return fmt.Sprintf("fn%d(%t)", next, action == 3)
	}
}
