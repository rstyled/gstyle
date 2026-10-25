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

	"github.com/BurntSushi/toml"
	"github.com/rstyled/gstyle/domain/quality"
)

func TestTOMLUndecodedCurrentSource(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := (Analyzer{Root: root}).Extract(context.Background(), []string{"application/qualitycheck/config.go"})
	if err != nil {
		t.Fatal(err)
	}
	function := findFunction(t, facts, "decodeConfig")
	var calls int
	for _, call := range function.Calls {
		if call.Callee == "github.com/BurntSushi/toml.MetaData.Undecoded" {
			calls++
			if call.Dynamic || call.Local {
				t.Fatalf("expected concrete external metadata query: %+v", call)
			}
		}
	}
	if calls != 1 {
		t.Fatalf("want one raw Undecoded call, got %d", calls)
	}
	report, err := quality.Evaluate(facts, quality.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	result := resultNamed(t, report, "decodeConfig")
	for _, effect := range result.Effects {
		if strings.Contains(effect.Detail, "github.com/BurntSushi/toml.MetaData.Undecoded") {
			t.Errorf("read-only metadata query retains false effect: %+v", effect)
		}
	}
	for _, name := range []string{"github.com/BurntSushi/toml.Decode", "github.com/BurntSushi/toml.Key.String"} {
		if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool {
			return e.Kind == quality.UnknownEffect && strings.Contains(e.Detail, name)
		}) {
			t.Errorf("lost unreviewed call %s: %+v", name, result)
		}
	}
}

func TestTOMLUndecodedBoundaries(t *testing.T) {
	root := qualitySourceModule(t, []byte(`package repro
import "github.com/BurntSushi/toml"
func Query(md *toml.MetaData)[]toml.Key{return md.Undecoded()}
func Expression(md *toml.MetaData)[]toml.Key{return (*toml.MetaData).Undecoded(md)}
func Forward(md *toml.MetaData)[]toml.Key{return Query(md)}
func Shared(md *toml.MetaData){md.Undecoded();*md=toml.MetaData{}}
func Alias(md *toml.MetaData){alias:=md;alias.Undecoded();*alias=toml.MetaData{}}
func Child(md []toml.MetaData){md[0].Undecoded();md[0]=toml.MetaData{}}
func Result(md *toml.MetaData){keys:=md.Undecoded();keys[0][0]="changed"}
func ResultAlias(md *toml.MetaData){keys:=md.Undecoded();alias:=keys[0];alias[0]="changed"}
var saved []toml.Key
func Publish(md *toml.MetaData){saved=md.Undecoded()}
func Callback(md *toml.MetaData,run func()){md.Undecoded();run()}
func IO(md *toml.MetaData){md.Undecoded();println("real")}
func Panic(md *toml.MetaData){md.Undecoded();panic("real")}
func Indirect(md *toml.MetaData){query:=md.Undecoded;query()}
func Dynamic(md interface{Undecoded()[]toml.Key}){md.Undecoded()}
func Decode(destination any){toml.Decode("key=1",destination)}
func Primitive(md *toml.MetaData,p toml.Primitive,destination any){md.PrimitiveDecode(p,destination)}
`))
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/quality-repro\n\ngo 1.26.0\n\nrequire github.com/BurntSushi/toml v1.6.0\n"), 0600); err != nil {
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
				t.Errorf("%s retains false metadata query effects: %+v", name, result)
			}
		}
	})
	t.Run("controls", func(t *testing.T) {
		for name, kind := range map[string]quality.Kind{"Shared": quality.MutationEffect, "Alias": quality.MutationEffect, "Child": quality.MutationEffect, "Result": quality.UnknownEffect, "ResultAlias": quality.UnknownEffect, "Publish": quality.Global, "Callback": quality.UnknownEffect, "IO": quality.IO, "Panic": quality.Panic, "Indirect": quality.UnknownEffect, "Dynamic": quality.UnknownEffect, "Decode": quality.UnknownEffect, "Primitive": quality.UnknownEffect} {
			result := resultNamed(t, report, name)
			if !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool { return e.Kind == kind }) {
				t.Errorf("%s loses %s: %+v", name, kind, result)
			}
		}
		for _, name := range []string{"Shared", "Alias", "Child", "Result", "ResultAlias"} {
			function := findFunction(t, facts, name)
			if len(function.Mutations) != 1 || function.Mutations[0].Provenance == quality.Local {
				t.Errorf("%s loses raw shared or unproved write: %+v", name, function.Mutations)
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
		t.Fatal("saved metadata call facts change evaluation", err)
	}
	config := quality.DefaultConfig()
	config.Prefixes = append(config.Prefixes, quality.Prefix{Path: "github.com/BurntSushi/toml.MetaData.Undecoded", Kind: quality.IO})
	overridden, err := quality.Evaluate(restored, config)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Query", "Expression", "Forward", "Shared", "Result"} {
		if result := resultNamed(t, overridden, name); !slices.ContainsFunc(result.Effects, func(e quality.Effect) bool {
			return e.Kind == quality.IO && strings.Contains(e.Detail, "github.com/BurntSushi/toml.MetaData.Undecoded")
		}) {
			t.Errorf("%s loses saved-fact metadata policy: %+v", name, result)
		}
	}
}

func TestTOMLUndecodedRetainsBorrowedKeys(t *testing.T) {
	var decoded struct{ Known int }
	metadata, err := toml.Decode("known=1\nunknown=2\n", &decoded)
	if err != nil {
		t.Fatal(err)
	}
	keys := metadata.Undecoded()
	if decoded.Known != 1 || !reflect.DeepEqual(keys, []toml.Key{{"unknown"}}) || !reflect.DeepEqual(keys, metadata.Undecoded()) {
		t.Fatalf("unexpected metadata query: %+v, %+v", decoded, keys)
	}
	keys[0][0] = "changed"
	if metadata.Keys()[1][0] != "changed" {
		t.Fatal("query result no longer retains borrowed key storage")
	}
}
