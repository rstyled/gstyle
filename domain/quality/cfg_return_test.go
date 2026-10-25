package quality

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCFGReturnClassification(t *testing.T) {
	const name = "golang.org/x/tools/go/cfg.Block.Return"
	f := fact("query", 1)
	f.Calls = []Call{{Callee: name, Line: 2, Site: 10}}
	report := evaluate(t, f, DefaultConfig())
	if result := report.Functions[0]; !result.Complete || len(result.Effects) != 0 {
		t.Errorf("concrete read-only query remains unknown: %+v", result)
	}
	for _, other := range []string{name + "Value", "golang.org/x/tools/go/cfg.New", "golang.org/x/tools/go/cfg.CFG.Format", "example.com/cfg.Block.Return"} {
		if ClassifyCallee(other) != UnknownEffect {
			t.Errorf("unreviewed call %s loses uncertainty", other)
		}
	}
	data, err := json.Marshal([]Function{f})
	if err != nil {
		t.Fatal(err)
	}
	var restored []Function
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	replay, err := Evaluate(restored, DefaultConfig())
	if err != nil || !reflect.DeepEqual(report, replay) {
		t.Fatal("saved call changed", err)
	}
	for _, kind := range []Kind{IO, MutationEffect, Global, UnknownEffect} {
		config := DefaultConfig()
		config.Prefixes = append(config.Prefixes, Prefix{Path: name, Kind: kind})
		overridden := evaluate(t, f, config).Functions[0]
		if len(overridden.Effects) != 1 || overridden.Effects[0].Kind != kind {
			t.Errorf("lost %s policy: %+v", kind, overridden)
		}
	}
}

func TestCFGReturnExactMatchKeepsPrefixPolicies(t *testing.T) {
	const name = "golang.org/x/tools/go/cfg.Block.Return"
	config := DefaultConfig()
	config.Prefixes = append(config.Prefixes, Prefix{Path: "golang.org/x/tools/go/cfg.", Kind: IO})
	f := fact("query", 1)
	f.Calls = []Call{{Callee: name + "Value", Line: 2}}
	result := evaluate(t, f, config).Functions[0]
	if len(result.Effects) != 1 || result.Effects[0].Kind != IO {
		t.Fatalf("exact mismatch skips prefix policy: %+v", result)
	}
	config.Prefixes = append(config.Prefixes, Prefix{Path: name, Kind: Global})
	result = evaluate(t, f, config).Functions[0]
	if len(result.Effects) != 1 || result.Effects[0].Kind != Global {
		t.Fatalf("configured prefix stops matching suffixes: %+v", result)
	}
}
