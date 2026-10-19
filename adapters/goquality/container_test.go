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
import("maps";"slices";"sort")
var shared = []int{2,1}
type Numbers []int
type Text string
type Runes []rune
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
	for _, name := range []string{"FreshBytes", "FreshRunes", "NamedString", "NilClone", "NilSelf", "NilSort", "NilDirect", "LocalSort", "NamedSort", "LocalCopy", "LocalReverse", "LocalCompact", "LocalDelete", "LocalInsert", "LocalReplace", "LocalInts", "LocalFloats", "LocalStrings", "LocalAlias", "LocalAppend", "SelfAppend", "CloneSort", "CloneCopy", "Caller"} {
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
		{"ShadowNil", quality.External}, {"OrdinaryConversion", quality.External}, {"ExternalSort", quality.External}, {"ExternalCopy", quality.External}, {"GenericSort", quality.External}, {"GlobalSort", quality.External}, {"ExternalAlias", quality.External}, {"ExternalAppend", quality.Unknown}, {"FakeClone", quality.Unknown}, {"Mixed", quality.Unknown}, {"Unknown", quality.Unknown},
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
	for _, name := range []string{"Shallow", "ShallowNilClone", "NilFunction"} {
		if len(resultNamed(t, report, name).Effects) == 0 {
			t.Fatalf("%s loses pointee or dynamic-call evidence", name)
		}
	}
	callback := findFunction(t, facts, "Callback")
	if callback.Calls[0].MutationTarget != "" {
		t.Fatal("callback-based operation received ownership proof")
	}
	if len(resultNamed(t, report, "Callback").Effects) == 0 {
		t.Fatal("callback operation became pure")
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
`))
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0700); err != nil {
		t.Fatal(err)
	}
	source := `package lib
import "slices"
func Sorted(values []int)[]int{local:=slices.Clone(values); slices.Sort(local); return local}
func Shared(values []int){slices.Sort(values)}
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
