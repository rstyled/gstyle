package gopatterns

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"
)

func unsafeExtractionBody(fn *ast.FuncDecl) bool {
	return hasFuncLit(fn.Body) || functionDirective(fn.Doc)
}

func functionDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, comment := range doc.List {
		if strings.HasPrefix(comment.Text, "//go:") {
			return true
		}
	}
	return false
}

func forwardableParameters(fn *ast.FuncDecl) bool {
	if fn.Type.Params == nil {
		return true
	}
	for _, field := range fn.Type.Params.List {
		if len(field.Names) == 0 {
			return false
		}
		for _, name := range field.Names {
			if name.Name == "_" {
				return false
			}
		}
	}
	return true
}

func (source *parameterizationSource) rewriteRisk(pair *parameterizationPair, holes []hole) string {
	if len(source.packageNames) == 0 {
		return "package binding names are unavailable for helper extraction"
	}
	if reason := parameterizationImportRisk(pair); reason != "" {
		return reason
	}
	return parameterizationGraphRisk(pair, holes, source.fset)
}
func parameterizationGraphRisk(pair *parameterizationPair, holes []hole, fset *token.FileSet) string {
	if !matchingParameterizationBindings(pair.left, pair.right) {
		return "PDG binding identities differ under the source substitutions"
	}
	if !matchingParameterizationDependencies(pair, holes, fset) {
		return "PDG operation flow, value, control or memory dependencies differ under the source substitutions"
	}
	return ""
}

func matchingCallSpreads(a, b *ast.FuncDecl) bool {
	return slices.Equal(callSpreads(a.Body), callSpreads(b.Body))
}

func callSpreads(node ast.Node) []bool {
	var spreads []bool
	ast.Inspect(node, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			spreads = append(spreads, call.Ellipsis.IsValid())
		}
		return true
	})
	return spreads
}

func parameterArgumentNames(fn *ast.FuncDecl) []string {
	names := paramNames(fn)
	if len(names) == 0 {
		return names
	}
	fields := fn.Type.Params.List
	if _, variadic := fields[len(fields)-1].Type.(*ast.Ellipsis); variadic {
		names[len(names)-1] += "..."
	}
	return names
}

func identifierNames(node ast.Node) map[string]bool {
	names := make(map[string]bool)
	ast.Inspect(node, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok {
			names[id.Name] = true
		}
		return true
	})
	return names
}

func freshIdentifier(base string, used map[string]bool) string {
	name := base
	for i := 1; used[name]; i++ {
		name = fmt.Sprintf("%s%d", base, i)
	}
	used[name] = true
	return name
}

func helperHoleNames(fn *ast.FuncDecl, count int) []string {
	used := identifierNames(fn)
	names := make([]string, count)
	for i := range names {
		base := "fn"
		if count != 1 {
			base = fmt.Sprintf("fn%d", i+1)
		}
		names[i] = freshIdentifier(base, used)
	}
	return names
}

func freshHelperName(source *parameterizationSource, pair *parameterizationPair) string {
	used := identifierNames(source.file)
	for _, item := range pair.second.snapshot.Imports {
		used[item.Name] = true
	}
	for name := range identifierNames(pair.second.file) {
		used[name] = true
	}
	for _, name := range source.packageNames {
		used[name] = true
	}
	base := helperName(pair.a.Name.Name, pair.b.Name.Name)
	if !validHelperIdentifier(base) {
		base = "extracted"
	}
	return freshIdentifier(base, used)
}

func validHelperIdentifier(name string) bool {
	return token.IsIdentifier(name) && name != "_" && name != "init" && types.Universe.Lookup(name) == nil
}

func holesAreCallTargets(a, b *ast.FuncDecl, holes []hole) bool {
	left, right := directCallTargets(a.Body), directCallTargets(b.Body)
	for _, hole := range holes {
		if !left[hole.aExpr] || !right[hole.bExpr] {
			return false
		}
	}
	return true
}

func directCallTargets(body *ast.BlockStmt) map[ast.Expr]bool {
	targets := map[ast.Expr]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			targets[call.Fun] = true
		}
		return true
	})
	return targets
}
