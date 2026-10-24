package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPrivateEntriesInReturnedLiteral(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
type Traversal struct { Backward [][]int; Lookup map[int][]int; Child [][]*int }
// Mirrors domain/patterns.newReturnTraversal: publish only in the return literal.
func TraversalFor(edges [][2]int) *Traversal {
 backward:=make([][]int,4)
 for _,edge:=range edges { backward[edge[0]]=append(backward[edge[0]],edge[1]) }
 return &Traversal{Backward:backward}
}
func Forward(edges [][2]int)*Traversal{return TraversalFor(edges)}
func Value()Traversal{out:=make([][]int,1);out[0]=append(out[0],1);return Traversal{Backward:out}}
func Mapping()*Traversal{out:=map[int][]int{};out[0]=append(out[0],1);return &Traversal{Lookup:(out)}}
func Parentheses()*Traversal{out:=make([][]int,1);out[0]=append(out[0],1);return (&(Traversal{Backward:(out)}))}
func Borrowed(out [][]int)*Traversal{out[0]=append(out[0],1);return &Traversal{Backward:out}}
func Seeded(input []int)*Traversal{out:=[][]int{input};out[0]=append(out[0],1);return &Traversal{Backward:out}}
func Injected(input []int)*Traversal{out:=make([][]int,1);out[0]=input;out[0]=append(out[0],1);return &Traversal{Backward:out}}
func Rebound(input [][]int)*Traversal{out:=make([][]int,1);out=input;out[0]=append(out[0],1);return &Traversal{Backward:out}}
func Alias()*Traversal{out:=make([][]int,1);alias:=out;out[0]=append(out[0],1);return &Traversal{Backward:alias}}
var saved *Traversal
func Published()*Traversal{out:=make([][]int,1);saved=&Traversal{Backward:out};out[0]=append(out[0],1);return &Traversal{Backward:out}}
func Call(publish func(*Traversal))*Traversal{out:=make([][]int,1);publish(&Traversal{Backward:out});out[0]=append(out[0],1);return &Traversal{Backward:out}}
func Closure()*Traversal{out:=make([][]int,1);f:=func()*Traversal{return &Traversal{Backward:out}};out[0]=append(out[0],1);return f()}
func Deferred()*Traversal{out:=make([][]int,1);defer func(){out[0]=append(out[0],1)}();return &Traversal{Backward:out}}
func Pointee(input *int)*Traversal{out:=make([][]*int,1);out[0]=append(out[0],input);*out[0][0]=1;return &Traversal{Child:out}}
func IO()*Traversal{out:=make([][]int,1);out[0]=append(out[0],1);println("write");return &Traversal{Backward:out}}
func Callback(callback func())*Traversal{out:=make([][]int,1);out[0]=append(out[0],1);callback();return &Traversal{Backward:out}}
var counter int
func key()int{counter++;return 0}
func Argument()*Traversal{out:=make([][]int,1);out[key()]=append(out[0],1);return &Traversal{Backward:out}}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"TraversalFor", "Forward", "Value", "Mapping", "Parentheses"} {
		result := resultNamed(t, report, name)
		writes := findFunction(t, facts, name).Mutations
		t.Logf("%s: %+v; raw=%+v", name, result, writes)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses returned private rows", name)
		}
		if name == "Forward" {
			continue
		}
		if len(writes) != 2 || slices.ContainsFunc(writes, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) {
			t.Errorf("%s loses two raw local writes: %+v", name, writes)
		}
	}
	for _, name := range []string{"Borrowed", "Seeded", "Injected", "Rebound", "Alias", "Published", "Call", "Closure", "Deferred", "Pointee"} {
		result := resultNamed(t, report, name)
		writes := findFunction(t, facts, name).Mutations
		t.Logf("%s: %+v; raw=%+v", name, result, writes)
		if len(writes) == 0 || !slices.ContainsFunc(writes, func(m quality.Mutation) bool { return m.Provenance != quality.Local }) || len(result.Effects) == 0 {
			t.Errorf("%s hides shared or unknown writes", name)
		}
	}
	for name, kind := range map[string]quality.Kind{"Borrowed": quality.MutationEffect, "Published": quality.Global, "IO": quality.IO, "Callback": quality.UnknownEffect, "Argument": quality.Global} {
		result := resultNamed(t, report, name)
		t.Logf("%s: %+v", name, result)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
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
		t.Fatal("saved returned-entry facts change evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.TraversalFor", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(resultNamed(t, overridden, "Forward").Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
		t.Fatal("returned-entry ownership hides explicit helper policy")
	}
}

// The returned wrapper does not copy row storage. Shared spare capacity makes
// the control's append observable, while fresh construction cannot change it.
func TestReturnedEntriesStorageContract(t *testing.T) {
	type traversal struct{ backward [][]int }
	input := make([]int, 1, 4)
	input[0] = 7
	shared := [][]int{input}
	shared[0] = append(shared[0], 9)
	borrowed := &traversal{backward: shared}
	borrowed.backward[0][0] = 8
	if input[0] != 8 || input[:cap(input)][1] != 9 {
		t.Fatal("shared returned rows must expose writes")
	}
	private := make([][]int, 1)
	private[0] = append(private[0], input...)
	fresh := &traversal{backward: private}
	fresh.backward[0][0] = 10
	if input[0] != 8 {
		t.Fatal("fresh returned rows share input storage")
	}
}
