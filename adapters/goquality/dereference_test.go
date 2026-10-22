package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestSharedDereferenceKeepsMutationWithoutUnknown(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("go/ast";"go/token")
type EntityIdentityHit struct{Line int}
// Mirrors adapters/gopatterns/entity.go:addUniqueHit in the master self-scan.
func addUniqueHit(out *[]EntityIdentityHit, seen map[int]bool, hit EntityIdentityHit, fset *token.FileSet, cond ast.Expr) {
 pos := fset.PositionFor(cond.Pos(), false)
 if !seen[pos.Line] {
  seen[pos.Line] = true
  *out = append(*out, hit)
 }
}
func Shared(out *[]int){*out=append(*out,1)}
func Forward(){var out []int;Shared(&out)}
type State struct{Count int}
func Pointer(shared **State){(*shared).Count++}
func Map(shared *map[int]int){(*shared)[0]++}
func Local(){value:=[]int{0};pointer:=&value;(*pointer)[0]++}
func Borrowed(shared []int){pointer:=&shared;(*pointer)[0]++}
func Unknown(factory func()*[]int){(*factory())[0]++}
func Rebound(shared *[]int){pointer:=new([]int);pointer=shared;(*pointer)[0]++}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	hit := resultNamed(t, report, "addUniqueHit")
	if !hit.Complete || hit.DensityMilli != 500 || len(hit.Effects) != 3 || hit.Mutations != 3 {
		t.Errorf("self-scan append wants three writes at two effect sites, no unknown: %+v", hit)
	}
	for _, name := range []string{"addUniqueHit", "Shared", "Forward", "Pointer", "Map"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) == 0 || slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind != quality.MutationEffect }) {
			t.Errorf("%s must retain shared mutations only: %+v", name, result)
		}
	}
	for _, name := range []string{"Local", "Borrowed", "Unknown", "Rebound"} {
		result := resultNamed(t, report, name)
		if result.Complete || !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.UnknownEffect }) {
			t.Errorf("%s must keep uncertain referenced storage: %+v", name, result)
		}
	}
	mutations := findFunction(t, facts, "Shared").Mutations
	if len(mutations) != 2 || mutations[1].Parameter != nil {
		t.Errorf("indirect append storage must not bind to outer pointer: %+v", mutations)
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
		t.Fatal("saved dereference evidence changed", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "go/token.FileSet.PositionFor", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(resultNamed(t, overridden, "addUniqueHit").Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
		t.Fatal("explicit call policy lost")
	}
}
