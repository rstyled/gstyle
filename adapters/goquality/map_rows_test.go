package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPrivateMapRows(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
// Mirrors domain/patterns.coModifiedPairs: empty map, fresh rows, direct writes.
func Local(input [][2]int)map[int]map[int]bool{
 out:=map[int]map[int]bool{}
 for _,pair:=range input{if out[pair[0]]==nil{out[pair[0]]=map[int]bool{}};out[pair[0]][pair[1]]=true}
 return out
}
func Forward(input [][2]int)map[int]map[int]bool{return Local(input)}
type Row map[int]bool
type Rows map[int]Row
func Named()Rows{out:=make(Rows);if nil!=(out[0]){};out[0]=make(Row);(out)[0][1]=true;return out}
func Slice()[]map[int]bool{out:=make([]map[int]bool,1);out[0]=map[int]bool{};out[0][1]=true;return out}
type Result struct{Rows map[int]map[int]bool}
func Wrapped()*Result{out:=map[int]map[int]bool{};out[0]=map[int]bool{};out[0][1]=true;return &Result{Rows:out}}
func Borrowed(out map[int]map[int]bool){out[0][1]=true}
func Seeded(input map[int]bool){out:=map[int]map[int]bool{0:input};out[0][1]=true}
func Injected(input map[int]bool){out:=map[int]map[int]bool{};out[0]=input;out[0][1]=true}
func Rebound(input map[int]map[int]bool){out:=map[int]map[int]bool{};out=input;out[0][1]=true}
func Alias(){out:=map[int]map[int]bool{};other:=out;out[0]=map[int]bool{};out[0][1]=true;_=other}
func RowAlias(){out:=map[int]map[int]bool{};out[0]=map[int]bool{};row:=out[0];row[1]=true}
var saved map[int]map[int]bool
var savedRow map[int]bool
func Published(){out:=map[int]map[int]bool{};out[0]=map[int]bool{};saved=out;out[0][1]=true}
func RowPublished(){out:=map[int]map[int]bool{};out[0]=map[int]bool{};savedRow=out[0];out[0][1]=true}
func Call(publish func(map[int]bool)){out:=map[int]map[int]bool{};out[0]=map[int]bool{};publish(out[0]);out[0][1]=true}
func Address(){out:=map[int]map[int]bool{};out[0]=map[int]bool{};p:=&out;(*p)[0][1]=true}
func Closure(){out:=map[int]map[int]bool{};out[0]=map[int]bool{};func(){out[0][1]=true}()}
func Deferred()map[int]map[int]bool{out:=map[int]map[int]bool{};out[0]=map[int]bool{};defer func(){out[0][1]=true}();return out}
func Pointee(input *int){out:=map[int]map[int]*int{};out[0]=map[int]*int{0:input};*out[0][0]=1}
func Child(input map[int]bool){out:=map[int]map[int]map[int]bool{};out[0]=map[int]map[int]bool{0:input};out[0][0][1]=true}
var counter int
func key()int{counter++;return 0}
func Argument(){out:=map[int]map[int]bool{};out[0]=map[int]bool{};out[key()][1]=true}
func IO(){out:=map[int]map[int]bool{};out[0]=map[int]bool{};out[0][1]=true;println("write")}
func Unknown(callback func()){out:=map[int]map[int]bool{};out[0]=map[int]bool{};out[0][1]=true;callback()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Forward", "Named", "Slice", "Wrapped"} {
		result := resultNamed(t, report, name)
		t.Logf("%s result=%+v raw=%+v", name, result, findFunction(t, facts, name))
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses private map row storage: %+v", name, result)
		}
	}
	local := findFunction(t, facts, "Local")
	if len(local.Mutations) != 2 || slices.ContainsFunc(local.Mutations, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
		t.Errorf("local map row construction loses raw local writes: %+v", local.Mutations)
	}
	for _, name := range []string{"Borrowed", "Seeded", "Injected", "Rebound", "Alias", "RowAlias", "Published", "RowPublished", "Call", "Address", "Closure", "Deferred", "Pointee", "Child"} {
		result := resultNamed(t, report, name)
		writes := findFunction(t, facts, name).Mutations
		t.Logf("%s result=%+v raw=%+v", name, result, writes)
		if len(result.Effects) == 0 || !slices.ContainsFunc(writes, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
			t.Errorf("%s hides shared or unknown map row storage", name)
		}
	}
	for name, kind := range map[string]quality.Kind{"Borrowed": quality.MutationEffect, "Published": quality.Global, "Argument": quality.Global, "IO": quality.IO, "Unknown": quality.UnknownEffect} {
		if result := resultNamed(t, report, name); !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses independent %s effect: %+v", name, kind, result)
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
		t.Fatal("saved map row facts change evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.Local", Kind: kind})
		changed, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if result := resultNamed(t, changed, "Forward"); !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("private map rows hide explicit %s policy: %+v", kind, result)
		}
	}
}
