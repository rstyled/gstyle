package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestScalarJSONDecodeKeepsWritesAndCallbacks(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "encoding/json"
type Boundary struct{ Kind string; Child *string }
// Mirrors domain/callflow.callBoundary.entry's scalar field destination.
func (b *Boundary) Entry(data []byte){json.Unmarshal(data,&b.Kind)}
type Scalar int
type Alias = string
type Custom int
func(*Custom)UnmarshalJSON([]byte)error{println("callback");return nil}
type Text string
func(*Text)UnmarshalText([]byte)error{println("text callback");return nil}
type Embedded struct{Custom}
func Local(data []byte){var value int;json.Unmarshal(data,&value)}
func Named(data []byte,value *Scalar){json.Unmarshal(data,value)}
func Aliased(data []byte,value *Alias){json.Unmarshal(data,value)}
func Shared(data []byte,value *string){json.Unmarshal(data,value)}
func Forward(data []byte,value *string){Shared(data,value)}
func PrivateForward(data []byte){var value string;Shared(data,&value)}
func Child(data []byte,b Boundary){json.Unmarshal(data,b.Child)}
func Rebound(data []byte,input *int){var local int;p:=&local;p=input;json.Unmarshal(data,p)}
var shared int
func Global(data []byte){json.Unmarshal(data,&shared)}
var saved *int
func Escaped(data []byte){var value int;saved=&value;json.Unmarshal(data,&value)}
func AliasUse(data []byte){var value int;p:=&value;json.Unmarshal(data,&value);_ =p}
func Callback(data []byte,value *Custom){json.Unmarshal(data,value)}
func TextCallback(data []byte,value *Text){json.Unmarshal(data,value)}
func Promoted(data []byte,value *Embedded){json.Unmarshal(data,value)}
func Open(data []byte,value any){json.Unmarshal(data,value)}
func References(data []byte,value *[]int){json.Unmarshal(data,value)}
func Unknown(next func()*int){json.Unmarshal(nil,next())}
func RawWrite(data []byte,value *int){var local int;json.Unmarshal(data,&local);*value=2}
func IO(data []byte){var value int;json.Unmarshal(data,&value);println("io")}
var counter int
func next()[]byte{counter++;return []byte("1")}
func ArgumentEffect(){var value int;json.Unmarshal(next(),&value)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s loses private scalar destination: %+v", name, result)
		}
	}
	for _, name := range []string{"Entry", "Named", "Aliased", "Shared", "Forward", "PrivateForward", "Child", "Rebound", "Global", "Escaped", "AliasUse"} {
		result := resultNamed(t, report, name)
		if !result.Complete || !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.MutationEffect }) {
			t.Errorf("%s must classify scalar JSON write without unknown callbacks: %+v", name, result)
		}
	}
	for _, name := range []string{"Entry", "Shared", "Child", "Rebound", "Local"} {
		fact := findFunction(t, facts, name)
		t.Logf("%s: calls=%+v; mutations=%+v", name, fact.Calls, fact.Mutations)
		if len(fact.Calls) != 1 || !fact.Calls[0].MemoryWriter || len(fact.Calls[0].StorageArgs) != 2 {
			t.Errorf("%s loses typed scalar JSON evidence: %+v", name, fact)
		}
	}
	for _, name := range []string{"Callback", "TextCallback", "Promoted", "Open", "References", "Unknown"} {
		result := resultNamed(t, report, name)
		if result.Complete || !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.UnknownEffect }) {
			t.Errorf("%s loses callback/reference uncertainty: %+v", name, result)
		}
	}
	for _, name := range []string{"Callback", "TextCallback", "Promoted", "Open", "References"} {
		if call := findFunction(t, facts, name).Calls[0]; call.MemoryWriter {
			t.Errorf("%s receives an unproved scalar certificate: %+v", name, call)
		}
	}
	for name, kind := range map[string]quality.Kind{"RawWrite": quality.MutationEffect, "IO": quality.IO, "Global": quality.Global, "ArgumentEffect": quality.Global} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
			t.Errorf("%s loses independent %s: %+v", name, kind, result)
		}
	}
	writes := findFunction(t, facts, "RawWrite").Mutations
	if len(writes) != 1 || writes[0].Provenance != quality.External {
		t.Errorf("raw caller write changes: %+v", writes)
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
		t.Fatal("saved scalar JSON evidence changes evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.MutationEffect, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "encoding/json.Unmarshal", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"Local", "Entry", "Forward"} {
			result := resultNamed(t, overridden, name)
			if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
				t.Errorf("%s hides explicit %s policy: %+v", name, kind, result)
			}
		}
	}
}
