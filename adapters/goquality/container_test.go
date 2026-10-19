package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestContainerStorageOwnership(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"maps";"slices";"sort";"strings")
var shared = []int{2,1}
var counter int
type Numbers []int
type Text string
type Runes []rune
type Seed struct{}
func(Seed) seeds(input []int) []int {var out []int;for _,v:=range input {out=append(out,v)};return out}
func filled(n int) []int {out:=make([]int,n);for i:=range out {out[i]=i};return out}
func mapping(input map[int]int) map[int]int {out:=make(map[int]int);for k,v:=range input {out[k]=v};return out}
func mixedReturn(input []int, flag bool) []int {if flag{return input};out:=make([]int,1);return out}
func publishedReturn() []int {out:=make([]int,1);shared=out;return out}
func writtenReturn(dst *[]int) []int {out:=make([]int,1);*dst=out;return out}
var leakedElement *int
func retain(out []int){shared=out}
func escapedReturn() []int {out:=make([]int,1);retain(out);return out}
func addressedReturn() []int {out:=make([]int,1);leakedElement=&out[0];return out}
func closureReturn() []int {out:=make([]int,1);f:=func(){out[0]=1};_ =f;return out}
func deferredReturn(input []int) []int {out:=make([]int,1);defer copy(out,input);return out}
func outputReturn() []int {out:=make([]int,1);println("output");return out}
func ReturnedQueue(input []int){out:=(Seed{}).seeds(input);out=append(out,1);slices.Reverse(out)}
func ReturnedFilled(){out:=filled(2);slices.Reverse(out)}
func ReturnedMap(){maps.Copy(mapping(nil),map[int]int{1:1})}
func ReturnedMixed(){out:=mixedReturn(shared,true);slices.Reverse(out)}
func ReturnedPublished(){out:=publishedReturn();slices.Reverse(out)}
func ReturnedWritten(){out:=writtenReturn(new([]int));slices.Reverse(out)}
func ReturnedEscape(){out:=escapedReturn();slices.Reverse(out)}
func ReturnedAddress(){out:=addressedReturn();slices.Reverse(out)}
func ReturnedClosure(){out:=closureReturn();slices.Reverse(out)}
func ReturnedDeferred(){out:=deferredReturn(shared);slices.Reverse(out)}
func ReturnedOutput(){out:=outputReturn();slices.Reverse(out)}
func FreshContainers(text string, data []byte, values []int){
slices.Reverse(strings.Split(text,","));slices.Reverse(strings.SplitN(text,",",1))
slices.Reverse(strings.SplitAfter(text,","));slices.Reverse(strings.SplitAfterN(text,",",1));slices.Reverse(strings.Fields(text))
slices.Reverse(bytes.Clone(data));slices.Reverse(bytes.Repeat(data,1))
slices.Reverse(bytes.Split(data,[]byte(",")));slices.Reverse(bytes.SplitN(data,[]byte(","),1))
slices.Reverse(bytes.SplitAfter(data,[]byte(",")));slices.Reverse(bytes.SplitAfterN(data,[]byte(","),1));slices.Reverse(bytes.Fields(data))
slices.Reverse(slices.Concat(values));slices.Reverse(slices.Repeat(values,1))
}
func BorrowedBytes(data []byte){values:=bytes.Split(data,[]byte(","));values[0][0]=1}
func BorrowedConcat(data []*int){values:=slices.Concat(data);*values[0]=1}
func CallbackFields(text string, test func(rune)bool){values:=strings.FieldsFunc(text,test);slices.Reverse(values)}
func FreshBytes(text string){values:=[]byte(text);slices.Reverse(values)}
func FreshRunes(text string){values:=[]rune(text);slices.Reverse(values)}
func NamedString(text Text){values:=Runes(text);slices.Reverse(values)}
func NilClone(src []int){values:=append([]int(nil),src...);slices.Sort(values)}
func NilSelf(src []int){values:=[]int(nil);values=append(values,src...);slices.Sort(values)}
func NilSort(){slices.Sort([]int(nil))}
func NilDirect(){slices.Reverse[[]int](nil)}
func OrdinaryConversion(values []int){converted:=Numbers(values);slices.Sort(converted)}
func ShadowNil(values []int){nil:=values;slices.Sort(nil)}
func NilFunction(){var callback func();callback()}
func ShallowNilClone(src []*int){values:=append([]*int(nil),src...);*values[0]=1}
func LocalSort(){values:=[]int{2,1}; slices.Sort(values)}
func NamedSort(){values:=Numbers{2,1}; slices.Sort(values)}
func LocalCopy(src map[int]int){dst:=make(map[int]int); maps.Copy(dst,src)}
func LocalReverse(){values:=[]int{2,1}; slices.Reverse(values)}
func LocalCompact(){values:=[]int{1,1}; _=slices.Compact(values)}
func LocalDelete(){values:=[]int{2,1}; _=slices.Delete(values,0,1)}
func LocalInsert(){values:=[]int{2,1}; _=slices.Insert(values,0,3)}
func LocalReplace(){values:=[]int{2,1}; _=slices.Replace(values,0,1,3)}
func LocalInts(){values:=[]int{2,1}; sort.Ints(values)}
func LocalFloats(){values:=[]float64{2,1}; sort.Float64s(values)}
func LocalStrings(){values:=[]string{"b","a"}; sort.Strings(values)}
func LocalAlias(){values:=[]int{2,1}; alias:=values[1:]; slices.Sort(alias)}
func LocalAppend(){values:=append([]int{2,1},3); slices.Sort(values)}
func SelfAppend(){values:=[]int{2,1}; values=append(values,3); slices.Sort(values)}
func CloneSort(values []int){local:=slices.Clone(values); slices.Sort(local)}
func CloneCopy(dst,src map[int]int){local:=maps.Clone(dst); maps.Copy(local,src)}
func Shallow(values []*int){local:=slices.Clone(values); *local[0]=1}
func Clone(values []int)[]int{return values}
func FakeClone(values []int){local:=Clone(values); slices.Sort(local)}
func ExternalSort(values []int){slices.Sort(values)}
func ExternalCopy(dst,src map[int]int){maps.Copy(dst,src)}
func GenericSort[S ~[]int](values S){slices.Sort(values)}
func GlobalSort(){slices.Sort(shared)}
func ExternalAlias(values []int){alias:=values[1:]; slices.Sort(alias)}
func ExternalAppend(values []int){alias:=append(values,3); slices.Sort(alias)}
func Mixed(values []int, flag bool){alias:=[]int{2,1}; if flag{alias=values}; slices.Sort(alias)}
func Unknown(factory func()[]int){values:=factory(); slices.Sort(values)}
func Callback(compare func(int,int)int){values:=[]int{2,1}; slices.SortFunc(values,compare)}
func InlineSort(){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{return a-b})}
func InlineStable(){v:=[]int{2,1};slices.SortStableFunc(v,func(a,b int)int{return a-b})}
func InlineReflect(){v:=[]int{2,1};sort.Slice(v,func(a,b int)bool{return v[a]<v[b]})}
func InlineReflectStable(){v:=[]int{2,1};sort.SliceStable(v,func(a,b int)bool{return v[a]<v[b]})}
func InlineShared(v []int){slices.SortFunc(v,func(a,b int)int{return a-b})}
func NamedCallback(){v:=[]int{2,1};slices.SortFunc(v,compare)}
func compare(a,b int)int{return a-b}
func VariableCallback(){v:=[]int{2,1};compare:=func(a,b int)int{return a-b};slices.SortFunc(v,compare)}
func InlineOutput(){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{println(a);return a-b})}
func InlineCaptured(shared *int){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{*shared++;return a-b})}
func InlineGlobal(){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{return a-b+counter})}
func InlineLaunch(){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{go func(){}();return a-b})}
func InlineChannel(ch chan int){v:=[]int{2,1};slices.SortFunc(v,func(a,b int)int{ch<-a;return a-b})}
func InlinePolicy(){v:=[]string{"b","a"};slices.SortFunc(v,func(a,b string)int{return strings.Compare(a,b)})}
func helper(){values:=[]int{2,1}; slices.Sort(values)}
func Caller(){helper()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ReturnedQueue", "ReturnedFilled", "ReturnedMap", "InlineSort", "InlineStable", "InlineReflect", "InlineReflectStable", "InlinePolicy", "FreshContainers", "FreshBytes", "FreshRunes", "NamedString", "NilClone", "NilSelf", "NilSort", "NilDirect", "LocalSort", "NamedSort", "LocalCopy", "LocalReverse", "LocalCompact", "LocalDelete", "LocalInsert", "LocalReplace", "LocalInts", "LocalFloats", "LocalStrings", "LocalAlias", "LocalAppend", "SelfAppend", "CloneSort", "CloneCopy", "Caller"} {
		t.Run(name, func(t *testing.T) {
			r := resultNamed(t, report, name)
			if len(r.Effects) != 0 || !r.Complete {
				t.Fatalf("local container effects: %+v", r)
			}
		})
	}
	for _, tc := range []struct {
		name       string
		provenance quality.Provenance
	}{
		{"ReturnedMixed", quality.Unknown}, {"ReturnedPublished", quality.Unknown}, {"ReturnedWritten", quality.Unknown}, {"ReturnedEscape", quality.Unknown}, {"ReturnedAddress", quality.Unknown}, {"ReturnedClosure", quality.Unknown}, {"ReturnedDeferred", quality.Unknown}, {"InlineShared", quality.External}, {"CallbackFields", quality.Unknown}, {"ShadowNil", quality.External}, {"OrdinaryConversion", quality.External}, {"ExternalSort", quality.External}, {"ExternalCopy", quality.External}, {"GenericSort", quality.External}, {"GlobalSort", quality.External}, {"ExternalAlias", quality.External}, {"ExternalAppend", quality.Unknown}, {"FakeClone", quality.Unknown}, {"Mixed", quality.Unknown}, {"Unknown", quality.Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := findFunction(t, facts, tc.name)
			call := f.Calls[len(f.Calls)-1]
			if call.MutationTarget != tc.provenance {
				t.Fatalf("target: %+v, want %s", call, tc.provenance)
			}
			r := resultNamed(t, report, tc.name)
			found := false
			for _, e := range r.Effects {
				found = found || e.Kind == quality.MutationEffect
			}
			if !found {
				t.Fatalf("observable or unknown target lost mutation: %+v", r)
			}
		})
	}
	for _, name := range []string{"Shallow", "ShallowNilClone", "NilFunction", "BorrowedBytes", "BorrowedConcat"} {
		if len(resultNamed(t, report, name).Effects) == 0 {
			t.Fatalf("%s loses pointee or dynamic-call evidence", name)
		}
	}
	for _, name := range []string{"Callback", "NamedCallback", "VariableCallback"} {
		callback := findFunction(t, facts, name)
		if callback.Calls[0].MutationTarget != "" || callback.Calls[0].CallbackExpanded || len(resultNamed(t, report, name).Effects) == 0 {
			t.Fatalf("unproved callback %s became local: %+v", name, callback)
		}
	}
	for _, tc := range []struct {
		name string
		kind quality.Kind
	}{
		{"ReturnedOutput", quality.IO}, {"InlineOutput", quality.IO}, {"InlineCaptured", quality.MutationEffect}, {"InlineGlobal", quality.Global}, {"InlineLaunch", quality.UnknownEffect}, {"InlineChannel", quality.UnknownEffect},
	} {
		got := resultNamed(t, report, tc.name)
		found := false
		for _, effect := range got.Effects {
			found = found || effect.Kind == tc.kind
		}
		if !found {
			t.Fatalf("%s loses %s callback evidence: %+v", tc.name, tc.kind, got)
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
	roundtrip, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report, roundtrip) {
		t.Fatal("saved ownership facts changed evaluation")
	}
	for _, path := range []string{"strings.Compare", "slices.SortFunc"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: path, Kind: quality.IO})
		replay, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		got := resultNamed(t, replay, "InlinePolicy")
		if len(got.Effects) != 1 || got.Effects[0].Kind != quality.IO {
			t.Fatalf("saved callback hid %s override: %+v", path, got)
		}
	}
	override := quality.DefaultConfig()
	override.Prefixes = append(override.Prefixes, quality.Prefix{Path: "slices.Sort", Kind: quality.IO})
	changed, err := quality.Evaluate(restored, override)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LocalSort", "Caller"} {
		r := resultNamed(t, changed, name)
		if len(r.Effects) != 1 || r.Effects[0].Kind != quality.IO {
			t.Fatalf("policy override lost for %s: %+v", name, r)
		}
	}
}

