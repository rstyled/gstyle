package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPrivateAllocatedSliceRows(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
// Mirrors the allocated bitset writes in adapters/gopatterns/ir_control_sets.go.
type controlPostSets [][]uint64
func allocated(order []int, size int) controlPostSets {
 sets := make(controlPostSets, size)
 for _, index := range order { sets[index] = make([]uint64, (size+63)/64) }
 for _, target := range order {
  for _, index := range order { sets[index][target/64] |= uint64(1) << (target%64) }
 }
 return sets
}
func Forward(order []int){allocated(order,len(order))}
func Map(){rows:=make(map[int][]int);rows[0]=make([]int,1);rows[0][0]++;rows[0]=append(rows[0],1)}
func Literal(){rows:=make([][]int,1);rows[0]=[]int{1};rows[0][0]=2}
func Borrowed(rows [][]int){rows[0][0]=1}
func Injected(input []int){rows:=make([][]int,1);rows[0]=input;rows[0][0]=1}
func Shadowed(input []int){rows:=make([][]int,1);make:=func(int)[]int{return input};rows[0]=make(1);rows[0][0]=1}
func Copied(input [][]int){rows:=make([][]int,1);rows[0]=make([]int,1);copy(rows,input);rows[0][0]=1}
func Rebound(input [][]int){rows:=make([][]int,1);rows=input;rows[0]=make([]int,1);rows[0][0]=1}
func Alias(){rows:=make([][]int,1);rows[0]=make([]int,1);alias:=rows;rows[0][0]=1;_ =alias}
func RowAlias(){rows:=make([][]int,1);rows[0]=make([]int,1);alias:=rows[0];rows[0][0]=1;_ =alias}
func Address(){rows:=make([][]int,1);rows[0]=make([]int,1);alias:=&rows[0][0];rows[0][0]=1;_ =alias}
var saved [][]int
func Escaped(){rows:=make([][]int,1);rows[0]=make([]int,1);saved=rows;rows[0][0]=1}
func Closure(){rows:=make([][]int,1);rows[0]=make([]int,1);func(){rows[0][0]=1}()}
func Deferred(){rows:=make([][]int,1);rows[0]=make([]int,1);defer func(){rows[0][0]=1}()}
func Unknown(factory func()[]int){rows:=make([][]int,1);rows[0]=factory();rows[0][0]=1}
func Pointee(input *int){rows:=make([][]*int,1);rows[0]=[]*int{input};*rows[0][0]=1}
func Deep(input []int){rows:=make([][][]int,1);rows[0]=[][]int{input};rows[0][0][0]=1}
var counter int
func key()int{counter++;return 0}
func KeyEffect(){rows:=make([][]int,1);rows[0]=make([]int,1);rows[key()][0]=1}
func IO(){rows:=make([][]int,1);rows[0]=make([]int,1);rows[0][0]=1;println("write")}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"allocated", "Forward", "Map", "Literal"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s writes private allocated rows, want no effects: %+v", name, result)
		}
	}
	mutations := findFunction(t, facts, "allocated").Mutations
	if len(mutations) != 2 {
		t.Fatalf("allocated rows lose header or bitset writes: %+v", mutations)
	}
	for _, mutation := range mutations {
		if mutation.Provenance != quality.Local {
			t.Errorf("allocated row write must stay local: %+v", mutation)
		}
	}
	for _, name := range []string{"Borrowed", "Injected", "Rebound", "Shadowed", "Copied", "Alias", "RowAlias", "Address", "Escaped", "Closure", "Deferred", "Unknown", "Pointee", "Deep"} {
		result := resultNamed(t, report, name)
		kind := quality.UnknownEffect
		if name == "Borrowed" {
			kind = quality.MutationEffect
		}
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses %s storage evidence: %+v", name, kind, result)
		}
	}
	for name, kind := range map[string]quality.Kind{"KeyEffect": quality.Global, "IO": quality.IO} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("%s loses independent %s: %+v", name, kind, result)
		}
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
		t.Fatal("saved allocated-row evidence changes evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.allocated", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(resultNamed(t, overridden, "Forward").Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
		t.Fatal("allocated-row proof hides explicit helper policy")
	}
}
