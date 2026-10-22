package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestFreshXMLDecodeWithBufferReaders(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"encoding/xml";"io")
type Document struct{Value string}
func Bytes(raw []byte){var parsed Document;xml.NewDecoder(bytes.NewBuffer(raw)).Decode(&parsed)}
func Text(raw string){var parsed Document;xml.NewDecoder(bytes.NewBufferString(raw)).Decode(&parsed)}
func Shared(raw []byte, parsed *Document){xml.NewDecoder(bytes.NewBuffer(raw)).Decode(parsed)}
func Forward(raw []byte){Bytes(raw)}
func Borrowed(input *bytes.Buffer){var parsed Document;xml.NewDecoder(input).Decode(&parsed)}
func Rebound(input *bytes.Buffer){r:=bytes.NewBuffer(nil);r=input;var parsed Document;xml.NewDecoder(r).Decode(&parsed)}
func Exposed(raw []byte){r:=bytes.NewBuffer(raw);r.Bytes();var parsed Document;xml.NewDecoder(r).Decode(&parsed)}
func Configured(raw []byte, charset func(string,io.Reader)(io.Reader,error)){d:=xml.NewDecoder(bytes.NewBuffer(raw));d.CharsetReader=charset;var parsed Document;d.Decode(&parsed)}
type Custom string
func(*Custom)UnmarshalText([]byte)error{println("callback");return nil}
func Callback(raw string){var parsed struct{Value Custom};xml.NewDecoder(bytes.NewBufferString(raw)).Decode(&parsed)}
var counter int
func data()[]byte{counter++;return nil}
func ArgumentEffect(){var parsed Document;xml.NewDecoder(bytes.NewBuffer(data())).Decode(&parsed)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Bytes", "Text", "Shared", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect || !result.Complete {
			t.Errorf("%s must retain only destination mutation: %+v", name, result)
		}
	}
	for _, name := range []string{"Borrowed", "Rebound", "Exposed", "Configured", "Callback"} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
			t.Errorf("%s must retain IO uncertainty: %+v", name, result)
		}
	}
	argument := resultNamed(t, report, "ArgumentEffect")
	if !slices.ContainsFunc(argument.Effects, func(e quality.Effect) bool { return e.Kind == quality.Global }) {
		t.Errorf("constructor argument effect lost: %+v", argument)
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
		t.Fatal("saved combined XML/buffer evidence changed", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "encoding/xml.Decoder.Decode", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, overridden, "Forward")
	if len(result.Effects) != 1 || result.Effects[0].Kind != quality.IO {
		t.Errorf("explicit IO override lost: %+v", result)
	}
}
