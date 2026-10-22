package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestClosedMemoryJSONDecodeKeepsMutationAndPolicy(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"encoding/json";"fmt";"io";"strings")
type Parameter struct{Role,Type string;Variadic bool}
// Mirrors domain/ccgraph.Parameter.UnmarshalJSON in the master self-scan.
func(p *Parameter)UnmarshalJSON(data []byte)error{
 var wire struct{Role,Type *string;Variadic *bool}
 d:=json.NewDecoder(bytes.NewReader(data));d.DisallowUnknownFields()
 if err:=d.Decode(&wire);err!=nil{return err}
 if wire.Role==nil||wire.Type==nil||wire.Variadic==nil{return fmt.Errorf("missing tuple")}
 *p=Parameter{*wire.Role,*wire.Type,*wire.Variadic};return nil
}
type Document struct{Values []string}
func Local(raw string){d:=json.NewDecoder(strings.NewReader(raw));d.UseNumber();var value Document;d.Decode(&value)}
func Direct(raw []byte){var value Document;json.NewDecoder(bytes.NewReader(raw)).Decode(&value)}
func Shared(raw string,value *Document){d:=json.NewDecoder(strings.NewReader(raw));d.Decode(value)}
func Forward(raw string){Local(raw)}
func Input(r io.Reader){var value Document;json.NewDecoder(r).Decode(&value)}
func Existing(d *json.Decoder){var value Document;d.Decode(&value)}
func Rebound(r io.Reader){d:=json.NewDecoder(strings.NewReader("{}"));d=json.NewDecoder(r);var value Document;d.Decode(&value)}
var saved *json.Decoder
func Escaped(raw string){d:=json.NewDecoder(strings.NewReader(raw));saved=d;var value Document;d.Decode(&value)}
func Reset(raw string){r:=strings.NewReader(raw);d:=json.NewDecoder(r);r.Reset("{}");var value Document;d.Decode(&value)}
func Expression(raw string){var value Document;(*json.Decoder).Decode(json.NewDecoder(strings.NewReader(raw)),&value)}
func Indirect(raw string){var value Document;decode:=json.NewDecoder(strings.NewReader(raw)).Decode;decode(&value)}
func Async(raw string){d:=json.NewDecoder(strings.NewReader(raw));var value Document;go d.Decode(&value)}
func Captured(raw string){d:=json.NewDecoder(strings.NewReader(raw));var value Document;func(){d.Decode(&value)}()}
type Custom string
func(*Custom)UnmarshalJSON([]byte)error{println("callback");return nil}
type Text string
func(*Text)UnmarshalText([]byte)error{println("callback");return nil}
func Callback(raw string){var value Custom;json.NewDecoder(strings.NewReader(raw)).Decode(&value)}
func TextCallback(raw string){var value Text;json.NewDecoder(strings.NewReader(raw)).Decode(&value)}
func Nested(raw string){var value struct{Values []*Custom};json.NewDecoder(strings.NewReader(raw)).Decode(&value)}
func Open(raw string){var value any;json.NewDecoder(strings.NewReader(raw)).Decode(&value)}
var counter int
func next()string{counter++;return "{}"}
func ArgumentEffect(){var value Document;json.NewDecoder(strings.NewReader(next())).Decode(&value)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Direct", "Shared", "Forward"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect {
			t.Errorf("%s writes only memory, want one mutation: %+v", name, result)
		}
	}
	parameter := resultNamed(t, report, "Parameter.UnmarshalJSON")
	if !parameter.Complete || len(parameter.Effects) != 2 || slices.ContainsFunc(parameter.Effects, func(e quality.Effect) bool { return e.Kind != quality.MutationEffect }) {
		t.Errorf("self-scan tuple decode retains both writes without unknown callbacks: %+v", parameter)
	}
	for _, name := range []string{"Input", "Existing", "Rebound", "Escaped", "Reset", "Expression", "Indirect", "Async", "Captured", "Callback", "TextCallback", "Nested", "Open"} {
		result := resultNamed(t, report, name)
		if result.Complete || !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.UnknownEffect }) {
			t.Errorf("%s loses decode uncertainty: %+v", name, result)
		}
	}
	argument := resultNamed(t, report, "ArgumentEffect")
	if !slices.ContainsFunc(argument.Effects, func(e quality.Effect) bool { return e.Kind == quality.Global }) {
		t.Errorf("argument effect lost: %+v", argument)
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
		t.Fatal("saved JSON decode evidence changed", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.UnknownEffect, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "encoding/json.Decoder.Decode", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		result := resultNamed(t, overridden, "Forward")
		if kind == quality.None {
			if len(result.Effects) != 0 {
				t.Errorf("pure override lost: %+v", result)
			}
			continue
		}
		if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("%s override lost: %+v", kind, result)
		}
	}
}
