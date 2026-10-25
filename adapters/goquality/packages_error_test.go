package goquality

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestPackagesErrorCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"adapters/goquality/analyzer.go"})
	if err != nil {
		t.Fatal(err)
	}
	function := findFunction(t, facts, "packageErrors")
	var calls int
	for _, call := range function.Calls {
		if call.Callee == "golang.org/x/tools/go/packages.Error.Error" {
			calls++
			if call.Dynamic || call.Local {
				t.Fatalf("expected concrete external method: %+v", call)
			}
		}
	}
	if calls != 1 {
		t.Fatalf("want one raw Error call, got %d", calls)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, report, "packageErrors")
	if !result.Complete || len(result.Effects) != 0 {
		t.Fatalf("string-only package error query retains false effects: %+v", result)
	}
}

func TestPackagesErrorBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "golang.org/x/tools/go/packages"
func Query(e packages.Error)string{return e.Error()}
func Expression(e packages.Error)string{return packages.Error.Error(e)}
func Forward(e packages.Error)string{return Query(e)}
func Shared(e *packages.Error){e.Error();e.Msg="changed"}
func Alias(e *packages.Error){alias:=e;alias.Error();alias.Pos="changed"}
func Child(e []packages.Error){e[0].Error();e[0].Msg="changed"}
var saved packages.Error
func Publish(e packages.Error){e.Error();saved=e}
func Callback(e packages.Error,run func()){e.Error();run()}
func IO(e packages.Error){e.Error();println("retained")}
func Indirect(e packages.Error){query:=e.Error;query()}
type errorer interface{Error()string}
func Dynamic(e errorer){e.Error()}
type custom struct{count int}
func(e *custom)Error()string{e.count++;return "custom"}
func Other(e *custom){e.Error()}
func Load(){packages.Load(nil)}
`))
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/quality-repro\n\ngo 1.26.0\n\nrequire golang.org/x/tools v0.50.0\n\nrequire (\n golang.org/x/mod v0.41.0 // indirect\n golang.org/x/sync v0.23.0 // indirect\n)\n"), 0600); err != nil {
		t.Fatal(err)
	}
	sums, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.sum"), sums, 0600); err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Run("queries", func(t *testing.T) {
		for _, name := range []string{"Query", "Expression", "Forward"} {
			result := resultNamed(t, report, name)
			if !result.Complete || len(result.Effects) != 0 {
				t.Errorf("%s retains false query effects: %+v", name, result)
			}
		}
	})
	t.Run("controls", func(t *testing.T) {
		for name, kind := range map[string]quality.Kind{"Shared": quality.MutationEffect, "Alias": quality.MutationEffect, "Child": quality.MutationEffect, "Publish": quality.Global, "Callback": quality.UnknownEffect, "IO": quality.IO, "Indirect": quality.UnknownEffect, "Dynamic": quality.UnknownEffect, "Other": quality.MutationEffect, "Load": quality.UnknownEffect} {
			result := resultNamed(t, report, name)
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("%s loses %s: %+v", name, kind, result)
			}
		}
		for _, name := range []string{"Shared", "Alias", "Child"} {
			function := findFunction(t, facts, name)
			if len(function.Mutations) != 1 || function.Mutations[0].Provenance != quality.External {
				t.Errorf("%s loses raw caller write: %+v", name, function.Mutations)
			}
		}
	})
	data, err := json.Marshal(facts)
	if err != nil {
		t.Fatal(err)
	}
	var restored []quality.Function
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := quality.Evaluate(restored, quality.DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("saved package error facts change evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "golang.org/x/tools/go/packages.Error.Error", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Query", "Expression", "Forward", "Shared"} {
		result := resultNamed(t, overridden, name)
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool {
			return e.Kind == quality.IO && strings.Contains(e.Detail, "golang.org/x/tools/go/packages.Error.Error")
		}) {
			t.Errorf("%s loses saved-fact query policy: %+v", name, result)
		}
	}
}
