package goquality

import (
	"context"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestCollectedSliceStorage(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("iter";"maps";"slices")
func Keys(input map[int]bool){out:=slices.Collect(maps.Keys(input));slices.Sort(out)}
func Values(input map[int]int){out:=slices.Collect[int](maps.Values(input));out[0]=1}
func Sequence(input iter.Seq[int]){out:=slices.Collect(input);slices.Reverse(out)}
func collect(input []int)[]int{return slices.Collect(slices.Values(input))}
func Forward(input []int){out:=collect(input);slices.Sort(out)}
func Borrowed(input []int){out:=input;slices.Sort(out)}
func Rebound(input []int){out:=slices.Collect(slices.Values(input));out=input;slices.Sort(out)}
func Pointee(input []*int){out:=slices.Collect(slices.Values(input));*out[0]=1}
func Nested(input map[int][]int){out:=slices.Collect(maps.Values(input));out[0][0]=1}
func AppendSeq(input []int){out:=slices.AppendSeq(input[:0],slices.Values(input));slices.Sort(out)}
var counter int
func ArgumentEffect(){out:=slices.Collect(argument());slices.Sort(out)}
func argument()iter.Seq[int]{counter++;return slices.Values([]int{1})}
func CallbackEffect(){out:=slices.Collect(func(yield func(int)bool){counter++;yield(1)});slices.Sort(out)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Keys", "Values", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 0 || result.DensityMilli != 0 || !result.Complete {
			t.Errorf("%s writes fresh slice storage, want no effects: %+v", name, result)
		}
	}
	// An arbitrary iterator still returns fresh storage. This assertion makes
	// no claim about the iterator's own effects.
	sequence := findFunction(t, facts, "Sequence")
	for _, call := range sequence.Calls {
		if call.Callee == "slices.Reverse" && call.MutationTarget != quality.Local {
			t.Errorf("collected storage remains shared: %+v", call)
		}
	}
	for _, name := range []string{"Borrowed", "Rebound", "Pointee", "Nested", "AppendSeq"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) == 0 {
			t.Errorf("%s loses shared storage evidence: %+v", name, result)
		}
	}
	for _, name := range []string{"ArgumentEffect", "CallbackEffect"} {
		result := resultNamed(t, report, name)
		found := false
		for _, effect := range result.Effects {
			found = found || effect.Kind == quality.Global
		}
		if !found {
			t.Errorf("%s loses global effect: %+v", name, result)
		}
	}
	for _, callee := range []string{"slices.Collect", "slices.Sort"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: quality.IO})
		overridden, err := quality.Evaluate(facts, config)
		if err != nil {
			t.Fatal(err)
		}
		result := resultNamed(t, overridden, "Keys")
		found := false
		for _, effect := range result.Effects {
			found = found || effect.Kind == quality.IO
		}
		if !found {
			t.Errorf("fresh slice proof hides %s policy override: %+v", callee, result)
		}
	}
}
