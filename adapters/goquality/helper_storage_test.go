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

func TestHelperStorageBindings(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("fmt";"example.com/quality-repro/lib")
type State struct{N int; Values []int; Pointer *int}
type Box struct{*State}
var global = new(State)
func set(p *State){p.N++}
func chain(p *State){set(p)}
func rebind(p *State){p=global;p.N++}
func field(p *State){p.Values[0]++}
func indirect(p **State){(*p).N++}
func (p *State) Set(){p.N++}
func (p *State) Rebind(){p=global;p.N++}
func (p *State) PointerWrite(){*p.Pointer=1}
func ioHelper(p *State){p.N++;fmt.Println(p.N)}
func Owned(){p:=new(State);chain(p)}
func Literal(){set(&State{})}
func Shared(p *State){chain(p)}
func Mixed(p *State,flag bool){q:=new(State);if flag{q=p};chain(q)}
func Twice(p *State){q:=new(State);set(q);set(p)}
func ReverseTwice(p *State){q:=new(State);set(p);set(q)}
func Reassigned(){p:=new(State);rebind(p)}
func ReferenceField(values []int){p:=new(State);p.Values=values;field(p)}
func Indirect(){p:=global;indirect(&p)}
func Receiver(){p:=new(State);p.Set()}
func AutoAddress(){var p State;p.Set()}
func SharedReceiver(p *State){p.Set()}
func MethodExpr(){p:=new(State);(*State).Set(p)}
func SharedMethodExpr(p *State){(*State).Set(p)}
func ReboundReceiver(){p:=new(State);p.Rebind()}
func Promoted(){var b Box;b.State=global;b.Set()}
func PromotedExpr(){var b Box;b.State=global;(*Box).Set(&b)}
func NestedPointer(p *int){s:=new(State);s.Pointer=p;s.PointerWrite()}
func IO(){p:=new(State);ioHelper(p)}
func CrossOwned(){p:=new(lib.State);lib.Chain(p)}
func CrossShared(p *lib.State){lib.Chain(p)}
func CrossReceiver(){p:=new(lib.State);p.Set()}
func CrossSort(){values:=[]int{2,1};lib.Sort(values)}
func CrossSharedSort(values []int){lib.Sort(values)}
func FunctionField(f func(*State)){s:=struct{F func(*State)}{f};s.F(new(State))}
`))
	if err := os.MkdirAll(filepath.Join(root, "lib"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib/helper.go"), []byte(`package lib
import "slices"
type State struct{N int}
func Set(p *State){p.N++}
func Chain(p *State){Set(p)}
func (p *State) Set(){p.N++}
func Sort(values []int){slices.Sort(values)}
`), 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Owned", "Literal", "Receiver", "AutoAddress", "MethodExpr", "CrossOwned", "CrossReceiver", "CrossSort"} {
		t.Run(name, func(t *testing.T) {
			r := resultNamed(t, report, name)
			if len(r.Effects) != 0 || !r.Complete {
				t.Fatalf("owned helper effects: %+v", r)
			}
		})
	}
	for _, name := range []string{"Shared", "Mixed", "Twice", "ReverseTwice", "Reassigned", "ReferenceField", "Indirect", "SharedReceiver", "SharedMethodExpr", "ReboundReceiver", "Promoted", "PromotedExpr", "NestedPointer", "CrossShared", "CrossSharedSort", "FunctionField"} {
		t.Run(name, func(t *testing.T) {
			r := resultNamed(t, report, name)
			visible := false
			for _, effect := range r.Effects {
				visible = visible || effect.Kind == quality.MutationEffect || effect.Kind == quality.UnknownEffect
			}
			if !visible {
				t.Fatalf("observable or unknown helper write lost: %+v", r)
			}
		})
	}
	for _, name := range []string{"Twice", "ReverseTwice"} {
		r := resultNamed(t, report, name)
		if len(r.Effects) != 1 || r.Effects[0].Kind != quality.MutationEffect {
			t.Fatalf("context cache conflated %s: %+v", name, r)
		}
	}
	io := resultNamed(t, report, "IO")
	if len(io.Effects) != 1 || io.Effects[0].Kind != quality.IO {
		t.Fatalf("IO was folded with local write: %+v", io)
	}
	encoded, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(encoded) {
		t.Fatal("evaluation changed source facts")
	}
	var restored []quality.Function
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	after, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report, after) {
		t.Fatal("saved binding evaluation changed")
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro/lib.Set", Kind: quality.IO})
	changed, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	if r := resultNamed(t, changed, "CrossOwned"); len(r.Effects) != 1 || r.Effects[0].Kind != quality.IO {
		t.Fatalf("helper override hidden: %+v", r)
	}
	// New formal evidence with old call facts cannot establish local ownership.
	method := findFunction(t, restored, "Receiver")
	method.Calls[0].Receiver = nil
	missingReceiver, err := quality.Evaluate([]quality.Function{method}, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if r := resultNamed(t, missingReceiver, "Receiver"); len(r.Effects) == 0 || r.Complete {
		t.Fatalf("missing receiver evidence became clean: %+v", r)
	}
	caller := findFunction(t, restored, "Owned")
	caller.Calls[0].StorageArgs = nil
	legacy, err := quality.Evaluate([]quality.Function{caller}, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if r := resultNamed(t, legacy, "Owned"); len(r.Effects) == 0 || r.Complete {
		t.Fatalf("missing ownership evidence became clean: %+v", r)
	}
}
