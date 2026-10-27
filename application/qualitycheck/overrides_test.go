package qualitycheck

import (
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/quality"
)

func TestOverridesReuseTopLevelKeysAndLastMatchWins(t *testing.T) {
	path := writeTemp(t, "gstyle.toml", `fn_params = { max = 6 }
[weights]
global = 5
[[prefixes]]
path = "example.com/store.Save"
kind = "io"
[[overrides]]
paths = ["adapters"]
side_effect_density = { max = 800 }
weights = { io = 0 }
[[overrides]]
paths = ["adapters/sql", "cmd/main.go"]
fn_params = { enabled = false }
`)
	config, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	adapters := config.For("adapters/git/diff.go")
	if adapters.SideEffectDensity.Max != 800 || adapters.Weights.IO != 0 || adapters.Weights.Global != 5 || adapters.Weights.Network != 3 || adapters.FnParams.Max != 6 {
		t.Fatalf("adapters override lost inherited settings: %+v", adapters)
	}
	if !slices.ContainsFunc(adapters.Prefixes, func(prefix quality.Prefix) bool { return prefix.Path == "example.com/store.Save" }) || len(adapters.Prefixes) != len(config.Prefixes) {
		t.Fatal("override must keep default and user prefixes")
	}
	sql := config.For("adapters/sql/store.go")
	if sql.FnParams.Enabled || sql.SideEffectDensity.Max != 500 || sql.Weights.IO != 3 {
		t.Fatalf("last matching override must apply alone over the top level: %+v", sql)
	}
	if file := config.For("cmd/main.go"); file.FnParams.Enabled {
		t.Fatal("file path override did not match")
	}
	for _, path := range []string{"adapterstool/x.go", "domain/a.go", "cmd/main.go.bak"} {
		if got := config.For(path); got.SideEffectDensity.Max != 500 || !got.FnParams.Enabled {
			t.Errorf("%s matched an override across a path boundary", path)
		}
	}
}

func TestOverridesRejectRepositoryKeysTyposAndBadPaths(t *testing.T) {
	for _, source := range []string{
		"[[overrides]]\npaths = ['a']\nguard = {}",
		"[[overrides]]\npaths = ['a']\nrisk = { pdg_change = 'high' }",
		"[[overrides]]\npaths = ['a']\n[[overrides.prefixes]]\npath = 'x'\nkind = 'io'",
		"[[overrides]]\npaths = ['a']\ngrouping = { caller_hops = 2 }",
		"[[overrides]]\npaths = ['a']\nside_effect_densty = { max = 1 }",
		"[[overrides]]\npaths = ['a']\nweights = { io = -1 }",
		"[[overrides]]\nfn_params = { max = 9 }",
		"[[overrides]]\npaths = ['../a']",
		"[[overrides]]\npaths = ['a/**']",
		"[[overrides]]\npaths = ['/a']",
	} {
		if _, err := LoadConfig(writeTemp(t, "bad.toml", source)); err == nil {
			t.Errorf("invalid override accepted: %s", source)
		}
	}
}

func TestOverridesChangeEvaluationByFunctionPath(t *testing.T) {
	config, err := LoadConfig(writeTemp(t, "gstyle.toml", "[[overrides]]\npaths = ['adapters']\nfn_params = { max = 6 }\n"))
	if err != nil {
		t.Fatal(err)
	}
	fact := func(path string) quality.Function {
		return quality.Function{Location: quality.Location{Path: path, Line: 1, Column: 1, Name: path}, Params: 5, Statements: 1, CodeLines: 1}
	}
	report, err := quality.Evaluate([]quality.Function{fact("adapters/a.go"), fact("domain/a.go")}, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Diagnostics) != 1 || report.Diagnostics[0].Path != "domain/a.go" || report.Diagnostics[0].RuleID != "fn_params" {
		t.Fatalf("override must apply only under adapters: %+v", report.Diagnostics)
	}
}
