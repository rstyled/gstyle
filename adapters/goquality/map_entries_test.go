package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPrivateMapEntryNilComparison(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
func Equal(input []int) map[int][]int {
	out:=make(map[int][]int)
	for _,value:=range input { if out[value]==nil { };out[value]=append(out[value],value) }
	return out
}
func Reverse(input []int) map[int][]int {
	out:=map[int][]int{}
	for _,value:=range input { if nil==(out[value]) { };out[value]=append(out[value],value) }
	return out
}
func Unequal(input []int) map[int][]int {
	out:=map[int][]int{}
	for _,value:=range input { if (out[value])!=(nil) { };out[value]=append(out[value],value) }
	return out
}
type Values []int
func ReverseUnequal() map[int]Values { out:=map[int]Values{};if nil!=out[0] { };out[0]=append(out[0],1);return out }
func Forward(input []int) { Equal(input) }
func Borrowed(out map[int][]int) { if out[0]==nil { };out[0]=append(out[0],1) }
func Seeded(input []int) { out:=map[int][]int{0:input};if out[0]!=nil { };out[0]=append(out[0],1) }
func Rebound(input map[int][]int) { out:=map[int][]int{};out=input;if out[0]==nil { };out[0]=append(out[0],1) }
var saved []int
func Escaped() { out:=map[int][]int{};out[0]=append(out[0],1);saved=out[0];if out[0]!=nil { };out[0]=append(out[0],2) }
func Pointee(input *int) { out:=map[int][]*int{};if out[0]==nil { };out[0]=append(out[0],input);*out[0][0]=1 }
var counter int
func key()int { counter++;return 0 }
func KeyEffect() { out:=map[int][]int{};if out[key()]==nil { };out[0]=append(out[0],1) }
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Equal", "Reverse", "Unequal", "ReverseUnequal", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 0 || result.DensityMilli != 0 || !result.Complete {
			t.Errorf("%s only observes nil before local appends, want no effects: %+v", name, result)
		}
	}
	for _, name := range []string{"Borrowed", "Seeded", "Rebound", "Escaped", "Pointee"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) == 0 || result.DensityMilli == 0 {
			t.Errorf("%s must retain shared-storage effects: %+v", name, result)
		}
	}
	key := resultNamed(t, report, "KeyEffect")
	if len(key.Effects) != 1 || key.Effects[0].Kind != quality.Global {
		t.Errorf("nil observation must retain key evaluation effects: %+v", key)
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
		t.Fatal("saved map-entry proof changes evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.Equal", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	forward := resultNamed(t, overridden, "Forward")
	if len(forward.Effects) != 1 || forward.Effects[0].Kind != quality.IO {
		t.Fatalf("map-entry proof hides helper policy override: %+v", forward)
	}
}
