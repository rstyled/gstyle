package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestAppendIntDestinationOwnership(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "strconv"
// Mirrors domain/ccgraph.neighborKey's private key construction.
func Key(parts [][2]int) string {
 var key []byte
 for _,part:=range parts {
  key=strconv.AppendInt(key,int64(part[0]),10)
  key=append(key,':')
  key=strconv.AppendInt(key,int64(part[1]),10)
  key=append(key,';')
 }
 return string(key)
}
func Forward(parts [][2]int)string{return Key(parts)}
func Local(){key:=make([]byte,0,32);key=strconv.AppendInt((key),1,10);key=append(key,':')}
func Shared(key []byte){strconv.AppendInt(key,1,10)}
func SharedSelf(key []byte){key=strconv.AppendInt(key,1,10);key=append(key,':')}
func Rebound(input []byte){key:=make([]byte,0,32);key=input;key=strconv.AppendInt(key,1,10);key=append(key,':')}
func Mixed(input []byte,flag bool){var key []byte;if flag{key=input};key=strconv.AppendInt(key,1,10);key=append(key,':')}
func Dynamic(encode func([]byte,int64,int)[]byte){var key []byte;key=encode(key,1,10);key=append(key,':')}
type Encoder struct{}
var shared=[]byte{0}
func(Encoder)AppendInt([]byte,int64,int)[]byte{return shared}
func Shadow(){strconv:=Encoder{};var key []byte;key=strconv.AppendInt(key,1,10);key=append(key,':')}
func IO(){var key []byte;key=strconv.AppendInt(key,1,10);key=append(key,':');println("write")}
func Callback(value func()int64){var key []byte;key=strconv.AppendInt(key,value(),10);key=append(key,':')}
var counter int64
func next()int64{counter++;return counter}
func Argument(){var key []byte;key=strconv.AppendInt(key,next(),10);key=append(key,':')}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Key", "Forward", "Local"} {
		result := resultNamed(t, report, name)
		fact := findFunction(t, facts, name)
		t.Logf("%s: %+v; writes=%+v; calls=%+v", name, result, fact.Mutations, fact.Calls)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses private destination ownership", name)
		}
		for _, mutation := range fact.Mutations {
			if mutation.Provenance != quality.Local {
				t.Errorf("%s loses raw local write: %+v", name, mutation)
			}
		}
	}
	if writes := findFunction(t, facts, "Key").Mutations; len(writes) != 6 {
		t.Errorf("key loses its two header and four builtin append writes: %+v", writes)
	}
	for name, kind := range map[string]quality.Kind{
		"Shared": quality.MutationEffect, "SharedSelf": quality.MutationEffect,
		"Rebound": quality.MutationEffect, "Mixed": quality.MutationEffect,
		"Dynamic": quality.UnknownEffect, "Shadow": quality.UnknownEffect,
		"IO": quality.IO, "Callback": quality.UnknownEffect, "Argument": quality.Global,
	} {
		result := resultNamed(t, report, name)
		fact := findFunction(t, facts, name)
		t.Logf("%s: %+v; writes=%+v; calls=%+v", name, result, fact.Mutations, fact.Calls)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses independent %s evidence", name, kind)
		}
	}
	for _, name := range []string{"Key", "Local", "Shared", "SharedSelf", "Rebound", "Mixed"} {
		for _, call := range findFunction(t, facts, name).Calls {
			if call.Callee == "strconv.AppendInt" && (call.MutationTarget == "" || (call.MutationTarget == quality.Local) != (name == "Key" || name == "Local")) {
				t.Errorf("%s has wrong typed destination: %+v", name, call)
			}
		}
	}
	for name, provenance := range map[string]quality.Provenance{
		"SharedSelf": quality.External, "Rebound": quality.Unknown,
		"Mixed": quality.Unknown, "Dynamic": quality.Unknown, "Shadow": quality.Unknown,
	} {
		writes := findFunction(t, facts, name).Mutations
		if len(writes) < 3 || writes[len(writes)-1].Root != "key" || writes[len(writes)-1].Provenance != provenance {
			t.Errorf("%s loses its raw backing write: %+v", name, writes)
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
		t.Fatal("saved AppendInt facts change evaluation", err)
	}
	for _, callee := range []string{"strconv.AppendInt", "example.com/quality-repro.Key"} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: callee, Kind: quality.IO})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(resultNamed(t, overridden, "Forward").Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
			t.Errorf("AppendInt ownership hides explicit %s policy", callee)
		}
	}
}

func TestAppendIntSharedCapacityControl(t *testing.T) {
	for _, value := range []int64{0, -1, 127, -1 << 63, 1<<63 - 1} {
		for _, base := range []int{2, 10, 36} {
			storage := make([]byte, 0, 128)
			result := strconv.AppendInt(storage, value, base)
			if len(result) == 0 || &result[0] != &storage[:cap(storage)][0] {
				t.Fatalf("shared-capacity control does not reuse storage: value=%d base=%d", value, base)
			}
			result[0] = '!'
			if storage[:cap(storage)][0] != '!' {
				t.Fatal("shared destination write is not observable")
			}
		}
	}
}
