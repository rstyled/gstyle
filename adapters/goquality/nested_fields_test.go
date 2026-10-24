package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestNestedFieldAccessKeepsPrivateStorage(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "slices"
type Flags struct{ Unknown bool }
type record struct{ local Flags; targets []string }
func targets()[]string{return []string{"target"}}
// Mirrors domain/callflow.describeFrames: nested flag update then private append.
func Local()record{out:=record{local:Flags{Unknown:false}};out.local.Unknown=out.local.Unknown||true;out.targets=append(out.targets,targets()...);return out}
func Forward()record{return Local()}
func Sorted()record{out:=record{targets:[]string{"b","a"}};out.local.Unknown=true;slices.Sort(out.targets);return out}
func Parenthesized()record{out:=record{};((out).local).Unknown=true;out.targets=append(out.targets,"a");return out}
func Shared(out *record){out.local.Unknown=true;out.targets=append(out.targets,"a")}
func Borrowed(input []string)record{out:=record{targets:input};out.local.Unknown=true;slices.Sort(out.targets);return out}
func Rebound(input []string)record{out:=record{};out.local.Unknown=true;out.targets=input;slices.Sort(out.targets);return out}
var published *record
func Published()record{out:=record{};out.local.Unknown=true;published=&out;out.targets=append(out.targets,"a");return out}
func Alias()record{out:=record{};out.local.Unknown=true;other:=&out;out.targets=append(out.targets,"a");_=other;return out}
func Address()record{out:=record{};flag:=&out.local;flag.Unknown=true;out.targets=append(out.targets,"a");return out}
func (f *Flags) Set(){f.Unknown=true}
func Method()record{out:=record{};out.local.Set();out.targets=append(out.targets,"a");return out}
func MethodValue()record{out:=record{};set:=out.local.Set;set();out.targets=append(out.targets,"a");return out}
func Closure()record{out:=record{};func(){published=&out}();out.local.Unknown=true;out.targets=append(out.targets,"a");return out}
func Deferred()record{out:=record{};defer func(){out.targets=append(out.targets,"late")}();out.local.Unknown=true;out.targets=append(out.targets,"a");return out}
func Pointee(input *int){out:=struct{local Flags;items []*int}{items:[]*int{input}};out.local.Unknown=true;*out.items[0]=1}
func IO()record{out:=Local();println("write");return out}
var counter int
func Global()record{out:=record{};out.local.Unknown=true;out.targets=append(out.targets,argument());return out}
func argument()string{counter++;return "a"}
func Unknown(next func()string)record{out:=record{};out.local.Unknown=true;out.targets=append(out.targets,next());return out}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Forward", "Sorted", "Parenthesized"} {
		result := resultNamed(t, report, name)
		t.Logf("%s result=%+v facts=%+v", name, result, findFunction(t, facts, name))
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses private storage after nested field access: %+v", name, result)
		}
	}
	for name, kind := range map[string]quality.Kind{"Shared": quality.MutationEffect, "Borrowed": quality.MutationEffect, "Rebound": quality.MutationEffect, "Published": quality.UnknownEffect, "Alias": quality.UnknownEffect, "Address": quality.UnknownEffect, "Method": quality.UnknownEffect, "MethodValue": quality.UnknownEffect, "Closure": quality.UnknownEffect, "Deferred": quality.UnknownEffect, "Pointee": quality.UnknownEffect, "IO": quality.IO, "Global": quality.Global, "Unknown": quality.UnknownEffect} {
		result := resultNamed(t, report, name)
		t.Logf("%s result=%+v facts=%+v", name, result, findFunction(t, facts, name))
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses %s evidence: %+v", name, kind, result)
		}
	}
	local := findFunction(t, facts, "Local")
	if len(local.Mutations) != 3 {
		t.Errorf("raw nested/header/append writes missing: %+v", local.Mutations)
	}
	for _, write := range local.Mutations {
		if write.Provenance != quality.Local {
			t.Errorf("private raw write: %+v", write)
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
		t.Fatal("saved nested field facts change evaluation", err)
	}
	for _, callee := range []string{"example.com/quality-repro.targets", "example.com/quality-repro.Local"} {
		for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
			config := quality.DefaultConfig()
			config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: kind})
			changed, err := quality.Evaluate(restored, config)
			if err != nil {
				t.Fatal(err)
			}
			result := resultNamed(t, changed, "Forward")
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("nested field ownership hides %s %s policy: %+v", callee, kind, result)
			}
		}
	}
}
