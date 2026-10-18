package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
	"testing"
)

// The oracle uses a concrete Boolean and two independent pending-exit bits.
// It does not call the production continuation resolver or state solver.
func TestUnwindValuesAgainstExplicitStackOracle(t *testing.T) {
	var source strings.Builder
	source.WriteString(`package example;import "runtime";func value(e0,a0,b0,e1,a1,b1,e2,a2,b2,initial,pendingPanic,pendingGoexit bool)(flag bool){flag=initial;`)
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&source, `if e%d{if a%d{if b%d{defer func(){recover()}()}else{defer func(){panic("child")}()}}else{if b%d{defer func(){flag=true}()}else{defer func(){flag=!flag}()}}};`, i, i, i, i)
	}
	source.WriteString(`if pendingPanic{panic("base")};if pendingGoexit{runtime.Goexit()};return flag};`)
	want := map[string]callflow.Outcomes{}
	index := 0
	for size := 0; size <= 3; size++ {
		choices := 1
		for range size {
			choices *= 4
		}
		for code := 0; code < choices; code++ {
			for _, initial := range []bool{false, true} {
				for pending := 0; pending < 3; pending++ {
					var words []int
					remaining := code
					args := []string{}
					for slot := 0; slot < 3; slot++ {
						op := remaining % 4
						remaining /= 4
						enabled := slot < size
						if enabled {
							words = append(words, op)
						}
						args = append(args, fmt.Sprint(enabled), fmt.Sprint(enabled && op&2 != 0), fmt.Sprint(enabled && op&1 != 0))
					}
					args = append(args, fmt.Sprint(initial), fmt.Sprint(pending == 1), fmt.Sprint(pending == 2))
					name := fmt.Sprintf("f%d", index)
					index++
					fmt.Fprintf(&source, "func %s(){if value(%s){panic(\"returned true\")}};", name, strings.Join(args, ","))
					flag, panicking, goexit := initial, pending == 1, pending == 2
					for i := len(words) - 1; i >= 0; i-- {
						switch words[i] {
						case 0:
							flag = !flag
						case 1:
							flag = true
						case 2:
							panicking = true
						case 3:
							panicking = false
						}
					}
					outcome := callflow.Returns
					if goexit {
						outcome = callflow.ExitsGoroutine
					}
					if panicking {
						outcome = callflow.Panics
						if goexit {
							outcome = callflow.PanicDuringGoexit
						}
					}
					if !panicking && !goexit && flag {
						outcome = callflow.Panics
					}
					want["example."+name] = outcome
				}
			}
		}
	}
	extraction, _ := proofExtraction(t, source.String())
	graphs := map[string]*pdg.Graph{}
	for _, graph := range extraction.Callables {
		graphs[graph.Tables.Text(graph.Function.ID)] = graph
	}
	summaries, err := callflow.ExitEffects(graphs)
	if err != nil {
		t.Fatal(err)
	}
	for name, expected := range want {
		if got := summaries[name].Ordinary; got != expected {
			t.Errorf("%s: got %v want %v", name, got, expected)
		}
	}
	if len(want) != 510 {
		t.Fatalf("oracle covers %d traces", len(want))
	}
}
