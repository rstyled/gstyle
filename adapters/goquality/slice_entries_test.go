package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPrivateSliceEntryAppends(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type Block struct { Index int; Succs []*Block }
type Graph struct { Blocks []*Block }
// Same construction as adapters/gopatterns/ir_dominators.go:executionPredecessors.
func executionPredecessors(graph *Graph) [][]int {
	predecessors := make([][]int, len(graph.Blocks))
	for _, block := range graph.Blocks {
		for _, next := range block.Succs {
			predecessors[next.Index] = append(predecessors[next.Index], int(block.Index))
		}
	}
	return predecessors
}
type Row []int
type Rows []Row
func Named() Rows { out:=make(Rows,2);out[0]=append(out[1],1);return out }
func Capacity() [][]int { out:=make([][]int,2,4);out[0]=append(out[0],1);return out }
func Forward(graph *Graph) { executionPredecessors(graph) }
func Borrowed(out [][]int) { out[0]=append(out[0],1) }
func Seeded(input []int) { out:=[][]int{input};out[0]=append(out[0],1) }
func Injected(input []int) { out:=make([][]int,1);out[0]=input;out[0]=append(out[0],1) }
func Rebound(input [][]int) { out:=make([][]int,1);out=input;out[0]=append(out[0],1) }
func Alias() { out:=make([][]int,1);alias:=out;out[0]=append(out[0],1);_ =alias }
func Exposed() { out:=make([][]int,1);row:=out[0];out[0]=append(out[0],1);_ =row }
func Pointee(input *int) { out:=make([][]*int,1);out[0]=append(out[0],input);*out[0][0]=1 }
func Closure() { out:=make([][]int,1);func(){out[0]=append(out[0],1)}() }
func Deferred() { out:=make([][]int,1);defer func(){out[0]=append(out[0],1)}() }
var saved [][]int
func Escaped() { out:=make([][]int,1);saved=out;out[0]=append(out[0],1) }
var counter int
func key()int { counter++;return 0 }
func KeyEffect() { out:=make([][]int,1);out[key()]=append(out[0],1) }
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"executionPredecessors", "Named", "Capacity", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 0 || result.DensityMilli != 0 || !result.Complete {
			t.Errorf("%s appends to private rows, want no effects: %+v", name, result)
		}
	}
	function := findFunction(t, facts, "executionPredecessors")
	if len(function.Mutations) != 2 {
		t.Fatalf("entry proof loses assignment or append evidence: %+v", function.Mutations)
	}
	for _, mutation := range function.Mutations {
		if mutation.Provenance != quality.Local {
			t.Errorf("private row mutation must retain local provenance: %+v", mutation)
		}
	}
	for _, name := range []string{"Borrowed", "Seeded", "Injected", "Rebound", "Alias", "Exposed", "Pointee", "Closure", "Deferred", "Escaped"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) == 0 || result.DensityMilli == 0 {
			t.Errorf("%s must retain shared-storage uncertainty: %+v", name, result)
		}
	}
	key := resultNamed(t, report, "KeyEffect")
	if len(key.Effects) != 1 || key.Effects[0].Kind != quality.Global {
		t.Errorf("entry proof must retain index evaluation effects: %+v", key)
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("saved slice-entry proof changes evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.executionPredecessors", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	forward := resultNamed(t, overridden, "Forward")
	if len(forward.Effects) != 1 || forward.Effects[0].Kind != quality.IO {
		t.Fatalf("slice-entry proof hides helper policy override: %+v", forward)
	}
}
