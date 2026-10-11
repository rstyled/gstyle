package pdgjson

import (
	"testing"

	"github.com/shanejonas/cyclo/domain/pdg"
)

func TestVariadicBooleanPreservesKnownAndUnknown(t *testing.T) {
	for _, value := range []bool{false, true} {
		g := exportFixture(t)
		g.Function.Inputs[0].Variadic = pdg.BooleanFact{Status: pdg.Known, Value: value}
		doc := exportedObject(t, g)
		input := doc["function"].(map[string]any)["inputs"].([]any)[0].(map[string]any)
		fact := input["variadic"].(map[string]any)
		if fact["status"] != "known" || fact["value"] != value {
			t.Fatalf("boolean became string or unknown: %v", fact)
		}
	}
	g := exportFixture(t)
	g.Function.Inputs[0].Variadic = pdg.BooleanFact{Status: pdg.Unknown, Reason: g.Producer.LanguageVersion.Value}
	doc := exportedObject(t, g)
	fact := doc["function"].(map[string]any)["inputs"].([]any)[0].(map[string]any)["variadic"].(map[string]any)
	if fact["status"] != "unknown" || fact["reason"] == nil {
		t.Fatal(fact)
	}
	if _, ok := fact["value"]; ok {
		t.Fatal("unknown became false")
	}
}
