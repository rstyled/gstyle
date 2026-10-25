package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestMemoryFormattingKeepsMutationAndCallbacks(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"fmt";"io";"os";"strings")
type Alias=bytes.Buffer
type Custom struct{bytes.Buffer}
type Stringer int
func(Stringer)String()string{println("callback");return "value"}
type Formatter int
func(Formatter)Format(fmt.State,rune){println("callback")}
type Failure int
func(Failure)Error()string{println("callback");return "error"}
type GoStringer int
func(GoStringer)GoString()string{println("callback");return "value"}
type Pointer int
func(*Pointer)String()string{println("callback");return "value"}
func Local(){var out bytes.Buffer;fmt.Fprintf(&out,"%s %d","value",1)}
func Builder(){var out strings.Builder;fmt.Fprintln(&out,"value")}
func AliasWriter(){var out Alias;fmt.Fprint(&out,"value")}
func Shared(out *bytes.Buffer){fmt.Fprint(out,"value")}
func Borrowed(data []byte){out:=bytes.NewBuffer(data[:0]);fmt.Fprint(out,"value")}
func Replaced(data []byte){out:=new(bytes.Buffer);*out=*bytes.NewBuffer(data[:0]);fmt.Fprint(out,"value")}
func Wrapped(){var out Custom;fmt.Fprint(&out,"value")}
func Open(out io.Writer){fmt.Fprint(out,"value")}
func File(){fmt.Fprint(os.Stdout,"value")}
func StringCallback(){var out bytes.Buffer;fmt.Fprint(&out,Stringer(1))}
func FormatCallback(){var out bytes.Buffer;fmt.Fprint(&out,Formatter(1))}
func ErrorCallback(){var out bytes.Buffer;fmt.Fprint(&out,Failure(1))}
func GoStringCallback(){var out bytes.Buffer;fmt.Fprintf(&out,"%#v",GoStringer(1))}
func PointerCallback(){var out bytes.Buffer;var value Pointer;fmt.Fprint(&out,&value)}
func OpenArgument(value any){var out bytes.Buffer;fmt.Fprint(&out,value)}
func Spread(values ...any){var out bytes.Buffer;fmt.Fprint(&out,values...)}
var counter int
func next()int{counter++;return counter}
func ArgumentEffect(){var out bytes.Buffer;fmt.Fprint(&out,next())}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if result := resultNamed(t, report, "Builder"); !result.Complete || len(result.Effects) != 0 {
		t.Fatalf("private builder retains formatting effect: %+v", result)
	}
	for _, name := range []string{"Local", "AliasWriter", "Shared", "Borrowed", "Replaced"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect {
			t.Fatalf("%s lost buffer mutation: %+v", name, result)
		}
	}
	for _, name := range []string{"Wrapped", "Open", "File", "StringCallback", "FormatCallback", "ErrorCallback", "GoStringCallback", "PointerCallback", "OpenArgument", "Spread"} {
		result := resultNamed(t, report, name)
		found := false
		for _, effect := range result.Effects {
			found = found || effect.Kind == quality.IO
		}
		if !found {
			t.Fatalf("%s lost IO or callback uncertainty: %+v", name, result)
		}
	}
	found := false
	for _, effect := range resultNamed(t, report, "ArgumentEffect").Effects {
		found = found || effect.Kind == quality.Global
	}
	if !found {
		t.Fatal("format argument lost its global effect")
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
		t.Fatal("saved memory-writer evidence changed", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "fmt.Fprintf", Kind: quality.IO})
	changed, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, changed, "Local")
	if len(result.Effects) != 1 || result.Effects[0].Kind != quality.IO {
		t.Fatalf("format proof hid policy override: %+v", result)
	}
}
