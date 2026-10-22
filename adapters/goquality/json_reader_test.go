package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestClosedMemoryJSONTokens(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"encoding/json";"io";"strings")
func Local(raw string){d:=json.NewDecoder(strings.NewReader(raw));d.Token();d.More()}
func Bytes(raw []byte){d:=json.NewDecoder(bytes.NewReader(raw));d.Token();d.More()}
func Direct(){json.NewDecoder(strings.NewReader("{}")).Token()}
type Alias = json.Decoder
func Expression(){d:=json.NewDecoder(strings.NewReader("{}"));(*Alias).Token(d);(*Alias).More(d)}
func Forward(raw string){Local(raw)}
func Borrowed(d *json.Decoder){d.Token();d.More()}
func Input(r io.Reader){d:=json.NewDecoder(r);d.Token();d.More()}
func Rebound(r io.Reader){d:=json.NewDecoder(strings.NewReader("{}"));d=json.NewDecoder(r);d.Token();d.More()}
func AliasUse(){d:=json.NewDecoder(strings.NewReader("{}"));other:=d;other.Token();d.More()}
func Returned()*json.Decoder{d:=json.NewDecoder(strings.NewReader("{}"));d.Token();d.More();return d}
func Reset(r io.Reader){input:=strings.NewReader("{}");d:=json.NewDecoder(input);input.Reset("[]");d.Token();d.More()}
func Closure(){d:=json.NewDecoder(strings.NewReader("{}"));func(){d.Token()}();d.More()}
func Async(){d:=json.NewDecoder(strings.NewReader("{}"));go d.Token();d.More()}
func MethodValue(){d:=json.NewDecoder(strings.NewReader("{}"));token:=d.Token;token();d.More()}
func Decode(){d:=json.NewDecoder(strings.NewReader("{}"));var value any;d.Decode(&value);d.Token();d.More()}
var saved *json.Decoder
func Global(){d:=json.NewDecoder(strings.NewReader("{}"));saved=d;d.Token();d.More()}
var counter int
func next()string{counter++;return "{}"}
func ArgumentEffect(){json.NewDecoder(strings.NewReader(next())).Token()}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Bytes", "Direct", "Expression", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 0 || result.DensityMilli != 0 || !result.Complete {
			t.Errorf("%s reads closed memory, want no effects: %+v", name, result)
		}
	}
	for _, name := range []string{"Borrowed", "Input", "Rebound", "AliasUse", "Returned", "Reset", "Closure", "Async", "MethodValue", "Decode", "Global"} {
		result := resultNamed(t, report, name)
		var reads int
		for _, effect := range result.Effects {
			if effect.Kind == quality.IO {
				reads++
			}
		}
		if reads == 0 {
			t.Errorf("%s must retain IO uncertainty: %+v", name, result)
		}
		for _, call := range findFunction(t, facts, name).Calls {
			if call.LocalMemoryRead {
				t.Errorf("%s has an unproved memory read: %+v", name, call)
			}
		}
	}
	argument := resultNamed(t, report, "ArgumentEffect")
	if len(argument.Effects) != 1 || argument.Effects[0].Kind != quality.Global {
		t.Errorf("reader proof lost constructor argument effects: %+v", argument)
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
		t.Fatal("saved JSON reader proof changes evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes,
		quality.Prefix{Path: "encoding/json.Decoder.Token", Kind: quality.IO},
		quality.Prefix{Path: "encoding/json.Decoder.More", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, overridden, "Local")
	if len(result.Effects) != 2 || result.Effects[0].Kind != quality.IO || result.Effects[1].Kind != quality.IO {
		t.Fatalf("JSON reader proof hid policy override: %+v", result)
	}
}
