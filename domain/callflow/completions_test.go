package callflow

import "testing"

func TestCompletionUnionAgainstMapOracle(t *testing.T) {
	choices := []completion{{mode: Returns}, {mode: Panics}, {mode: Returns, values: 1}, {mode: Returns, returned: "tuple"}, {mode: Returns, recovered: true}}
	for mask := 0; mask < 1<<len(choices); mask++ {
		var result completions
		want := map[completion]bool{}
		for index, choice := range choices {
			if mask&(1<<index) == 0 {
				continue
			}
			if !result.add(singleCompletion(choice)) {
				t.Fatal("new completion does not grow set")
			}
			want[choice] = true
		}
		if result.add(result) {
			t.Fatal("union with itself grows set")
		}
		got := map[completion]bool{}
		for value := range result.all() {
			got[value] = true
		}
		if len(got) != len(want) {
			t.Fatalf("mask=%d loses completions", mask)
		}
		for value := range want {
			if !got[value] {
				t.Fatalf("mask=%d loses %+v", mask, value)
			}
		}
		count := 0
		for range result.all() {
			count++
			break
		}
		if count > 1 {
			t.Fatal("iterator ignores early stop")
		}
	}
}

func TestInlineCompletionHasNoAllocation(t *testing.T) {
	count := testing.AllocsPerRun(100, func() {
		result := terminalCompletion(Returns)
		var stored completions
		stored.add(result)
		if stored.outcomes() != Returns {
			t.Fatal("singleton loses outcome")
		}
	})
	if count != 0 {
		t.Fatalf("singleton allocates %g times", count)
	}
}

func TestCompletionZeroRemoval(t *testing.T) {
	result := terminalCompletion(0)
	result.removeZero()
	for range result.all() {
		t.Fatal("zero completion remains")
	}
	result.add(terminalCompletion(Returns))
	result.add(terminalCompletion(0))
	result.removeZero()
	count := 0
	for value := range result.all() {
		count++
		if value.mode != Returns {
			t.Fatal("removal retains wrong completion")
		}
	}
	if count != 1 {
		t.Fatal("removal loses return")
	}
}

func TestAcyclicCompletionSolveNeedsOneTraversal(t *testing.T) {
	body := choiceExitBodies(1, 0)["f0"]
	evaluation := newExitEvaluation(body, ExitSummaries{})
	if got := evaluation.run(false, ""); got != Returns {
		t.Fatalf("outcomes=%v", got)
	}
	if evaluation.cyclic {
		t.Fatal("acyclic flow marked cyclic")
	}
	if !evaluation.changed {
		t.Fatal("acyclic flow repeats a completed traversal")
	}
}

func TestCyclicCompletionSolveReachesFixedPoint(t *testing.T) {
	body := choiceExitBodies(1, 0)["f0"]
	body.flow.Blocks[0].Successors = append(body.flow.Blocks[0].Successors, 1)
	evaluation := newExitEvaluation(body, ExitSummaries{})
	if got := evaluation.run(false, ""); got != Returns|Diverges {
		t.Fatalf("outcomes=%v", got)
	}
	if !evaluation.cyclic {
		t.Fatal("back edge does not mark cycle")
	}
	if evaluation.changed {
		t.Fatal("cyclic solve stops before convergence")
	}
}
