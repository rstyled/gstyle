package gopatterns

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"
)

func TestStructuralReviewsRequireReasons(t *testing.T) {
	for _, tc := range []struct {
		comment string
		kinds   []string
		reason  string
	}{
		{"//lint:ignore chop, thin_slice one cohesive calculation", []string{"chop", "thin_slice"}, "one cohesive calculation"},
		{"//lint:ignore parameterize distinct contracts", []string{"parameterize"}, "distinct contracts"},
		{"//lint:ignore chop", nil, ""},
		{"//lint:ignore unknown unrelated", nil, "unknown unrelated"},
		{"// ordinary comment", nil, ""},
		{"//lint:ignorechop malformed", nil, ""},
	} {
		kinds, reason := patternReview(tc.comment)
		if !reflect.DeepEqual(kinds, tc.kinds) || reason != tc.reason {
			t.Fatalf("%q: %v %q", tc.comment, kinds, reason)
		}
	}
	file, err := parser.ParseFile(token.NewFileSet(), "a.go", "package p\n//lint:ignore chop one operation\nfunc f() {}", parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	if got := findPatternReviews(fn); got["chop"] != "one operation" {
		t.Fatalf("review not extracted: %v", got)
	}
}
