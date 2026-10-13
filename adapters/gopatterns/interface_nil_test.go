package gopatterns

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/interfacenil"
	"github.com/rstyled/gstyle/domain/pdg"
)

func TestSyntheticInterfaceNilPDG(t *testing.T) {
	root, err := filepath.Abs("testdata/interface-nil")
	if err != nil {
		t.Fatal(err)
	}
	extraction, err := Extract(context.Background(), root, []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(root, "sample.go"))
	if err != nil {
		t.Fatal(err)
	}
	var expected, actual []int
	for line, text := range strings.Split(string(source), "\n") {
		if strings.Contains(text, "// want-nonnil:") {
			expected = append(expected, line+1)
		}
	}
	for _, function := range extraction.Funcs {
		for _, hit := range function.TypedNilHits {
			actual = append(actual, hit.Line)
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("PDG warnings %v; expected %v", actual, expected)
	}
	testInterfaceProofDependencies(t, extraction.Funcs)
}

// Removing supported dependencies must remove proof, not trigger an AST fallback.
func testInterfaceProofDependencies(t *testing.T, functions []FuncPdg) {
	t.Helper()
	graphs := make([]pdg.Graph, len(functions))
	var target int
	for i, function := range functions {
		graphs[i] = function.Pdg
		if strings.HasSuffix(function.Name, ".assignedCall") {
			target = i
		}
	}
	edges := make([]pdg.Edge, 0, len(graphs[target].Edges))
	for _, edge := range graphs[target].Edges {
		if graphs[target].Tables.Text(edge.Subkind) != "reaching-definition" {
			edges = append(edges, edge)
		}
	}
	graphs[target].Edges = edges
	if hits := interfacenil.Find(graphs)[target]; len(hits) != 0 {
		t.Fatalf("proof survives removed PDG dependencies: %v", hits)
	}
	for i := range graphs {
		graphs[i].Capabilities = nil
	}
	for _, hits := range interfacenil.Find(graphs) {
		if len(hits) != 0 {
			t.Fatal("proof survives missing capability evidence")
		}
	}
}
