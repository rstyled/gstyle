package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestAssertedStoragePreservesOrigin(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "go/ast"
// Mirrors buildStructLiteral in adapters/gopatterns/valuefix.go.
func Literal() *ast.CompositeLit {
 lit := &ast.CompositeLit{Type: &ast.Ident{Name:"Record"}}
 lit.Type.(*ast.Ident).NamePos = 1
 return lit
}
func Forward(){Literal()}
type holder struct{Value any}
type node struct{Value int; Child *int; Items []int}
func Pointer(){h:=holder{Value:&node{}};h.Value.(*node).Value=1}
func Slice(){h:=holder{Value:[]int{0}};h.Value.([]int)[0]=1}
func Map(){h:=holder{Value:map[int]int{}};h.Value.(map[int]int)[0]=1}
func Parentheses(){h:=holder{Value:&node{}};((h.Value)).(*node).Value=1}
func Alias(){h:=holder{Value:&node{}};p:=h.Value.(*node);p.Value=1}
func Shared(value any){value.(*node).Value=1}
func Borrowed(input *node){h:=holder{Value:input};h.Value.(*node).Value=1}
func Mixed(input *node,flag bool){h:=holder{Value:&node{}};if flag{h.Value=input};h.Value.(*node).Value=1}
func Rebound(input any){h:=holder{Value:&node{}};h.Value=input;h.Value.(*node).Value=1}
func Unknown(factory func()any){h:=holder{Value:factory()};h.Value.(*node).Value=1}
func Child(input *int){h:=holder{Value:&node{Child:input}};*h.Value.(*node).Child=1}
func Items(input []int){h:=holder{Value:&node{Items:input}};h.Value.(*node).Items[0]=1}
func Nested(input *int){h:=holder{Value:[]*int{input}};*h.Value.([]*int)[0]=1}
func Address(input *node){h:=holder{Value:&node{}};p:=&h.Value;*p=input;h.Value.(*node).Value=1}
func IO(){h:=holder{Value:&node{}};h.Value.(*node).Value=1;println("write")}
func Callback(callback func()){h:=holder{Value:&node{}};h.Value.(*node).Value=1;callback()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Literal", "Forward", "Pointer", "Slice", "Map", "Parentheses", "Alias"} {
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v", name, result)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses owned asserted storage: %+v", name, result)
		}
		if name != "Forward" {
			mutations := findFunction(t, facts, name).Mutations
			if len(mutations) != 1 || mutations[0].Provenance != quality.Local {
				t.Errorf("%s loses its local mutation: %+v", name, mutations)
			}
		}
	}
	mutations := findFunction(t, facts, "Literal").Mutations
	if len(mutations) != 1 || mutations[0].Provenance != quality.Local || mutations[0].FieldPath != "NamePos" {
		t.Errorf("literal assertion must retain its local write: %+v", mutations)
	}
	for _, name := range []string{"Shared", "Borrowed", "Mixed", "Rebound", "Unknown", "Child", "Items", "Nested", "Address"} {
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v", name, result)
		kind := quality.UnknownEffect
		if name == "Shared" {
			kind = quality.MutationEffect
		}
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses shared or unknown storage: %+v", name, result)
		}
	}
	for name, kind := range map[string]quality.Kind{"IO": quality.IO, "Callback": quality.UnknownEffect} {
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v", name, result)
		if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
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
		t.Fatal("saved asserted-storage evidence changes evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.Literal", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(resultNamed(t, overridden, "Forward").Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
		t.Fatal("asserted storage hides explicit helper policy")
	}
}
