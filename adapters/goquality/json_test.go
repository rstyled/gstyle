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
func IndentPlain(){json.MarshalIndent(&Alias{},"", "  ")}
func IndentCallback(){json.MarshalIndent(Custom(1),"", "  ")}
func IndentZeroCallback(){json.MarshalIndent(ZeroRecord{},"", "  ")}
func IndentDynamic(v any){json.MarshalIndent(v,"", "  ")}
func Scalars(){json.Marshal([]int{1,2})}
func Shared(v *Record){json.Marshal(v)}
func Callback(){json.Marshal(Custom(1))}
func PointerCallback(){json.Marshal(new(Pointer))}
func Nested(){json.Marshal(struct{Values []Custom}{})}
func MapKey(){json.Marshal(map[Key]int{})}
func Embedded(){json.Marshal(struct{Custom}{})}
func Raw(){json.Marshal(json.RawMessage("{}"))}
func RawNested(){json.Marshal(struct{Data json.RawMessage}{})}
func RawDecoded(data []byte)json.RawMessage{var result json.RawMessage;json.Unmarshal(data,&result);return result}
func CheckedRecord(data []byte)bool{var result Record;json.Unmarshal(data,&result);return result.Name!=""}
func CheckedMap(data []byte)bool{var result map[string]json.RawMessage;json.Unmarshal(data,&result);return result!=nil}
func CheckedPointer(data []byte)bool{var result *Record;json.Unmarshal(data,&result);return result!=nil&&result.Name!=""}
func ScalarLoop(data []byte)bool{var result string;for i:=0;i<2;i++{json.Unmarshal(data,&result)};return result!=""}
func ScalarLabel(data []byte)bool{var result string;again:json.Unmarshal(data,&result);if false{goto again};return result!=""}
func ReturnLoop(data []byte)Record{var result Record;for i:=0;i<2;i++{json.Unmarshal(data,&result)};return result}
func DecodeLoop(data []byte){var result Record;for i:=0;i<2;i++{json.Unmarshal(data,&result)}}
func DecodeLabel(data []byte){var result Record;again:json.Unmarshal(data,&result);if false{goto again}}
func DecodeFieldAddress(data []byte){var result Record;p:=&result.Name;json.Unmarshal(data,&result);_ =p}
func DecodeFieldWrite(data []byte){var result Record;json.Unmarshal(data,&result);result.Name="new"}
func DecodeFieldView(data []byte){var result Record;json.Unmarshal(data,&result);_ =result.Values[:]}
func DecodeCaptureRecord(data []byte){var result Record;f:=func(){_ =result};json.Unmarshal(data,&result);f()}
func DecodeAsyncRead(data []byte){var result Record;json.Unmarshal(data,&result);go func(Record){}(result)}
func RawShared(data []byte, result *json.RawMessage){json.Unmarshal(data,result)}
func RawPreloaded(data []byte)json.RawMessage{result:=json.RawMessage(data);json.Unmarshal(data,&result);return result}
func ScalarChecked(data []byte)bool{var result string;json.Unmarshal(data,&result);return result!=""}
func ScalarInitialized(data []byte)bool{result:="";json.Unmarshal(data,&result);return result!=""}
func ScalarRead(data []byte){var result int;json.Unmarshal(data,&result);_ = result+1}
func ScalarAlias(data []byte){var result string;p:=&result;json.Unmarshal(data,&result);_ = p}
func ScalarCapture(data []byte){var result string;f:=func(){_ = result};json.Unmarshal(data,&result);f()}
type Scalar string
var scalarPointer *Scalar
func(s *Scalar) Expose(){scalarPointer=s}
func ScalarMethod(data []byte){var result Scalar;result.Expose();json.Unmarshal(data,&result)}
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
	for _, name := range []string{"ScalarLoop", "ScalarLabel", "ReturnLoop", "CheckedRecord", "CheckedMap", "CheckedPointer", "Raw", "RawNested", "RawDecoded", "ScalarChecked", "ScalarInitialized", "ScalarRead", "IgnoredFields", "DecodedIgnored", "ZeroOrdinary", "ZeroNamed", "DecodedZero", "OmitZeroScalar", "NonCallbackZero", "Plain", "IndentPlain", "Scalars", "Shared", "Wrapped", "UnrelatedMethod", "Decoded", "DecodedSlice", "DecodedPointer", "WrappedDecode"} {
		got := resultNamed(t, report, name)
		if !got.Complete || len(got.Effects) != 0 {
			t.Fatalf("%s plain value: %+v", name, got)
		}
	}
	for _, name := range []string{"DecodeLoop", "DecodeLabel", "DecodeFieldAddress", "DecodeFieldWrite", "DecodeFieldView", "DecodeCaptureRecord", "DecodeAsyncRead", "IndentCallback", "IndentZeroCallback", "IndentDynamic", "IgnoredEmbedded", "DashName", "ZeroCallback", "Callback", "PointerCallback", "Nested", "MapKey", "Embedded", "RawShared", "RawPreloaded", "ScalarAlias", "ScalarCapture", "ScalarMethod", "Dynamic", "RecursiveOpen", "Generic", "Decode", "Mixed", "ReverseMixed", "ReboundCaller", "DecodeCallback", "DecodeMapKey", "PreloadedDecode", "AliasedDecode", "PublishedDecode", "CapturedDecode", "DeferredDecode", "ConcurrentDecode", "ReusedDecode"} {
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
		for _, name := range []string{"Plain", "IndentPlain"} {
			got := resultNamed(t, changed, name)
			if len(got.Effects) != 1 || got.Effects[0].Kind != kind {
				t.Fatalf("policy override %s lost: %+v", kind, got)
			}
		}
	}
}
