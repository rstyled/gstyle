package goquality

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestStringScanningKeepsMutationAndCallbacks(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import("fmt";"io")
type Alias = string
type Scalar int
type Custom int
func(*Custom)Scan(fmt.ScanState,rune)error{println("callback");return nil}
// Mirrors adapters/gitchanged.diffNewPath's two private string destinations.
func Local(){var old,new string;fmt.Sscanf("old new","%q %q",&old,&new)}
func PlainScan(){var value int;fmt.Sscan("1",&value)}
func Scanln(){var value bool;fmt.Sscanln("true",&value)}
func Aliased(){var value Alias;fmt.Sscan("text",&value)}
func Named(){var value Scalar;fmt.Sscan("1",&value)}
func Shared(value *int){fmt.Sscan("1",value)}
func Mixed(value *int){var local int;fmt.Sscan("1 2",&local,value)}
var shared int
func Global(){fmt.Sscan("1",&shared)}
func Rebound(value *int){var local int;p:=&local;p=value;fmt.Sscan("1",p)}
func LocalAlias(){var local int;p:=&local;fmt.Sscan("1",p)}
func Forward(){Local()}
func SharedForward(value *int){Shared(value)}
func LocalForward(){var local int;Shared(&local)}
func RawWrite(value *int){var local int;fmt.Sscan("1",&local);*value=2}
func Callback(){var value Custom;fmt.Sscan("1",&value)}
func Open(value any){fmt.Sscan("1",value)}
func Spread(values ...any){fmt.Sscan("1",values...)}
func Reader(input io.Reader){var value int;fmt.Fscan(input,&value)}
func Stdin(){var value int;fmt.Scan(&value)}
var counter int
func next()string{counter++;return "1"}
func ArgumentEffect(){var value int;fmt.Sscan(next(),&value)}
func UnknownArgument(next func()string){var value int;fmt.Sscan(next(),&value)}
func IO(){var value int;fmt.Sscan("1",&value);println("write")}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "PlainScan", "Scanln", "Aliased", "Named", "LocalAlias", "Forward", "LocalForward"} {
		result := resultNamed(t, report, name)
		if !result.Complete || len(result.Effects) != 0 {
			t.Errorf("%s must retain private scalar scan storage: %+v", name, result)
		}
	}
	for name, detail := range map[string]string{
		"Shared": "fmt.Sscan", "Mixed": "fmt.Sscan", "Global": "fmt.Sscan", "Rebound": "fmt.Sscan",
		"SharedForward": "example.com/quality-repro.Shared → fmt.Sscan", "RawWrite": "value",
	} {
		result := resultNamed(t, report, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == quality.MutationEffect && e.Detail == detail }) {
			t.Errorf("%s loses its %s write: %+v", name, detail, result)
		}
		want := 1
		if name == "Global" {
			want = 2 // The global address read is separate evidence.
		}
		if len(result.Effects) != want {
			t.Errorf("%s has unexpected effects: %+v", name, result)
		}
	}
	for _, name := range []string{"Local", "Shared", "Mixed", "Global", "Rebound", "LocalAlias", "RawWrite"} {
		fact := findFunction(t, facts, name)
		t.Logf("%s: calls=%+v; writes=%+v", name, fact.Calls, fact.Mutations)
		if len(fact.Calls) != 1 || !fact.Calls[0].MemoryWriter || len(fact.Calls[0].StorageArgs) == 0 {
			t.Errorf("%s loses typed raw scan evidence: %+v", name, fact)
		}
	}
	for _, name := range []string{"Callback", "Open", "Spread", "Reader", "Stdin"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.IO {
			t.Errorf("%s must keep IO or callback uncertainty: %+v", name, result)
		}
	}
	found := false
	for _, effect := range resultNamed(t, report, "ArgumentEffect").Effects {
		found = found || effect.Kind == quality.Global
	}
	if !found {
		t.Error("scan argument lost its global effect")
	}
	for name, kind := range map[string]quality.Kind{"UnknownArgument": quality.UnknownEffect, "IO": quality.IO, "Global": quality.Global} {
		result := resultNamed(t, report, name)
		found := false
		for _, effect := range result.Effects {
			found = found || effect.Kind == kind
		}
		if !found {
			t.Errorf("%s loses independent %s evidence: %+v", name, kind, result)
		}
	}
	writes := findFunction(t, facts, "RawWrite").Mutations
	if len(writes) != 1 || writes[0].Provenance != quality.External {
		t.Errorf("scan ownership loses raw caller write: %+v", writes)
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
		t.Fatal("saved scan evidence changed", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "fmt.Sscanf", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, overridden, "Local")
	if len(result.Effects) != 1 || result.Effects[0].Kind != quality.IO {
		t.Errorf("scan proof hid policy override: %+v", result)
	}
}
