package gocyclo

import (
	"path/filepath"
	"testing"
)

func TestCurrentSourceMethodCallsAreNotRecursion(t *testing.T) {
	for _, test := range []struct {
		path       string
		name       string
		cognitive  int
		cyclomatic int
	}{
		{"domain/callflow/environment_context.go", "(*exitEvaluation).concreteValue", 2, 3},
		{"domain/patterns/ccgraph.go", "(ccPairMatcher).union", 4, 4},
		{"domain/pdg/tables.go", "(*Builder).Text", 1, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			file, err := analyzeFile(filepath.Join("..", "..", test.path))
			if err != nil {
				t.Fatal(err)
			}
			for _, function := range file.Functions {
				if function.Name != test.name {
					continue
				}
				if function.CognitiveComplexity != test.cognitive || function.Complexity != test.cyclomatic {
					t.Fatalf("%s: cognitive=%d, want %d; cyclomatic=%d, want %d; diagnostics=%+v", test.path, function.CognitiveComplexity, test.cognitive, function.Complexity, test.cyclomatic, function.CognitiveDiagnostics)
				}
				return
			}
			t.Fatalf("%s: function %s missing", test.path, test.name)
		})
	}
}

func TestCognitiveRecursionUsesDeclarationIdentity(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		score  int
	}{
		{"cross-file function", "type T struct{}\nfunc (T) work() { work() }", 0},
		{"cross-file conversion", "type T struct{}\nfunc (*T) Text() { _ = Text(1) }", 0},
		{"nested cross-file call", "type T struct{}\nfunc (T) work() { f := func() { work() }; f() }", 0},
		{"real recursion", "func work(n int) { if n > 0 { work(n-1) } }", 2},
		{"nested real recursion", "func work() { f := func() { work() }; f() }", 1},
		{"shadowed function", "func work() { work := func() {}; work() }", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := analyzeLiteralSource(t, "package p\n"+test.source+"\n")
			function := file.Functions[0]
			if function.CognitiveComplexity != test.score {
				t.Fatalf("cognitive=%d, want %d; diagnostics=%+v", function.CognitiveComplexity, test.score, function.CognitiveDiagnostics)
			}
		})
	}
}
