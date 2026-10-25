package gocyclo

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain"
)

func TestCognitiveLogicalDiagnosticPositions(t *testing.T) {
	for _, test := range []struct {
		name        string
		source      string
		diagnostics []domain.CognitiveDiagnostic
	}{
		{
			name:   "current isASCIILetter condition",
			source: "package p\nfunc isASCIILetter(c byte) bool {\n\treturn 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'\n}\n",
			diagnostics: []domain.CognitiveDiagnostic{
				{Increment: 1, Kind: "&&", Line: 3, Column: 18},
				{Increment: 1, Kind: "||", Line: 3, Column: 30},
				{Increment: 1, Kind: "&&", Line: 3, Column: 42},
			},
		},
		{
			name:        "one homogeneous sequence keeps its location",
			source:      "package p\nfunc f(a, b, c bool) bool { return a && b && c }\n",
			diagnostics: []domain.CognitiveDiagnostic{{Increment: 1, Kind: "&&", Line: 2, Column: 43}},
		},
		{
			name:   "parenthesized sequences",
			source: "package p\nfunc f(a, b, c, d bool) bool { return (a && b) || (c && d) }\n",
			diagnostics: []domain.CognitiveDiagnostic{
				{Increment: 1, Kind: "||", Line: 2, Column: 48},
				{Increment: 1, Kind: "&&", Line: 2, Column: 42},
				{Increment: 1, Kind: "&&", Line: 2, Column: 54},
			},
		},
		{
			name:   "negation keeps separate sequences",
			source: "package p\nfunc f(a, b, c, d bool) bool { return a && !(b || c) && d }\n",
			diagnostics: []domain.CognitiveDiagnostic{
				{Increment: 1, Kind: "&&", Line: 2, Column: 54},
				{Increment: 1, Kind: "||", Line: 2, Column: 48},
			},
		},
		{
			name:   "function variable initializer",
			source: "package p\nvar f = func(a, b, c bool) bool { return a || b && c }\n",
			diagnostics: []domain.CognitiveDiagnostic{
				{Increment: 1, Kind: "||", Line: 2, Column: 44},
				{Increment: 1, Kind: "&&", Line: 2, Column: 49},
			},
		},
		{
			name:   "physical multiline sequence and branch",
			source: "package p\n//line imaginary.go:100\nfunc f(a, b, c bool) {\n\tif a ||\n\t\tb && c {\n\t}\n}\n",
			diagnostics: []domain.CognitiveDiagnostic{
				{Increment: 1, Kind: "if", Line: 4, Column: 2},
				{Increment: 1, Kind: "||", Line: 4, Column: 7},
				{Increment: 1, Kind: "&&", Line: 5, Column: 5},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			function := analyzeLiteralSource(t, test.source).Functions[0]
			if !reflect.DeepEqual(function.CognitiveDiagnostics, test.diagnostics) {
				t.Fatalf("diagnostics = %+v, want %+v", function.CognitiveDiagnostics, test.diagnostics)
			}
			if function.CognitiveComplexity != len(test.diagnostics) {
				t.Fatalf("complexity = %d, want %d", function.CognitiveComplexity, len(test.diagnostics))
			}
		})
	}
}
