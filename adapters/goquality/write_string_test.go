package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestMemoryWriteStringKeepsMutationAndPolicy(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("bytes";"io";"os";"strings")
type Alias = bytes.Buffer
type Custom struct{bytes.Buffer}
func(*Custom)WriteString(text string)(int,error){println("callback");return len(text),nil}
func Local(){var out bytes.Buffer;io.WriteString(&out,"value")}
func Builder(){var out strings.Builder;io.WriteString(&out,"value")}
func Aliased(){var out Alias;io.WriteString(&out,"value")}
func Shared(out *bytes.Buffer){io.WriteString(out,"value")}
func Borrowed(data []byte){out:=bytes.NewBuffer(data[:0]);io.WriteString(out,"value")}
func Forward(){Local()}
func Wrapped(){var out Custom;io.WriteString(&out,"value")}
func Open(out io.Writer){io.WriteString(out,"value")}
func File(){io.WriteString(os.Stdout,"value")}
func Indirect(){var out bytes.Buffer;write:=io.WriteString;write(&out,"value")}
var counter int
func next()string{counter++;return "value"}
func ArgumentEffect(){var out bytes.Buffer;io.WriteString(&out,next())}
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
		t.Fatalf("private builder retains WriteString effect: %+v", result)
	}
	for _, name := range []string{"Local", "Aliased", "Shared", "Borrowed", "Forward"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect || !result.Complete {
			t.Errorf("%s writes only memory, want one mutation: %+v", name, result)
		}
	}
	for _, name := range []string{"Wrapped", "Open", "File"} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.IO }) {
			t.Errorf("%s must retain IO or callback uncertainty: %+v", name, result)
		}
	}
	indirect := resultNamed(t, report, "Indirect")
	if indirect.Complete || indirect.UnclassifiedCalls == 0 {
		t.Errorf("indirect write must retain dynamic call uncertainty: %+v", indirect)
	}
	argument := resultNamed(t, report, "ArgumentEffect")
	if !slices.ContainsFunc(argument.Effects, func(e quality.Effect) bool { return e.Kind == quality.Global }) {
		t.Errorf("memory writer loses argument effects: %+v", argument)
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
		t.Fatal("saved WriteString evidence changed", err)
	}
	for _, kind := range []quality.Kind{quality.IO, quality.Global, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "io.WriteString", Kind: kind})
		overridden, err := quality.Evaluate(restored, config)
		if err != nil {
			t.Fatal(err)
		}
		result := resultNamed(t, overridden, "Forward")
		if kind == quality.None {
			if len(result.Effects) != 0 {
				t.Errorf("pure policy override lost: %+v", result)
			}
			continue
		}
		if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("%s policy override lost: %+v", kind, result)
		}
	}
}
