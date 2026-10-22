package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestFlagSetOutputKeepsMutationWithoutGlobalEffect(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("flag";"io";"log")
type Alias = flag.FlagSet
type Writer struct{}
func(Writer)Write(data []byte)(int,error){println("callback");return len(data),nil}
func Set(flags *flag.FlagSet, out io.Writer){flags.SetOutput(out)}
func Aliased(flags *Alias, out io.Writer){flags.SetOutput(out)}
func Expression(flags *flag.FlagSet, out io.Writer){(*flag.FlagSet).SetOutput(flags,out)}
func Forward(flags *flag.FlagSet, out io.Writer){Set(flags,out)}
func Local(out io.Writer){flags:=flag.NewFlagSet("local",flag.ContinueOnError);flags.SetOutput(out)}
func Callback(){var flags flag.FlagSet;flags.SetOutput(Writer{})}
func Nil(){var flags flag.FlagSet;flags.SetOutput(nil)}
func Parse(flags *flag.FlagSet, args []string)error{return flags.Parse(args)}
func Print(flags *flag.FlagSet){flags.PrintDefaults()}
func Logging(out io.Writer){log.SetOutput(out)}
func Global(out io.Writer){flag.CommandLine.SetOutput(out)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Set", "Aliased", "Expression", "Forward", "Local", "Callback", "Nil"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect || !result.Complete {
			t.Errorf("%s only stores the writer, want one mutation: %+v", name, result)
		}
	}
	for name, kind := range map[string]quality.Kind{"Parse": quality.IO, "Print": quality.IO, "Logging": quality.Global} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != kind {
			t.Errorf("%s must retain %s: %+v", name, kind, result)
		}
	}
	global := resultNamed(t, report, "Global")
	if len(global.Effects) != 2 || global.Effects[0].Kind != quality.Global || global.Effects[1].Kind != quality.MutationEffect {
		t.Errorf("CommandLine must retain its global read and mutation: %+v", global)
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
		t.Fatal("saved flag output effects changed", err)
	}
	for _, kind := range []quality.Kind{quality.Global, quality.None} {
		config := quality.DefaultConfig()
		config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "flag.FlagSet.SetOutput", Kind: kind})
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
			t.Errorf("global policy override lost: %+v", result)
		}
	}
}