func TestContainerHelpersAcrossPackages(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "example.com/quality-repro/lib"
func Caller(values []int)[]int{return lib.Sorted(values)}
func Shared(values []int){lib.Shared(values)}
func InlineHelper(values []int)[]int{return lib.SortedInline(values)}
`))
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0700); err != nil {
		t.Fatal(err)
	}
	source := `package lib
import "slices"
func Sorted(values []int)[]int{local:=slices.Clone(values); slices.Sort(local); return local}
func Shared(values []int){slices.Sort(values)}
func SortedInline(values []int)[]int{local:=slices.Clone(values);slices.SortFunc(local,func(a,b int)int{return a-b});return local}
`
	if err := os.WriteFile(filepath.Join(root, "lib/helper.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	caller := resultNamed(t, report, "Caller")
	if len(caller.Effects) != 0 || !caller.Complete {
		t.Fatalf("fresh helper reported effects: %+v", caller)
	}
	shared := resultNamed(t, report, "Shared")
	if len(shared.Effects) != 1 || shared.Effects[0].Kind != quality.MutationEffect || !shared.Complete {
		t.Fatalf("external helper lost observable write: %+v", shared)
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	inline := resultNamed(t, report, "InlineHelper")
	if len(inline.Effects) != 0 || !inline.Complete {
		t.Fatalf("inline helper stays effectful: %+v", inline)
	}
	callbackConfig := quality.DefaultConfig()
	callbackConfig.Prefixes = append(callbackConfig.Prefixes, quality.Prefix{Path: "slices.SortFunc", Kind: quality.IO})
	changed, err := quality.Evaluate(restored, callbackConfig)
	if err != nil {
		t.Fatal(err)
	}
	inline = resultNamed(t, changed, "InlineHelper")
	if len(inline.Effects) != 1 || inline.Effects[0].Kind != quality.IO {
		t.Fatalf("saved helper callback proof hid override: %+v", inline)
	}
	for _, callee := range []string{"slices.Sort", "slices.Clone"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: quality.IO})
		changed, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		caller := resultNamed(t, changed, "Caller")
		if len(caller.Effects) != 1 || caller.Effects[0].Kind != quality.IO {
			t.Fatalf("saved helper hid %s override: %+v", callee, caller)
		}
	}
	for i := range restored {
		for j := range restored[i].Calls {
			restored[i].Calls[j].MutationTarget = ""
		}
		for j := range restored[i].Helpers {
			for k := range restored[i].Helpers[j].Calls {
				restored[i].Helpers[j].Calls[k].MutationTarget = ""
			}
		}
	}
	legacy, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(resultNamed(t, legacy, "Caller").Effects) == 0 {
		t.Fatal("missing ownership evidence silently treated as local")
	}
}
