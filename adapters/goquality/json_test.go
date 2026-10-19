package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPlainJSONKeepsCallbacksAndWritesConservative(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "encoding/json"
type Record struct { Name string; Values []int; Lookup map[string]int; Next *Record }
type Alias = Record
type Stringer int
func(Stringer) String()string{println("not a JSON callback");return "value"}
func UnrelatedMethod(){json.Marshal(Stringer(1))}
type Custom int
func(Custom) MarshalJSON()([]byte,error){println("callback");return []byte("1"),nil}
func(*Custom) UnmarshalJSON([]byte)error{println("decode callback");return nil}
type Pointer int
func(*Pointer) MarshalJSON()([]byte,error){println("callback");return []byte("1"),nil}
type Key string
func(Key) MarshalText()([]byte,error){println("key callback");return []byte("key"),nil}
func(*Key) UnmarshalText([]byte)error{println("key decode callback");return nil}
type Open struct{Next *Open; Value any}
type Ignored struct{Callback Custom `+"`json:\"-\"`"+`; private Custom}
func IgnoredFields(){json.Marshal(Ignored{})}
func DecodedIgnored(data []byte)Ignored{var result Ignored;json.Unmarshal(data,&result);return result}
func IgnoredEmbedded(){json.Marshal(struct{Custom `+"`json:\"-\"`"+`}{})}
func DashName(){json.Marshal(struct{Value Custom `+"`json:\"-,\"`"+`}{})}
type Zero int
func(*Zero) IsZero()bool{println("zero callback");return false}
type ZeroRecord struct{Value Zero `+"`json:\",omitzero\"`"+`}
func ZeroCallback(){json.Marshal(ZeroRecord{})}
func ZeroOrdinary(){json.Marshal(struct{Value Zero}{})}
func ZeroNamed(){json.Marshal(struct{Value Zero `+"`json:\"omitzero\"`"+`}{})}
func DecodedZero(data []byte)ZeroRecord{var result ZeroRecord;json.Unmarshal(data,&result);return result}
func OmitZeroScalar(){json.Marshal(struct{Value int `+"`json:\",omitzero\"`"+`}{})}
type WrongZero int
func(WrongZero) IsZero()int{return 0}
func NonCallbackZero(){json.Marshal(struct{Value WrongZero `+"`json:\",omitzero\"`"+`}{})}
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
func Decoded(data []byte)(Alias,error){var result Alias;err:=json.Unmarshal(data,&result);return result,err}
func DecodedSlice(data []byte)([]Record,error){var result []Record;err:=json.Unmarshal(data,&result);return result,err}
func DecodedPointer(data []byte)(*Pointer,error){var result *Pointer;err:=json.Unmarshal(data,&result);return result,err}
func WrappedDecode(){Decoded(nil)}
func DecodeCallback(data []byte)Custom{var result Custom;json.Unmarshal(data,&result);return result}
func DecodeMapKey(data []byte)map[Key]int{var result map[Key]int;json.Unmarshal(data,&result);return result}
func PreloadedDecode(data []byte,values []int)Record{result:=Record{Values:values};json.Unmarshal(data,&result);return result}
func AliasedDecode(data []byte)Record{var result Record;alias:=&result;json.Unmarshal(data,&result);_ = alias;return result}
var escaped *Record
func PublishedDecode(data []byte)Record{var result Record;escaped=&result;json.Unmarshal(data,&result);return result}
func CapturedDecode(data []byte)func()Record{var result Record;json.Unmarshal(data,&result);return func()Record{return result}}
func DeferredDecode(data []byte)Record{var result Record;defer json.Unmarshal(data,&result);return result}
func ConcurrentDecode(data []byte)Record{var result Record;go json.Unmarshal(data,&result);return result}
func ReusedDecode(data []byte)Record{var result Record;json.Unmarshal(data,&result);json.Unmarshal(data,&result);return result}

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
	for _, name := range []string{"IgnoredFields", "DecodedIgnored", "ZeroOrdinary", "ZeroNamed", "DecodedZero", "OmitZeroScalar", "NonCallbackZero", "Plain", "Scalars", "Shared", "Wrapped", "UnrelatedMethod", "Decoded", "DecodedSlice", "DecodedPointer", "WrappedDecode"} {
		got := resultNamed(t, report, name)
		if !got.Complete || len(got.Effects) != 0 {
			t.Fatalf("%s plain value: %+v", name, got)
		}
	}
	for _, name := range []string{"IgnoredEmbedded", "DashName", "ZeroCallback", "Callback", "PointerCallback", "Nested", "MapKey", "Embedded", "Raw", "Dynamic", "RecursiveOpen", "Generic", "Decode", "Mixed", "ReverseMixed", "ReboundCaller", "DecodeCallback", "DecodeMapKey", "PreloadedDecode", "AliasedDecode", "PublishedDecode", "CapturedDecode", "DeferredDecode", "ConcurrentDecode", "ReusedDecode"} {
		got := resultNamed(t, report, name)
		if got.Complete || !slices.ContainsFunc(got.Effects, func(e quality.Effect) bool { return e.Kind == quality.UnknownEffect }) {
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
