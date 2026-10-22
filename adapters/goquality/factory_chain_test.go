package goquality

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestComposedContainerFactories(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "slices"
func wrap(ds []int, enabled bool) []int {
 ds = appendRule(ds, enabled)
 ds = appendRule(ds, enabled)
 return ds
}
func appendRule(ds []int, enabled bool) []int {
 if !enabled { return ds }
 return append(ds, 1)
}
func Local(enabled bool) []int {
 ds := []int{}
 ds = wrap(ds, enabled)
 ds = append(ds, 2)
 return ds
}
func Borrowed(ds []int) []int { ds = wrap(ds, true); ds = append(ds, 2); return ds }
var shared []int
func Mixed(ds []int, useShared bool) []int {
 if useShared { return shared }
 return wrap(ds, true)
}
func MixedCaller() { ds := Mixed([]int{}, true); ds = append(ds, 2) }
func Fresh() { ds := freshWrapper(); slices.Reverse(ds) }
func freshWrapper() []int { return freshLeaf() }
func freshLeaf() []int { return []int{1,2} }
func Effect() { ds := effectWrapper(nil); slices.Reverse(ds) }
func effectWrapper(ds []int) []int { return effectLeaf(ds) }
func effectLeaf(ds []int) []int { println("effect"); return append(ds, 1) }
func Published() { ds := publishedWrapper(nil); slices.Reverse(ds) }
func publishedWrapper(ds []int) []int { return publishedLeaf(ds) }
func publishedLeaf(ds []int) []int { shared = ds; return append(ds, 1) }
func Aliased() { ds := aliasedWrapper(nil); slices.Reverse(ds) }
func aliasedWrapper(ds []int) []int { alias := ds; shared = alias; return wrap(ds, true) }
var slot *[]int
func Addressed() { ds := addressedWrapper(nil); slices.Reverse(ds) }
func addressedWrapper(ds []int) []int { slot = &ds; return wrap(ds, true) }
func Unknown(get func([]int)[]int) { ds := unknownWrapper(nil, get); slices.Reverse(ds) }
func unknownWrapper(ds []int, get func([]int)[]int) []int { return get(ds) }
func Mutual() { ds := mutualA(nil, true); slices.Reverse(ds) }
func mutualA(ds []int, stop bool) []int { if stop { return shared }; return mutualB(ds) }
func mutualB(ds []int) []int { return mutualA(ds, true) }
func Elements(input *int) { ds := elementWrapper(input); *ds[0] = 1 }
func elementWrapper(input *int) []*int { return elementLeaf(input) }
func elementLeaf(input *int) []*int { return []*int{input} }
func OtherInput(other []int) { ds := otherWrapper(nil, other, true); slices.Reverse(ds) }
func otherWrapper(ds, other []int, flag bool) []int { return otherLeaf(ds, other, flag) }
func otherLeaf(ds, other []int, flag bool) []int { if flag { return other }; return ds }
func Callback(run func()) { ds := callbackWrapper(nil, run); slices.Reverse(ds) }
func callbackWrapper(ds []int, run func()) []int { run(); return wrap(ds, true) }

`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Fresh"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 0 || !result.Complete {
			t.Errorf("%s writes only owned storage through composed helpers: %+v", name, result)
		}
	}
	borrowed := resultNamed(t, report, "Borrowed")
	if !slices.ContainsFunc(borrowed.Effects, func(e quality.Effect) bool { return e.Kind == quality.MutationEffect }) {
		t.Errorf("Borrowed loses parameter mutation: %+v", borrowed)
	}
	for _, name := range []string{"Published", "Aliased", "Addressed", "Unknown", "Mutual", "OtherInput"} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.MutationEffect && e.Detail == "slices.Reverse" }) {
			t.Errorf("%s loses the caller's unproved storage mutation: %+v", name, result)
		}
	}
	for _, name := range []string{"MixedCaller", "Elements"} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.UnknownEffect && e.Detail == "ds" }) {
			t.Errorf("%s loses unproved child or mixed storage mutation: %+v", name, result)
		}
	}
	callback := resultNamed(t, report, "Callback")
	if !slices.ContainsFunc(callback.Effects, func(e quality.Effect) bool {
		return e.Kind == quality.UnknownEffect && strings.HasSuffix(e.Detail, " → run")
	}) {
		t.Errorf("composed ownership loses dynamic callback: %+v", callback)
	}

	effect := resultNamed(t, report, "Effect")
	if len(effect.Effects) != 1 || effect.Effects[0].Kind != quality.IO || !effect.Complete {
		t.Errorf("composed ownership loses helper IO: %+v", effect)
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
		t.Fatal("composed storage facts change after replay", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.wrap", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		result := resultNamed(t, overridden, "Local")
		if kind == quality.None {
			if len(result.Effects) != 0 {
				t.Errorf("pure override lost: %+v", result)
			}
			continue
		}
		if len(result.Effects) != 1 || !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s override lost: %+v", kind, result)
		}
	}
}

func TestComposedContainerFactoryDepthLimit(t *testing.T) {
	var source strings.Builder
	source.WriteString("package repro\nimport \"slices\"\nfunc Caller(){out:=chain0();slices.Reverse(out)}\n")
	const depth = 10
	for index := range depth {
		fmt.Fprintf(&source, "func chain%d()[]int{return chain%d()}\n", index, index+1)
	}
	fmt.Fprintf(&source, "func chain%d()[]int{return []int{1,2}}\n", depth)
	root := qualitySourceModule(t, []byte(source.String()))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	caller := resultNamed(t, report, "Caller")
	if !slices.ContainsFunc(caller.Effects, func(e quality.Effect) bool { return e.Kind == quality.MutationEffect }) {
		t.Errorf("chain beyond proof limit must retain storage uncertainty: %+v", caller)
	}
}
