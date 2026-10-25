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

func TestNilEntriesCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"adapters/govalidation/dependencies.go"})
	if err != nil {
		t.Fatal(err)
	}
	function := findFunction(t, facts, "reverseFlow")
	if len(function.Mutations) == 0 {
		t.Fatal("lost raw construction writes")
	}
	for _, write := range function.Mutations {
		if write.Root == "out" && write.Provenance != quality.Local {
			t.Errorf("private nil-seeded map retains false external uncertainty: %+v", write)
		}
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if result := resultNamed(t, report, "reverseFlow"); !result.Complete || len(result.Effects) != 0 {
		t.Errorf("private construction retains false effects: %+v", result)
	}
}

func TestNilEntriesBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
func NilSeed()map[int][]int{out:=map[int][]int{0:nil,1:(nil)};out[0]=append(out[0],1);return out}
func NilReset()map[int][]int{out:=map[int][]int{};out[0]=nil;out[0]=append(out[0],1);return out}
func Discard()map[int][]int{out:=map[int][]int{};if _,ok:=out[0];!ok{out[0]=nil};out[0]=append(out[0],1);return out}
func NilSlice()[][]int{out:=[][]int{nil,1:nil};out[0]=append(out[0],1);return out}
func NilMap()map[int]map[int]int{out:=map[int]map[int]int{0:nil};out[0]=make(map[int]int);out[0][0]=1;return out}
func Forward(){NilSeed()}
func Borrowed(out map[int][]int){out[0]=nil;out[0]=append(out[0],1)}
func Seeded(input []int){out:=map[int][]int{0:input};out[0]=append(out[0],1)}
func Mixed(input []int){out:=map[int][]int{0:nil,1:input};out[1]=append(out[1],1)}
func Shadowed(input []int){nil:=input;out:=map[int][]int{0:nil};out[0]=append(out[0],1)}
func Alias(input []int){out:=map[int][]int{0:nil};alias:=out;alias[0]=input;out[0]=append(out[0],1)}
func Rebound(input map[int][]int){out:=map[int][]int{0:nil};out=input;out[0]=append(out[0],1)}
var saved map[int][]int
var row []int
func Publish(){out:=map[int][]int{0:nil};saved=out;out[0]=append(out[0],1)}
func RetainRow(){out:=map[int][]int{0:nil};row, _ = out[0];out[0]=append(out[0],1)}
func Child(input *int){out:=map[int][]*int{0:nil};out[0]=append(out[0],input);*out[0][0]=1}
func Callback(run func(map[int][]int)){out:=map[int][]int{0:nil};run(out);out[0]=append(out[0],1)}
func Unknown(run func()){out:=map[int][]int{0:nil};out[0]=append(out[0],1);run()}
func IO(){out:=map[int][]int{0:nil};out[0]=append(out[0],1);println("retained")}
var count int
func key()int{count++;return 0}
func KeyEffect(){out:=map[int][]int{key():nil};out[0]=append(out[0],1)}
func Deferred(){out:=map[int][]int{0:nil};defer func(){saved=out}();out[0]=append(out[0],1)}
func Captured(){out:=map[int][]int{0:nil};_ =func(){saved=out};out[0]=append(out[0],1)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Run("private", func(t *testing.T) {
		for _, name := range []string{"NilSeed", "NilReset", "Discard", "NilSlice", "NilMap", "Forward"} {
			result := resultNamed(t, report, name)
			if !result.Complete || len(result.Effects) != 0 {
				t.Errorf("%s retains false effect: %+v", name, result)
			}
		}
	})
	t.Run("controls", func(t *testing.T) {
		for _, name := range []string{"Borrowed", "Seeded", "Mixed", "Shadowed", "Alias", "Rebound", "Publish", "RetainRow", "Child", "Callback", "Deferred", "Captured"} {
			function := findFunction(t, facts, name)
			write := function.Mutations[len(function.Mutations)-1]
			if write.Provenance == quality.Local {
				t.Errorf("%s loses shared or unproved write: %+v", name, write)
			}
		}
		for name, kind := range map[string]quality.Kind{"Unknown": quality.UnknownEffect, "IO": quality.IO, "KeyEffect": quality.Global} {
			result := resultNamed(t, report, name)
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("%s loses %s: %+v", name, kind, result)
			}
		}
	})
	data, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("saved nil-entry facts change evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.NilSeed", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	forward := resultNamed(t, overridden, "Forward")
	if !slices.ContainsFunc(forward.Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
		t.Errorf("nil-entry proof hides helper policy: %+v", forward)
	}
}
