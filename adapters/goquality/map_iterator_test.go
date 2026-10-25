package goquality

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPrivateRowsWithSortedMapKeys(t *testing.T) {
	// Keep the actual self-scan body, including its outer len(ruleSets) use.
	groups, err := os.ReadFile("../../domain/quality/groups.go")
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Split(strings.Split(string(groups), "func violatorRules(")[1], "\n// callGraph")[0]
	source := `package repro
import("iter";"maps";"slices")
type Function struct{Path string;Line,Column int}
type Diagnostic struct{Path string;Line,Column int;RuleID string}
type functionKey struct{path string;line,column int}
func violatorRules(` + body + `
func Forward(f []Function,d []Diagnostic)map[int][]string{return violatorRules(f,d)}
func Parens(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{_=slices.Sorted[string]((maps.Keys[map[string]bool]((row))))}}
func Borrowed(rows map[int]map[string]bool){rows[0]["a"]=true;for _,row:=range rows{_=slices.Sorted(maps.Keys(row))}}
func Seeded(input map[string]bool){rows:=map[int]map[string]bool{0:input};rows[0]["a"]=true;for _,row:=range rows{_=slices.Sorted(maps.Keys(row))}}
func Injected(input map[string]bool){rows:=map[int]map[string]bool{};rows[0]=input;rows[0]["a"]=true;for _,row:=range rows{_=slices.Sorted(maps.Keys(row))}}
func Rebound(input map[int]map[string]bool){rows:=map[int]map[string]bool{};rows=input;rows[0]["a"]=true;for _,row:=range rows{_=slices.Sorted(maps.Keys(row))}}
var saved map[string]bool
var sequence iter.Seq[string]
func Escaped(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{saved=row;_=slices.Sorted(maps.Keys(row))}}
func IteratorEscape(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{sequence=maps.Keys(row)}}
func IteratorAlias(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{seq:=maps.Keys(row);_=slices.Sorted(seq)}}
func RowAlias(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{alias:=row;_=slices.Sorted(maps.Keys(alias))}}
func Write(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{row["b"]=true;_=slices.Sorted(maps.Keys(row))}}
func Address(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{_=&row;_=slices.Sorted(maps.Keys(row))}}
func Assigned(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;var row map[string]bool;for _,row=range rows{_=slices.Sorted(maps.Keys(row))};saved=row}
func Closure(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{func(){_=slices.Sorted(maps.Keys(row))}()}}
func Deferred(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{defer slices.Sorted(maps.Keys(row))}}
func Async(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{go slices.Sorted(maps.Keys(row))}}
func Callback(run func(iter.Seq[string])){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{run(maps.Keys(row))}}
func Comparator(run func(string,string)int){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{_=slices.SortedFunc(maps.Keys(row),run)}}
func ShadowedKeys(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;maps:=struct{Keys func(map[string]bool)iter.Seq[string]}{func(row map[string]bool)iter.Seq[string]{saved=row;return nil}};for _,row:=range rows{_=slices.Sorted(maps.Keys(row))}}
func ShadowedSorted(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;slices:=struct{Sorted func(iter.Seq[string])[]string}{func(seq iter.Seq[string])[]string{sequence=seq;return nil}};for _,row:=range rows{_=slices.Sorted(maps.Keys(row))}}
func Child(input *int){rows:=map[int]map[string]*int{};rows[0]=map[string]*int{"a":input};*rows[0]["a"]=1;for _,row:=range rows{_=slices.Sorted(maps.Keys(row))}}
func IO(){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{println(len(slices.Sorted(maps.Keys(row))))}}
func Unknown(run func()){rows:=map[int]map[string]bool{};rows[0]=map[string]bool{};rows[0]["a"]=true;for _,row:=range rows{_=slices.Sorted(maps.Keys(row))};run()}
`
	root := qualitySourceModule(t, []byte(source))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"violatorRules", "Forward", "Parens"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s only collects private map keys: %+v", name, result)
		}
	}
	local := findFunction(t, facts, "violatorRules")
	if len(local.Mutations) != 4 || slices.ContainsFunc(local.Mutations, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
		t.Errorf("actual source loses raw local writes: %+v", local.Mutations)
	}
	for _, name := range []string{"Borrowed", "Seeded", "Injected", "Rebound", "Escaped", "IteratorEscape", "IteratorAlias", "RowAlias", "Write", "Address", "Assigned", "Closure", "Deferred", "Async", "Callback", "Comparator", "ShadowedKeys", "ShadowedSorted", "Child"} {
		result := resultNamed(t, report, name)
		writes := findFunction(t, facts, name).Mutations
		if len(result.Effects) == 0 || !slices.ContainsFunc(writes, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
			t.Errorf("%s hides unproved storage: result=%+v writes=%+v", name, result, writes)
		}
	}
	for name, kind := range map[string]quality.Kind{"Borrowed": quality.MutationEffect, "Escaped": quality.Global, "Callback": quality.UnknownEffect, "IO": quality.IO, "Unknown": quality.UnknownEffect} {
		if !slices.ContainsFunc(resultNamed(t, report, name).Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses independent %s", name, kind)
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
		t.Fatal("saved iterator evidence changes evaluation", err)
	}
	for _, callee := range []string{"maps.Keys", "slices.Sorted", "example.com/quality-repro.violatorRules"} {
		for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
			config := quality.DefaultConfig()
			config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: kind})
			overridden, err := quality.Evaluate(restored, config)
			if err != nil {
				t.Fatal(err)
			}
			result := resultNamed(t, overridden, "Forward")
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("iterator proof hides %s %s policy: %+v", callee, kind, result)
			}
		}
	}
}

func BenchmarkSortedMapKeyExtraction(b *testing.B) {
	root := os.Getenv("GSTYLE_ROW_CORPUS")
	if root == "" {
		b.Skip("set GSTYLE_ROW_CORPUS to a fixed source corpus")
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := (Analyzer{Root: root}).Extract(context.Background(), nil); err != nil {
			b.Fatal(err)
		}
	}
}
