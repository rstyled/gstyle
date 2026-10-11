package ccgraph

import (
	"context"
	"errors"
	"math"
	"math/big"
	"reflect"
	"slices"
	"testing"
)

func stageFunction(key string) StageFunction {
	references := uint64(0)
	inputs := []Parameter{}
	outputs := []Parameter{{Role: "return", Type: "unit"}}
	return StageFunction{Key: key, Name: "same", Graph: Graph{Nodes: []Node{{"a", Decl}, {"b", Other}}, Edges: []Edge{{"a", "b", Data}}}, References: &references, Inputs: &inputs, Outputs: &outputs, PointerWidth: 64}
}

func TestPairEnumerationAndTrace(t *testing.T) {
	functions := []StageFunction{stageFunction("c"), stageFunction("a"), stageFunction("b")}
	var got []Pair
	run, err := Evaluate(context.Background(), functions, DefaultConfig(), func(p Pair) error { got = append(got, p); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !run.Complete || run.Pairs != 3 || run.Profile != Profile || run.ContractCommit != ContractCommit {
		t.Fatalf("run=%+v", run)
	}
	var keys [][2]string
	for _, p := range got {
		keys = append(keys, [2]string{p.Left, p.Right})
		if !p.Accepted || p.Final != Passed || p.Kernel.Depth != 1 || p.StringScore.Mean.Cmp(big.NewRat(1, 1)) != 0 {
			t.Fatalf("pair=%+v", p)
		}
	}
	if !reflect.DeepEqual(keys, [][2]string{{"a", "b"}, {"a", "c"}, {"b", "c"}}) {
		t.Fatal(keys)
	}
}

func TestPairGatesShortCircuit(t *testing.T) {
	cases := []struct {
		name    string
		edit    func(*StageFunction)
		config  Config
		stopped Gate
		next    func(Pair) Gate
	}{
		{"empty", func(f *StageFunction) { f.Graph = Graph{} }, DefaultConfig(), Rejected, func(p Pair) Gate { return p.Size }},
		{"ratio", func(f *StageFunction) { f.Graph.Nodes = append(f.Graph.Nodes, Node{"c", Other}) }, DefaultConfig(), Rejected, func(p Pair) Gate { return p.Numerical }},
		{"numerical", func(f *StageFunction) { f.Graph.Nodes[0].Label = Assign; f.Graph.Edges[0].Family = Ctrl }, DefaultConfig(), Rejected, func(p Pair) Gate { return p.Strings }},
		{"strings", func(f *StageFunction) { f.Name = "different" }, Config{"0.8", "0.9", "1", "0.9", Overlap}, Rejected, func(p Pair) Gate { return p.Final }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b := stageFunction("a"), stageFunction("b")
			tc.edit(&b)
			var got Pair
			_, err := Evaluate(context.Background(), []StageFunction{a, b}, tc.config, func(p Pair) error { got = p; return nil })
			if err != nil {
				t.Fatal(err)
			}
			if got.Accepted || tc.next(got) != NotEvaluated || got.Kernel != nil {
				t.Fatalf("later stage ran: %+v", got)
			}
		})
	}
}

func TestRequiredUnknownFactsAreIneligible(t *testing.T) {
	cases := []struct {
		name string
		edit func(*StageFunction)
	}{
		{"references", func(f *StageFunction) { f.References = nil }},
		{"inputs", func(f *StageFunction) { f.Inputs = nil }},
		{"outputs", func(f *StageFunction) { f.Outputs = nil }},
		{"name", func(f *StageFunction) { f.Name = "" }},
		{"type", func(f *StageFunction) { (*f.Outputs)[0].Type = "unknown" }},
		{"target", func(f *StageFunction) { f.UsesNativeWidth = true; f.PointerWidth = 0 }},
		{"dangling", func(f *StageFunction) { f.Graph.Edges[0].From = "missing" }},
		{"duplicate node", func(f *StageFunction) { f.Graph.Nodes[1].ID = "a" }},
		{"label", func(f *StageFunction) { f.Graph.Nodes[0].Label = "unknown" }},
		{"family", func(f *StageFunction) { f.Graph.Edges[0].Family = "unknown" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := stageFunction("invalid")
			tc.edit(&f)
			run, err := Evaluate(context.Background(), []StageFunction{f, stageFunction("valid")}, DefaultConfig(), func(Pair) error { t.Fatal("ineligible pair emitted"); return nil })
			if err != nil || !run.Complete || len(run.Ineligible) != 1 {
				t.Fatalf("run=%+v err=%v", run, err)
			}
		})
	}
}

func TestPairFailureIsIncomplete(t *testing.T) {
	stop := errors.New("sink stopped")
	run, err := Evaluate(context.Background(), []StageFunction{stageFunction("a"), stageFunction("b")}, DefaultConfig(), func(Pair) error { return stop })
	if !errors.Is(err, stop) || run.Complete {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	run, err = Evaluate(ctx, []StageFunction{stageFunction("a"), stageFunction("b")}, DefaultConfig(), func(Pair) error { return nil })
	if !errors.Is(err, context.Canceled) || run.Complete {
		t.Fatalf("run=%+v err=%v", run, err)
	}
}

func TestDuplicateFunctionKeysFailBeforeEmission(t *testing.T) {
	_, err := Evaluate(context.Background(), []StageFunction{stageFunction("same"), stageFunction("same")}, DefaultConfig(), func(Pair) error { t.Fatal("emitted"); return nil })
	if err == nil {
		t.Fatal("duplicate key admitted")
	}
}

func TestNativeTargetMismatchIsPairIneligible(t *testing.T) {
	a, b := stageFunction("a"), stageFunction("b")
	a.UsesNativeWidth = true
	b.PointerWidth = 32
	var got Pair
	_, err := Evaluate(context.Background(), []StageFunction{a, b}, DefaultConfig(), func(p Pair) error { got = p; return nil })
	if err != nil || got.Diagnostic == "" || got.Nonempty != NotEvaluated {
		t.Fatalf("pair=%+v err=%v", got, err)
	}
}

func TestDefaultAndExperimentalScoring(t *testing.T) {
	a, b := stageFunction("a"), stageFunction("b")
	a.Graph = Graph{Nodes: []Node{{"a", Other}, {"b", Other}, {"c", Decl}, {"d", Decl}}}
	b.Graph = Graph{Nodes: append(slices.Clone(a.Graph.Nodes), Node{"e", Decl})}
	for _, scoring := range []string{Overlap, CrossCosine} {
		c := DefaultConfig()
		c.Scoring = scoring
		c.Final = "1"
		var got Pair
		_, err := Evaluate(context.Background(), []StageFunction{a, b}, c, func(p Pair) error { got = p; return nil })
		if err != nil || got.Accepted != (scoring == Overlap) {
			t.Fatalf("%s: %+v %v", scoring, got, err)
		}
	}
}

func TestNoDepthCapAndPermutationInvariance(t *testing.T) {
	g := Graph{}
	for i := 0; i < 10; i++ {
		id := string(rune('a' + i))
		g.Nodes = append(g.Nodes, Node{id, Other})
		if i > 0 {
			g.Edges = append(g.Edges, Edge{string(rune('a' + i - 1)), id, Execution})
		}
	}
	permuted := Graph{Nodes: slices.Clone(g.Nodes), Edges: slices.Clone(g.Edges)}
	slices.Reverse(permuted.Nodes)
	slices.Reverse(permuted.Edges)
	k, err := CompareWL(context.Background(), g, permuted)
	if err != nil || k.Depth != 9 || len(k.Rounds) != 10 || k.Overlap.Cmp(big.NewRat(1, 1)) != 0 {
		t.Fatalf("kernel=%+v err=%v", k, err)
	}
}

func TestNumericalArithmeticDoesNotOverflow(t *testing.T) {
	a := [9]uint64{math.MaxUint64, math.MaxUint64, 1}
	if got := Numerical(a, a); !got.AtLeast("1") || got.Dot.BitLen() <= 64 {
		t.Fatalf("overflow: %+v", got)
	}
}

func TestConfigurationRejectsInvalidOverrides(t *testing.T) {
	for _, data := range []string{`{"unknown":1}`, `{"final_threshold":"NaN"}`, `{"final_threshold":"1.01"}`, `{"final_threshold":"-0.1"}`, `{"scoring":"legacy"}`, `{} {}`, `null`, `[]`, `{"final_threshold":null}`} {
		if _, err := ParseConfig([]byte(data)); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	c, err := ParseConfig([]byte(`{"final_threshold":"0.90000000000000000000000001"}`))
	if err != nil || atLeast(big.NewRat(9, 10), c.Final) {
		t.Fatalf("rounded threshold: %+v %v", c, err)
	}
}

func TestProjectionRejectsUnknownReachabilityAndBindings(t *testing.T) {
	for _, n := range []Operation{{ID: "x", Category: "assignment"}, {ID: "x", Category: "entry"}, {ID: "x", Category: "memory_summary"}} {
		if _, err := Project(ProjectionInput{Nodes: []Operation{n}}); err == nil {
			t.Fatalf("accepted %+v", n)
		}
	}
	reachable := true
	if _, err := Project(ProjectionInput{Nodes: []Operation{{ID: "x", Category: "read", Reachable: &reachable}}}); err == nil {
		t.Fatal("unknown read binding admitted")
	}
}
