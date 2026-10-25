package goquality

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestRemovedEntriesCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"domain/patterns/alattin.go"})
	if err != nil {
		t.Fatal(err)
	}
	function := findFunction(t, facts, "alternativePostings")
	t.Logf("raw writes: %+v", function.Mutations)
	if len(function.Mutations) != 3 {
		t.Fatalf("lost construction or delete writes: %+v", function.Mutations)
	}
	for _, write := range function.Mutations {
		if write.Root != "co" || write.Provenance != quality.Local {
			t.Errorf("private map removal loses row ownership: %+v", write)
		}
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, report, "alternativePostings")
	t.Logf("result: %+v", result)
	if !result.Complete || len(result.Effects) != 0 {
		t.Errorf("private construction retains false effects: %+v", result)
	}
}

func TestRemovedEntriesBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
func Delete()map[int][]int{out:=map[int][]int{};out[0]=append(out[0],1);delete((out),0);return out}
func Clear()map[int][]int{out:=map[int][]int{};out[0]=append(out[0],1);clear((out));out[1]=append(out[1],2);return out}
func Slice()[][]int{out:=make([][]int,1);out[0]=append(out[0],1);clear(out);out[0]=append(out[0],2);return out}
func Map()map[int]map[int]int{out:=map[int]map[int]int{};out[0]=make(map[int]int);out[0][0]=1;delete(out,0);return out}
func Forward(){Delete()}
func Borrowed(out map[int][]int){out[0]=append(out[0],1);delete(out,0)}
func Seeded(input []int){out:=map[int][]int{0:input};out[0]=append(out[0],1);delete(out,0)}
func Injected(input []int){out:=map[int][]int{};out[0]=input;out[0]=append(out[0],1);clear(out)}
func Rebound(input map[int][]int){out:=map[int][]int{};out=input;out[0]=append(out[0],1);delete(out,0)}
func Alias(input []int){out:=map[int][]int{};alias:=out;alias[0]=input;out[0]=append(out[0],1);clear(out)}
var saved map[int][]int
var row []int
func Publish(){out:=map[int][]int{};saved=out;out[0]=append(out[0],1);delete(out,0)}
func Row(){out:=map[int][]int{};out[0]=append(out[0],1);row=out[0];clear(out)}
func Child(input *int){out:=map[int][]*int{};out[0]=append(out[0],input);*out[0][0]=1;delete(out,0)}
func Callback(run func(map[int][]int)){out:=map[int][]int{};run(out);out[0]=append(out[0],1);clear(out)}
func Captured(){out:=map[int][]int{};_ =func(){saved=out};out[0]=append(out[0],1);delete(out,0)}
func Deferred(){out:=map[int][]int{};defer clear(out);out[0]=append(out[0],1)}
func Async(){out:=map[int][]int{};go clear(out);out[0]=append(out[0],1)}
func Shadowed(){clear:=func(value map[int][]int){saved=value};out:=map[int][]int{};out[0]=append(out[0],1);clear(out)}
func IO(){out:=map[int][]int{};out[0]=append(out[0],1);delete(out,0);println("retained")}
func Unknown(run func()){out:=map[int][]int{};out[0]=append(out[0],1);clear(out);run()}
var count int
func key()int{count++;return 0}
func GlobalKey(){out:=map[int][]int{};out[0]=append(out[0],1);delete(out,key())}
func panicKey()int{panic("retained")}
func PanicKey(){out:=map[int][]int{};out[0]=append(out[0],1);delete(out,panicKey())}
func DynamicKey(key func()int){out:=map[int][]int{};out[0]=append(out[0],1);delete(out,key())}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Delete", "Clear", "Slice", "Map", "Forward"} {
		result := resultNamed(t, report, name)
		t.Logf("%s raw=%+v result=%+v", name, findFunction(t, facts, name).Mutations, result)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s retains false row effects: %+v", name, result)
		}
	}
	for _, name := range []string{"Borrowed", "Seeded", "Injected", "Rebound", "Alias", "Publish", "Row", "Child", "Callback", "Captured", "Deferred", "Async", "Shadowed"} {
		writes := findFunction(t, facts, name).Mutations
		result := resultNamed(t, report, name)
		t.Logf("%s raw=%+v result=%+v", name, writes, result)
		if !slices.ContainsFunc(writes, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) || len(result.Effects) == 0 {
			t.Errorf("%s hides unproved storage: %+v", name, result)
		}
	}
	for name, kind := range map[string]quality.Kind{"IO": quality.IO, "Unknown": quality.UnknownEffect, "GlobalKey": quality.Global, "PanicKey": quality.Panic, "DynamicKey": quality.UnknownEffect} {
		result := resultNamed(t, report, name)
		t.Logf("%s raw=%+v result=%+v", name, findFunction(t, facts, name).Mutations, result)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses %s: %+v", name, kind, result)
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
		t.Fatal("saved removal facts change evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.Delete", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(resultNamed(t, overridden, "Forward").Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("removal proof hides %s helper policy", kind)
		}
	}
}
