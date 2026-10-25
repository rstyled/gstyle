package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestCopiedRecordFieldReplacement(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "slices"
type Record struct{ Items []int; Children []*int; Lookup map[int]int; Cell *int }
// Mirrors irBuilder.summaryInventory: copy the value, replace, then fill.
func Local(input *Record) *Record { out:=*input;out.Items=make([]int,len(input.Items));for i:=range input.Items{out.Items[i]=i};return &out }
func Forward(input *Record)*Record{return Local(input)}
func Value(input Record){out:=input;out.Items=make([]int,1);out.Items[0]=1}
func Parenthesized(input Record){out:=input;(out).Items=make([]int,1);((out).Items)[0]=1}
func Map(input Record){out:=input;out.Lookup=make(map[int]int);out.Lookup[0]=1}
func Pointer(input Record){out:=input;out.Cell=new(int);*out.Cell=1}
func Cloned(input Record){out:=input;out.Items=slices.Clone(input.Items);out.Items[0]=1}
func CloneForward(input Record){Cloned(input)}
func Borrowed(input Record){out:=input;out.Items[0]=1}
func Shared(input *Record){input.Items=make([]int,1);input.Items[0]=1}
func PointerAlias(input *Record){out:=input;out.Items=make([]int,1);out.Items[0]=1}
func UnknownPointer(next func()*Record){out:=next();out.Items=make([]int,1);out.Items[0]=1}
func Conditional(input Record,flag bool){out:=input;if flag{out.Items=make([]int,1)};out.Items[0]=1}
func Later(input Record){out:=input;out.Items[0]=1;out.Items=make([]int,1)}
func SameStatement(input Record){out:=input;out.Items,out.Items[0]=make([]int,1),1}
func Rebound(input Record){out:=Record{};out.Items=make([]int,1);out=input;out.Items[0]=1}
func FieldRebound(input Record,borrowed []int){out:=input;out.Items=make([]int,1);out.Items=borrowed;out.Items[0]=1}
func Loop(input Record){out:=input;for i:=0;i<2;i++{out.Items[0]=1;out.Items=make([]int,1)}}
func Labeled(input Record){out:=input;again:out.Items[0]=1;out.Items=make([]int,1);if false{goto again}}
var saved *Record
func Escaped(input Record){out:=input;saved=&out;out.Items=make([]int,1);out.Items[0]=1}
func Address(input Record,change func(*[]int)){out:=input;out.Items=make([]int,1);change(&out.Items);out.Items[0]=1}
func (r *Record) Reset(input []int){r.Items=input}
func Method(input Record,borrowed []int){out:=input;out.Items=make([]int,1);out.Reset(borrowed);out.Items[0]=1}
func Closure(input Record){out:=input;out.Items=make([]int,1);func(){saved=&out}();out.Items[0]=1}
func Deferred(input Record){out:=input;out.Items=make([]int,1);defer func(){saved=&out}();out.Items[0]=1}
func ClosureReturn(input Record,next func(func()*Record)){out:=input;out.Items=make([]int,1);next(func()*Record{return &out});out.Items[0]=1}
func DeferredHeader(input Record,borrowed []int){out:=input;out.Items=make([]int,1);defer func(){out.Items=borrowed}();out.Items[0]=1}
func SelfAppend(input Record){out:=input;out.Items=append(out.Items,1);out.Items[0]=1}
func Child(input Record,p *int){out:=input;out.Children=make([]*int,1);out.Children[0]=p;*out.Children[0]=1}
func IO(input Record){out:=input;out.Items=make([]int,1);out.Items[0]=1;println("io")}
var counter int
func size()int{counter++;return 1}
func Global(input Record){out:=input;out.Items=make([]int,size());out.Items[0]=1}
func Unknown(input Record,next func()int){out:=input;out.Items=make([]int,1);out.Items[0]=next()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Forward", "Value", "Parenthesized", "Map", "Pointer", "Cloned", "CloneForward"} {
		result := resultNamed(t, report, name)
		t.Logf("%s raw=%+v report=%+v", name, findFunction(t, facts, name).Mutations, result)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s writes only replaced private storage: %+v", name, result)
		}
	}
	local := findFunction(t, facts, "Local")
	if len(local.Mutations) != 2 || slices.ContainsFunc(local.Mutations, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
		t.Errorf("private header and element writes must remain: %+v", local.Mutations)
	}
	for _, name := range []string{"Borrowed", "Shared", "PointerAlias", "UnknownPointer", "Conditional", "Later", "SameStatement", "Rebound", "FieldRebound", "Loop", "Labeled", "Escaped", "Address", "Method", "Closure", "Deferred", "ClosureReturn", "DeferredHeader", "SelfAppend", "Child"} {
		writes := findFunction(t, facts, name).Mutations
		result := resultNamed(t, report, name)
		t.Logf("%s raw=%+v effects=%+v", name, writes, result.Effects)
		if len(result.Effects) == 0 || !slices.ContainsFunc(writes, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
			t.Errorf("%s loses shared or unknown write: %+v", name, result)
		}
	}
	for name, kind := range map[string]quality.Kind{"Shared": quality.MutationEffect, "Escaped": quality.Global, "IO": quality.IO, "Global": quality.Global, "Unknown": quality.UnknownEffect} {
		if result := resultNamed(t, report, name); !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
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
		t.Fatal("saved replacement facts change evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.Local", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if result := resultNamed(t, overridden, "Forward"); len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("replacement proof hides explicit %s: %+v", kind, result)
		}
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "slices.Clone", Kind: kind})
		overridden, err = quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if result := resultNamed(t, overridden, "CloneForward"); len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("fresh clone hides explicit %s: %+v", kind, result)
		}
	}
}
