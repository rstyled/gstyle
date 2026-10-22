package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestFreshMemoryXMLDecodeKeepsMutationAndPolicy(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"encoding/xml";"io";"strings")
type Node struct{XMLName xml.Name; Row int `+"`xml:\"row,attr\"`"+`;Children []Node `+"`xml:\",any\"`"+`}
type Document struct{Root Node `+"`xml:\"source>node\"`"+`}
type Alias = Document
func Local(raw []byte){var parsed Document;xml.NewDecoder(bytes.NewReader(raw)).Decode(&parsed)}
func Text(raw string){var parsed Document;xml.NewDecoder(strings.NewReader(raw)).Decode(&parsed)}
func Aliased(raw []byte){var parsed Alias;xml.NewDecoder(bytes.NewReader(raw)).Decode(&parsed)}
func Shared(raw []byte, parsed *Document){xml.NewDecoder(bytes.NewReader(raw)).Decode(parsed)}
func Forward(raw []byte){Local(raw)}
func Input(r io.Reader){var parsed Document;xml.NewDecoder(r).Decode(&parsed)}
func Existing(d *xml.Decoder){var parsed Document;d.Decode(&parsed)}
func BufferDestination(d *xml.Decoder){var parsed bytes.Buffer;d.Decode(&parsed)}
func Expression(raw []byte){var parsed Document;(*xml.Decoder).Decode(xml.NewDecoder(bytes.NewReader(raw)),&parsed)}
func Configured(raw []byte, charset func(string,io.Reader)(io.Reader,error)){d:=xml.NewDecoder(bytes.NewReader(raw));d.CharsetReader=charset;var parsed Document;d.Decode(&parsed)}
type Custom struct{}
func(*Custom)UnmarshalXML(*xml.Decoder,xml.StartElement)error{println("callback");return nil}
type Attr string
func(*Attr)UnmarshalXMLAttr(xml.Attr)error{println("callback");return nil}
type TextValue string
func(*TextValue)UnmarshalText([]byte)error{println("callback");return nil}
func Callback(raw []byte){var parsed Custom;xml.NewDecoder(bytes.NewReader(raw)).Decode(&parsed)}
func Attribute(raw []byte){var parsed struct{Value Attr `+"`xml:\"v,attr\"`"+`};xml.NewDecoder(bytes.NewReader(raw)).Decode(&parsed)}
func TextCallback(raw []byte){var parsed struct{Value TextValue};xml.NewDecoder(bytes.NewReader(raw)).Decode(&parsed)}
func Nested(raw []byte){var parsed struct{Values []*Custom};xml.NewDecoder(bytes.NewReader(raw)).Decode(&parsed)}
func OpenValue(raw []byte, parsed any){xml.NewDecoder(bytes.NewReader(raw)).Decode(parsed)}
func Wrapped(raw []byte, parsed struct{Value any}){xml.NewDecoder(bytes.NewReader(raw)).Decode(&parsed)}
func Indirect(raw []byte){var parsed Document;decode:=xml.NewDecoder(bytes.NewReader(raw)).Decode;decode(&parsed)}
var counter int
func next()[]byte{counter++;return nil}
func ArgumentEffect(){var parsed Document;xml.NewDecoder(bytes.NewReader(next())).Decode(&parsed)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "Text", "Aliased", "Shared", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect || !result.Complete {
			t.Errorf("%s writes only memory, want one mutation: %+v", name, result)
		}
	}
	for _, name := range []string{"Input", "Existing", "BufferDestination", "Expression", "Configured", "Callback", "Attribute", "TextCallback", "Nested", "OpenValue", "Wrapped"} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
			t.Errorf("%s must retain IO uncertainty: %+v", name, result)
		}
	}
	indirect := resultNamed(t, report, "Indirect")
	if indirect.Complete || indirect.UnclassifiedCalls == 0 {
		t.Errorf("indirect decode loses uncertainty: %+v", indirect)
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
		t.Fatal("saved XML evidence changed", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "encoding/xml.Decoder.Decode", Kind: kind})
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
