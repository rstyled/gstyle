package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPrivateRowsWithLengthOnlyRange(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type Member struct{Name,SelfTy string}
// Mirrors domain/patterns/sigmine.go:sharedNameTypes with scalar keys.
func Local(members []Member)int{
 owners:=map[string]map[string]bool{}
 for _,m:=range members{if owners[m.Name]==nil{owners[m.Name]=map[string]bool{}};owners[m.Name][m.SelfTy]=true}
 best:=0
 for _,tys:=range owners{if len(tys)>best{best=len(tys)}}
 return best
}
func Forward(members []Member)int{return Local(members)}
type Row []int
type Rows []Row
func Capacity()int{out:=make(Rows,1);out[0]=make(Row,1,4);out[0][0]=1;n:=0;for _,row:=range (out){n+=(len((row))+cap(row))};return n}
func Keys()int{out:=map[int]map[int]bool{};out[0]=map[int]bool{};out[0][1]=true;n:=0;for key:=range out{n+=key};return n}
func Discard(){out:=map[int][]int{};out[0]=append(out[0],1);for range out{}}
func Borrowed(out map[int]map[int]bool){out[0][1]=true;for _,row:=range out{_=len(row)}}
func Seeded(input map[int]bool){out:=map[int]map[int]bool{0:input};out[0][1]=true;for _,row:=range out{_=len(row)}}
func Injected(input map[int]bool){out:=map[int]map[int]bool{};out[0]=input;out[0][1]=true;for _,row:=range out{_=len(row)}}
func Rebound(input map[int]map[int]bool){out:=map[int]map[int]bool{};out=input;out[0][1]=true;for _,row:=range out{_=len(row)}}
func Alias(){out:=map[int][]int{};out[0]=append(out[0],1);other:=out;for _,row:=range out{_=len(row)};_=other}
func RowAlias(){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{other:=row;_=len(other)}}
var saved []int
func Escaped(){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{saved=row};out[0]=append(out[0],2)}
func Assigned(){out:=map[int][]int{};out[0]=append(out[0],1);var row []int;for _,row=range out{_=len(row)};saved=row;out[0]=append(out[0],2)}
func Reassign(input []int){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{row=input;_=len(row)}}
func Address(){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{_= &row}}
func Element(){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{row[0]=2}}
func Call(callback func([]int)){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{callback(row)}}
func Shadowed(){out:=map[int][]int{};out[0]=append(out[0],1);len:=func(row []int)int{saved=row;return 0};for _,row:=range out{_=len(row)}}
type MethodRow []int
func(row MethodRow)Size()int{saved=row;return 0}
func Method(){out:=map[int]MethodRow{};out[0]=append(out[0],1);for _,row:=range out{_=row.Size()}}
func Closure(){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{func(){_=len(row)}()}}
func Deferred(){out:=map[int][]int{};out[0]=append(out[0],1);defer func(){}();for _,row:=range out{_=len(row)}}
func Child(input *int){out:=map[int][]*int{};out[0]=append(out[0],input);*out[0][0]=1;for _,row:=range out{_=len(row)}}
var counter int
func key()int{counter++;return 0}
func Argument(){out:=map[int][]int{};out[key()]=append(out[0],1);for _,row:=range out{_=len(row)}}
func IO(){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{println(len(row))}}
func Unknown(callback func()){out:=map[int][]int{};out[0]=append(out[0],1);for _,row:=range out{_=len(row)};callback()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Forward", "Capacity", "Keys", "Discard"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s only observes private row metadata: %+v", name, result)
		}
	}
	local := findFunction(t, facts, "Local")
	t.Logf("Local raw writes: %+v", local.Mutations)
	if len(local.Mutations) != 3 || slices.ContainsFunc(local.Mutations, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
		t.Errorf("local row write evidence changes: %+v", local.Mutations)
	}
	for _, name := range []string{"Borrowed", "Seeded", "Injected", "Rebound", "Alias", "RowAlias", "Escaped", "Assigned", "Reassign", "Address", "Element", "Call", "Shadowed", "Method", "Closure", "Deferred", "Child"} {
		result := resultNamed(t, report, name)
		writes := findFunction(t, facts, name).Mutations
		t.Logf("%s raw writes=%+v effects=%+v", name, writes, result.Effects)
		if len(result.Effects) == 0 || !slices.ContainsFunc(writes, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
			t.Errorf("%s hides shared or unknown storage: result=%+v writes=%+v", name, result, writes)
		}
	}
	for name, kind := range map[string]quality.Kind{"Borrowed": quality.MutationEffect, "Escaped": quality.Global, "Argument": quality.Global, "IO": quality.IO, "Unknown": quality.UnknownEffect} {
		result := resultNamed(t, report, name)
		t.Logf("%s raw=%+v effects=%+v", name, findFunction(t, facts, name), result.Effects)
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
		t.Fatal("saved range proof changes evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.Local", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		result := resultNamed(t, overridden, "Forward")
		if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("range proof hides explicit %s policy: %+v", kind, result)
		}
	}
}
