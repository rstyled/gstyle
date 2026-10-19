package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPlainJSONMarshalKeepsCallbacksAndWritesConservative(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "encoding/json"
type Record struct { Name string; Values []int; Lookup map[string]int; Next *Record }
type Alias = Record
type Stringer int
func(Stringer) String()string{println("not a JSON callback");return "value"}
func UnrelatedMethod(){json.Marshal(Stringer(1))}
type Custom int
func(Custom) MarshalJSON()([]byte,error){println("callback");return []byte("1"),nil}
type Pointer int
func(*Pointer) MarshalJSON()([]byte,error){println("callback");return []byte("1"),nil}
type Key string
func(Key) MarshalText()([]byte,error){println("key callback");return []byte("key"),nil}
type Open struct{Next *Open; Value any}
func Plain(){json.Marshal(&Alias{})}
func Scalars(){json.Marshal([]int{1,2})}
func Shared(v *Record){json.Marshal(v)}
func Callback(){json.Marshal(Custom(1))}
func PointerCallback(){json.Marshal(new(Pointer))}
func Nested(){json.Marshal(struct{Values []Custom}{})}
func MapKey(){json.Marshal(map[Key]int{})}
func Embedded(){json.Marshal(struct{Custom}{})}
func Raw(){json.Marshal(json.RawMessage("{}"))}
func Dynamic(v any){json.Marshal(v)}
func RecursiveOpen(){json.Marshal(new(Open))}
func Generic[T any](v T){json.Marshal(v)}
func Decode(v *Record){json.Unmarshal([]byte("{}"),v)}
func wrap(v any){json.Marshal(v)}
func chain(v any){wrap(v)}
func Wrapped(){chain(&Record{})}
func Mixed(){wrap(&Record{});wrap(Custom(1))}
func ReverseMixed(){wrap(Custom(1));wrap(&Record{})}
func Rebound(v any){v=Custom(1);json.Marshal(v)}
func ReboundCaller(){Rebound(&Record{})}

`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Plain", "Scalars", "Shared", "Wrapped", "UnrelatedMethod"} {
		got := resultNamed(t, report, name)
		if !got.Complete || len(got.Effects) != 0 {
			t.Fatalf("%s plain value: %+v", name, got)
		}
	}
	for _, name := range []string{"Callback", "PointerCallback", "Nested", "MapKey", "Embedded", "Raw", "Dynamic", "RecursiveOpen", "Generic", "Decode", "Mixed", "ReverseMixed", "ReboundCaller"} {
		got := resultNamed(t, report, name)
		if got.Complete || len(got.Effects) != 1 || got.Effects[0].Kind != quality.UnknownEffect {
			t.Fatalf("%s lost uncertainty: %+v", name, got)
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
		t.Fatal("saved JSON proof changes evaluation", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.UnknownEffect} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "encoding/json.Marshal", Kind: kind})
		changed, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		got := resultNamed(t, changed, "Plain")
		if len(got.Effects) != 1 || got.Effects[0].Kind != kind {
			t.Fatalf("policy override %s lost: %+v", kind, got)
		}
	}
}
