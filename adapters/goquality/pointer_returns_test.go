package goquality

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPointerReturnOwnershipKeepsEffects(t *testing.T) {
	source := `package repro
type State struct{ Count int; Child *State }
var global *State
var counter int
func fresh()*State{return new(State)}
func borrow(value *State)*State{return value}
func composedFresh()*State{return fresh()}
func composedBorrow(value *State)*State{return borrow(value)}
func ioFresh()*State{println("io");return new(State)}
func ioBorrow(value *State)*State{println("io");return value}
func globalFresh()*State{counter++;return new(State)}
func callbackFresh(run func())*State{run();return new(State)}
func genericPtr[T any](value T)*T{return &value}
func Fresh(){local:=fresh();local.Count++}
func Borrowed(value *State){local:=borrow(value);local.Count++}
func ComposedFresh(){local:=composedFresh();local.Count++}
func ComposedBorrowed(value *State){local:=composedBorrow(value);local.Count++}
func LocalInput(){local:=new(State);result:=borrow(local);result.Count++}
func FreshAlias(){local:=fresh();alias:=local;alias.Count++}
func IOFresh(){local:=ioFresh();local.Count++}
func IOBorrowed(value *State){local:=ioBorrow(value);local.Count++}
func GlobalFresh(){local:=globalFresh();local.Count++}
func Argument(){local:=borrow(globalFresh());local.Count++}
func Callback(run func()){local:=callbackFresh(run);local.Count++}
func GenericScalar(){local:=genericPtr(1);*local=2}
func GenericChild(value *State){local:=genericPtr(value);(*local).Count++}
func mixed(value *State,flag bool)*State{if flag{return new(State)};return value}
func Mixed(value *State){local:=mixed(value,true);local.Count++}
func rebound(value *State)*State{value=global;return value}
func Rebound(value *State){local:=rebound(value);local.Count++}
func published(value *State)*State{global=value;return value}
func Published(){local:=published(new(State));local.Count++}
func Addressed(value *State)*State{slot:=&value;*slot=global;return value}
func AddressCaller(value *State){local:=Addressed(value);local.Count++}
func Named()(result *State){result=new(State);return}
func NamedCaller(){local:=Named();local.Count++}
func Dynamic(get func()*State){local:=get();local.Count++}
func PublishFresh(){local:=fresh();global=local;local.Count++}
func PublishAlias(){local:=fresh();alias:=local;global=alias;local.Count++}
func PublishInput(){input:=new(State);global=input;local:=borrow(input);local.Count++}
func Captured(){local:=fresh();_ =func(){global=local};local.Count++}
func Deferred(){local:=fresh();defer published(local);local.Count++}
func Async(){local:=fresh();go published(local);local.Count++}
func child(value *State,flag bool)*State{if flag{return &State{Child:value}};return &State{}}
func SharedChild(value *State){local:=child(value,true);local.Child.Count++}
func Mutate(value *State){local:=borrow(value);local.Count++}
func ForwardOwned(){Mutate(new(State))}
func ForwardShared(value *State){Mutate(value)}
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
	for _, name := range []string{"Fresh", "ComposedFresh", "LocalInput", "FreshAlias", "GenericScalar", "ForwardOwned"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s must retain private storage: %+v", name, result)
		}
	}
	for _, name := range []string{"Borrowed", "ComposedBorrowed", "ForwardShared"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect {
			t.Errorf("%s must classify its caller write: %+v", name, result)
		}
	}
	for _, name := range []string{"Mixed", "Rebound", "Published", "AddressCaller", "NamedCaller", "Dynamic", "PublishFresh", "PublishAlias", "PublishInput", "Captured", "Deferred", "Async", "SharedChild", "GenericChild"} {
		function := findFunction(t, facts, name)
		write := function.Mutations[len(function.Mutations)-1]
		if write.Provenance == quality.Local {
			t.Errorf("%s loses unproved or shared storage: %+v", name, write)
		}
	}
	for name, kind := range map[string]quality.Kind{"IOFresh": quality.IO, "IOBorrowed": quality.IO, "GlobalFresh": quality.Global, "Argument": quality.Global, "Callback": quality.UnknownEffect} {
		result := resultNamed(t, report, name)
		if !pointerEffectKind(result, kind) {
			t.Errorf("%s loses %s call effects: %+v", name, kind, result)
		}
		function := findFunction(t, facts, name)
		write := function.Mutations[len(function.Mutations)-1]
		if name != "IOBorrowed" && write.Provenance != quality.Local {
			t.Errorf("%s confuses call effects and returned ownership: %+v", name, write)
		}
		if name == "IOBorrowed" && write.Provenance != quality.External {
			t.Errorf("IOBorrowed loses its external write: %+v", write)
		}
	}
	data, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var replay []quality.Function
	if err := json.Unmarshal(data, &replay); err != nil {
		t.Fatal(err)
	}
	restored, err := quality.Evaluate(replay, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(restored, report) {
		t.Fatal("pointer-return facts change after JSON replay", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "example.com/quality-repro.borrow", Kind: kind})
		overridden, err := quality.Evaluate(replay, config)
		if err != nil {
			t.Fatal(err)
		}
		if result := resultNamed(t, overridden, "Borrowed"); !pointerEffectKind(result, quality.MutationEffect) || kind != quality.None && !pointerEffectKind(result, kind) {
			t.Errorf("%s override loses separate call/write evidence: %+v", kind, result)
		}
	}
}

func pointerEffectKind(result quality.FunctionResult, kind quality.Kind) bool {
	for _, effect := range result.Effects {
		if effect.Kind == kind {
			return true
		}
	}
	return false
}

func TestPointerReturnDepthAndRecursionStayUnknown(t *testing.T) {
	var source strings.Builder
	source.WriteString("package repro\ntype State struct{Count int}\nfunc leaf()*State{return new(State)}\n")
	for index := range 12 {
		next := "leaf()"
		if index < 11 {
			next = fmt.Sprintf("step%d()", index+1)
		}
		fmt.Fprintf(&source, "func step%d()*State{return %s}\n", index, next)
	}
	source.WriteString("func Deep(){local:=step0();local.Count++}\nfunc recursive()*State{return recursive()}\nfunc Recursive(){local:=recursive();local.Count++}\n")
	root := qualitySourceModule(t, []byte(source.String()))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Deep", "Recursive"} {
		function := findFunction(t, facts, name)
		if function.Mutations[0].Provenance != quality.Unknown {
			t.Errorf("%s exceeds the proven composition bound: %+v", name, function.Mutations)
		}
	}
}
