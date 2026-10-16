package gopatterns

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/interfacenil"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestInterfaceNilEvidenceMatchesAllFixtureFindings(t *testing.T) {
	root, err := filepath.Abs("testdata/interface-nil")
	if err != nil {
		t.Fatal(err)
	}
	extracted, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	graphs := make([]pdg.Graph, len(extracted.Funcs))
	for i, f := range extracted.Funcs {
		graphs[i] = f.Pdg
	}
	want := interfacenil.Find(graphs)
	actual := make([][]pdg.Ref, len(graphs))
	for _, proof := range interfacenil.Explain(graphs) {
		actual[proof.Comparison.Graph] = append(actual[proof.Comparison.Graph], proof.Comparison.Node)
		if len(proof.Roots) == 0 || len(proof.Steps) == 0 {
			t.Fatal("empty successful proof")
		}
		for _, step := range proof.Steps {
			if !step.Seed && len(step.Dependencies) == 0 {
				t.Fatal("non-seed has no proof inputs")
			}
			if step.Site.Graph < 0 || step.Site.Graph >= len(graphs) {
				t.Fatal("proof lost graph identity")
			}
		}
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf("Explain and Find disagree: %v != %v", actual, want)
	}
}
