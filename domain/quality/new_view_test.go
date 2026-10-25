package quality

import "testing"

func TestNewViewClassification(t *testing.T) {
	const name = "charm.land/bubbletea/v2.NewView"
	f := fact("build", 1)
	f.Calls = []Call{{Callee: name, Line: 2, Site: 10}}
	result := evaluate(t, f, DefaultConfig()).Functions[0]
	if !result.Complete || len(result.Effects) != 0 {
		t.Errorf("local view constructor remains unknown: %+v", result)
	}
	for _, other := range []string{name + "Value", "charm.land/bubbletea/v2.NewProgram", "charm.land/bubbletea/v2.View.SetContent", "example.com/bubbletea.NewView"} {
		if ClassifyCallee(other) != UnknownEffect {
			t.Errorf("unreviewed call %s loses uncertainty", other)
		}
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

func TestNewViewExactMatchKeepsPrefixPolicies(t *testing.T) {
	const name = "charm.land/bubbletea/v2.NewView"
	config := DefaultConfig()
	config.Prefixes = append(config.Prefixes, Prefix{Path: "charm.land/bubbletea/v2.", Kind: IO})
	f := fact("build", 1)
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
