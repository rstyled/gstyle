package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestNilCloneReplacesParameterStorage(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "sort"
type Record struct{Rank int; Child *int}
// Mirrors application.rankedFunctions: the comparator reads the copied slice.
func Ranked(values []Record) []Record {
 values=append([]Record(nil),values...)
 sort.SliceStable(values,func(i,j int)bool{return values[i].Rank<values[j].Rank})
 return values
}
func Forward(values []Record)[]Record{return Ranked(values)}
func Write(values []int){values=append([]int(nil),values...);values[0]=1}
func Empty(values []int){values=append([]int(nil));sort.Ints(values)}
func Named(values []int){type Numbers []int;values=append(Numbers(nil),values...);values[0]=1}
func Before(values []int){values[0]=1;values=append([]int(nil),values...);values[0]=2}
func Shared(values []int){values[0]=1}
func Borrowed(values []int){values=append(values[:0],values...);values[0]=1}
func Spare(values []int){values=append(values[:0:cap(values)],values...);values[0]=1}
func Mixed(values []int,flag bool){if flag{values=append([]int(nil),values...)};values[0]=1}
func Rebound(values,other []int){values=append([]int(nil),values...);values=other;values[0]=1}
func Captured(values,other []int){values=append([]int(nil),values...);f:=func(){values=other};f();values[0]=1}
func Address(values,other []int){values=append([]int(nil),values...);p:=&values;*p=other;values[0]=1}
func Loop(values []int){for i:=0;i<2;i++{values[0]=1;values=append([]int(nil),values...)}}
func Child(values []Record){values=append([]Record(nil),values...);*values[0].Child=1}
func Dynamic(values []int,callback func()){values=append([]int(nil),values...);values[0]=1;callback()}
func IO(values []int){values=append([]int(nil),values...);values[0]=1;println("write")}
func CallbackIO(values []Record){values=append([]Record(nil),values...);sort.SliceStable(values,func(i,j int)bool{println("compare");return values[i].Rank<values[j].Rank})}
func CallbackWrite(values []Record,input *int){values=append([]Record(nil),values...);sort.SliceStable(values,func(i,j int)bool{*input=1;return values[i].Rank<values[j].Rank})}
func CallbackRebind(values,other []Record){values=append([]Record(nil),values...);sort.SliceStable(values,func(i,j int)bool{values=other;values[i].Rank=1;return false})}
func ShadowNil(values []int){nil:=values;values=append([]int(nil),values...);values[0]=1}
func ShadowAppend(values []int){append:=func(dst,src []int)[]int{return src};values=append([]int(nil),values);values[0]=1}
func Labeled(values []int){start:values=append([]int(nil),values...);values[0]=1;if len(values)==0{goto start}}
func Unknown(values []int,factory func()[]int){values=append(factory(),values...);values[0]=1}
func Deferred(values,other []int){values=append([]int(nil),values...);defer func(){values=other;values[0]=1}()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Ranked", "Forward", "Write", "Empty", "Named"} {
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v", name, result)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses nil-cloned outer storage: %+v", name, result)
		}
	}
	before := findFunction(t, facts, "Before")
	if before.Mutations[0].Provenance != quality.External || before.Mutations[len(before.Mutations)-1].Provenance != quality.Local {
		t.Errorf("clone loses the shared-to-private boundary: %+v", before.Mutations)
	}
	ranked := findFunction(t, facts, "Ranked")
	if len(ranked.Mutations) != 2 || ranked.Calls[len(ranked.Calls)-1].MutationTarget != quality.Local {
		t.Errorf("clone loses raw mutations or sort ownership: %+v", ranked)
	}
	for _, name := range []string{"Shared", "Borrowed", "Spare", "Mixed", "Rebound", "Captured", "Address", "Loop", "Child", "CallbackRebind", "ShadowNil", "ShadowAppend", "Labeled", "Unknown", "Deferred"} {
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v", name, result)
		mutations := findFunction(t, facts, name).Mutations
		write := mutations[len(mutations)-1]
		if name == "Loop" {
			write = mutations[1] // The element write precedes the cloning append.
		}
		if write.Provenance == quality.Local {
			t.Errorf("%s hides its shared or unknown element write: %+v", name, mutations)
		}
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool {
			return e.Kind == quality.MutationEffect || e.Kind == quality.UnknownEffect
		}) {
			t.Errorf("%s loses shared or unknown storage: %+v", name, result)
		}
	}
	for name, kind := range map[string]quality.Kind{"Dynamic": quality.UnknownEffect, "IO": quality.IO, "CallbackIO": quality.IO, "CallbackWrite": quality.MutationEffect} {
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v", name, result)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
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
		t.Fatal("saved nil-clone facts change evaluation", err)
	}
	for _, path := range []string{"sort.SliceStable", "example.com/quality-repro.Ranked"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: path, Kind: quality.IO})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(resultNamed(t, overridden, "Forward").Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
			t.Errorf("clone hides explicit %s policy", path)
		}
	}
}
