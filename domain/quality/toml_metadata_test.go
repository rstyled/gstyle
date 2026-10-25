package quality

import "testing"

func TestTOMLUndecodedClassification(t *testing.T) {
	const name = "github.com/BurntSushi/toml.MetaData.Undecoded"
	f := fact("query", 1)
	f.Calls = []Call{{Callee: name, Line: 2, Site: 10}}
	result := evaluate(t, f, DefaultConfig()).Functions[0]
	if !result.Complete || len(result.Effects) != 0 {
		t.Errorf("concrete read-only metadata query remains unknown: %+v", result)
	}
	for _, other := range []string{name + "Value", "github.com/BurntSushi/toml.Decode", "github.com/BurntSushi/toml.MetaData.PrimitiveDecode", "github.com/BurntSushi/toml.MetaData.Keys", "github.com/BurntSushi/toml.Key.String", "example.com/toml.MetaData.Undecoded"} {
		if ClassifyCallee(other) != UnknownEffect {
			t.Errorf("unreviewed call %s loses uncertainty", other)
		}
	}
	for _, kind := range []Kind{IO, MutationEffect, Global, UnknownEffect, Panic} {
		config := DefaultConfig()
		config.Prefixes = append(config.Prefixes, Prefix{Path: name, Kind: kind})
		overridden := evaluate(t, f, config).Functions[0]
		if len(overridden.Effects) != 1 || overridden.Effects[0].Kind != kind {
			t.Errorf("lost %s policy: %+v", kind, overridden)
		}
	}
}

func TestTOMLUndecodedExactMatchKeepsPrefixPolicies(t *testing.T) {
	const name = "github.com/BurntSushi/toml.MetaData.Undecoded"
	config := DefaultConfig()
	config.Prefixes = append(config.Prefixes, Prefix{Path: "github.com/BurntSushi/toml.", Kind: IO})
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
