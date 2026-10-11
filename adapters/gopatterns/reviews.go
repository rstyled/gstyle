package gopatterns

import (
	"go/ast"
	"strings"
)

// findPatternReviews preserves explicit reasons separately from legacy bug
// suppressions. Structural candidates require a reason at every source site.
func findPatternReviews(fn *ast.FuncDecl) map[string]string {
	if fn.Doc == nil {
		return nil
	}
	reviews := make(map[string]string)
	for _, comment := range fn.Doc.List {
		kinds, reason := patternReview(comment.Text)
		for _, kind := range kinds {
			reviews[kind] = reason
		}
	}
	return reviews
}

func patternReview(comment string) ([]string, string) {
	text, ok := strings.CutPrefix(comment, "//lint:ignore ")
	if !ok {
		return nil, ""
	}
	return reviewFields(strings.Fields(text))
}

func reviewFields(fields []string) ([]string, string) {
	var kinds []string
	for i, field := range fields {
		kind := strings.TrimSuffix(field, ",")
		if !isPatternKind(kind) {
			return kinds, strings.Join(fields[i:], " ")
		}
		kinds = append(kinds, kind)
	}
	// A list of kinds without a reason does not constitute a review.
	return nil, ""
}
