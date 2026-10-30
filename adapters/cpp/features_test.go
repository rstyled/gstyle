package cpp

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/quality"
)

func featureExtraction(t *testing.T) *Extraction {
	t.Helper()
	extraction, err := Load(context.Background(), "testdata/features.graphson.json")
	if err != nil {
		t.Fatal(err)
	}
	return extraction
}

func TestCPPSharedQuality(t *testing.T) {
	extraction := featureExtraction(t)
	config := DefaultQualityConfig()
	config.FnParams.Max = 2
	report, err := quality.Evaluate(extraction.QualityFacts(), config)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Functions != len(extraction.Functions) {
		t.Fatal("quality omits imported functions")
	}
	results := map[string]quality.FunctionResult{}
	for _, result := range report.Functions {
		name, _, _ := strings.Cut(strings.TrimPrefix(result.Name, "cpp::"), ":")
		results[name] = result
	}
	if !results["pure_caller"].Complete || len(results["pure_caller"].Effects) != 0 {
		t.Fatalf("body-backed pure helper is opaque: %+v", results["pure_caller"])
	}
	if results["opaque_caller"].Complete || results["opaque_caller"].UnclassifiedCalls != 1 {
		t.Fatalf("opaque external call is hidden: %+v", results["opaque_caller"])
	}
	if len(results["change_caller"].Effects) != 2 || results["change"].Mutations != 2 {
		t.Fatalf("shared helper mutations are lost: %+v, %+v", results["change_caller"], results["change"])
	}
	if results["read_volatile"].Complete || !slices.ContainsFunc(results["read_global"].Effects, func(effect quality.Effect) bool { return effect.Kind == quality.GlobalRead }) {
		t.Fatalf("C++ volatile or global field read is lost: %+v, %+v", results["read_volatile"], results["read_global"])
	}
	rules := map[string]bool{}
	for _, diagnostic := range report.Diagnostics {
		rules[diagnostic.RuleID] = true
	}
	for _, rule := range []string{"fn_params", "aggregate", "repository", "mutable_identity"} {
		if !rules[rule] {
			t.Errorf("C++ does not reach shared %s evaluation", rule)
		}
	}
	for _, fact := range extraction.QualityFacts() {
		if fact.CodeLines < 1 {
			t.Errorf("missing function length for %s", fact.Name)
		}
		if strings.HasPrefix(fact.Name, "cpp::change:") {
			for _, mutation := range fact.Mutations {
				if mutation.Provenance != quality.External || mutation.Parameter == nil || mutation.FieldPath == "" {
					t.Errorf("missing bound field mutation: %+v", mutation)
				}
			}
		}
	}
}

func TestCPPGeneralSourceAndMiningInputs(t *testing.T) {
	extraction := featureExtraction(t)
	report := extraction.Patterns()
	kinds := map[patterns.CandidateKind]bool{}
	for _, candidate := range report.Candidates {
		if candidate.FixSpec != nil || candidate.Kind.Group() == patterns.GoPatterns {
			t.Fatalf("C++ enables a foreign source fix: %+v", candidate)
		}
		kinds[candidate.Kind] = true
	}
	for _, kind := range []patterns.CandidateKind{patterns.PrimitiveObsession, patterns.ValueObject, patterns.Specification, patterns.Factory, patterns.EntityIdentity, patterns.MissingIdentity, patterns.MutableIdentity, patterns.DomainService, patterns.AnemicModel, patterns.Repository, patterns.EnumDispatch, patterns.CCGraphClone} {
		if !kinds[kind] {
			t.Errorf("missing C++ %s input", kind)
		}
	}
	for _, function := range extraction.Functions {
		if function.mining.SigKey == "" || len(function.mining.AstTypes) == 0 {
			t.Fatalf("missing shared mining facts for %s", function.Score.Name)
		}
	}
	var literalValues []string
	for _, function := range extraction.Functions {
		graph := patterns.MiningView(function.Graph)
		for _, node := range graph.Nodes {
			if node.Kind == patterns.Lit {
				literalValues = append(literalValues, node.Detail)
			}
		}
	}
	if !slices.Contains(literalValues, "2") || !slices.Contains(literalValues, "8") {
		t.Fatal("mining erases literal differences")
	}
}

func TestCPPCodeLines(t *testing.T) {
	for _, test := range []struct {
		source string
		want   int
	}{
		{"// comment\n\n/* block\ncomment */\n", 0},
		{"int f() {\n // comment\n return 1; /* tail */\n}\n", 3},
		{"const char *p = R\"tag(//\n/* text */\n)tag\";", 3},
		{"const char *p = \"/* text */\";\nchar c = 'x';", 2},
		{"// continued \\\nnot code\nint x;", 1},
		{"int x; /*\n comment\n */ int y;", 2},
		{"int x = 1'000;\n// comment\nreturn x;", 2},
	} {
		if got := codeLines(test.source); got != test.want {
			t.Errorf("codeLines(%q) = %d, want %d", test.source, got, test.want)
		}
	}
}

func TestCPPSharedMiningCacheIsStable(t *testing.T) {
	extraction := featureExtraction(t)
	params := patterns.DefaultParams()
	params.ThresholdMilli = 700
	cache := patterns.NewWlCache()
	want := extraction.PatternsWithMining(params, nil)
	for range 2 {
		got := extraction.PatternsWithMining(params, cache)
		if !reflect.DeepEqual(got, want) {
			t.Fatal("C++ mining cache changes the report")
		}
	}
}
