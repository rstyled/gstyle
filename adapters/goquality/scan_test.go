package goquality

import (
	"context"
	"encoding/json"
	"reflect"
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
func Local(){var old,new string;fmt.Sscanf("old new","%s %s",&old,&new)}
func PlainScan(){var value int;fmt.Sscan("1",&value)}
func Scanln(){var value bool;fmt.Sscanln("true",&value)}
func Aliased(){var value Alias;fmt.Sscan("text",&value)}
func Named(){var value Scalar;fmt.Sscan("1",&value)}
func Shared(value *int){fmt.Sscan("1",value)}
func Callback(){var value Custom;fmt.Sscan("1",&value)}
func Open(value any){fmt.Sscan("1",value)}
func Spread(values ...any){fmt.Sscan("1",values...)}
func Reader(input io.Reader){var value int;fmt.Fscan(input,&value)}
func Stdin(){var value int;fmt.Scan(&value)}
var counter int
func next()string{counter++;return "1"}
func ArgumentEffect(){var value int;fmt.Sscan(next(),&value)}
`))
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Local", "PlainScan", "Scanln", "Aliased", "Named", "Shared"} {
		result := resultNamed(t, report, name)
		if len(result.Effects) != 1 || result.Effects[0].Kind != quality.MutationEffect {
			t.Errorf("%s parses a string, want mutation without IO: %+v", name, result)
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
