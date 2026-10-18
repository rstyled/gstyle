package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/patterns"
)

// gostaticanalysis extractors: nilerr, forcetypeassert, typednil.

var errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// nilErrCheck holds the typed subject and outcome of a canonical nil predicate.
type nilErrCheck struct {
	path     bindingflow.Path
	isNotNil bool
}

// errorResultPositions returns every error result, including named results.
func errorResultPositions(fn *ast.FuncDecl, info *types.Info) []int {
	if fn.Type == nil || fn.Type.Results == nil {
		return nil
	}
	if info != nil {
		if object, ok := info.Defs[fn.Name].(*types.Func); ok {
			return signatureErrorPositions(object.Type().(*types.Signature))
		}
	}
	return astErrorPositions(fn.Type.Results)
}

func signatureErrorPositions(sig *types.Signature) []int {
	var positions []int
	for i := 0; i < sig.Results().Len(); i++ {
		if isNilErrErrorType(sig.Results().At(i).Type()) {
			positions = append(positions, i)
		}
	}
	return positions
}

func astErrorPositions(results *ast.FieldList) []int {
	var positions []int
	pos := 0
	for _, field := range results.List {
		count := max(len(field.Names), 1)
		if isErrorASTType(field.Type) {
			for i := 0; i < count; i++ {
				positions = append(positions, pos+i)
			}
		}
		pos += count
	}
	return positions
}

// isNilErrErrorType reports whether t is the error interface type.
func isNilErrErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	if _, generic := types.Unalias(t).(*types.TypeParam); generic {
		return false
	}
	iface, ok := t.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	// error is the interface with exactly the Error() string method.
	return types.Identical(iface, errorType)
}

// isErrorASTType reports whether the AST type expression is `error`.
func isErrorASTType(e ast.Expr) bool {
	ident, ok := e.(*ast.Ident)
	return ok && ident.Name == "error"
}

// isErrorType reports whether e has an error type.
func isErrorExpr(info *types.Info, e ast.Expr) bool {
	if info == nil {
		return false
	}
	t := info.TypeOf(e)
	if t == nil {
		return false
	}
	return types.Implements(t, errorType)
}

// condText returns the source text of a condition.
func condText(fset *token.FileSet, cond ast.Expr) string {
	var sb strings.Builder
	// Simple printer: just use the position info.
	// For the spike, we'll use a basic format.
	sb.WriteString("(condition)")
	return sb.String()
}

// findForceTypeAssertHits finds unchecked `x.(T)` type assertions.
func findForceTypeAssertHits(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info) []patterns.ForceTypeAssertHit {
	var out []patterns.ForceTypeAssertHit
	if fn.Body == nil {
		return out
	}
	commaOk := collectCommaOk(fn.Body)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if hit := checkTypeAssert(n, fset, info, commaOk); hit != nil {
			out = append(out, *hit)
		}
		return true
	})
	return out
}

// collectCommaOk finds type assertions in comma-ok form: `v, ok := x.(T)`.
func collectCommaOk(body ast.Node) map[token.Pos]bool {
	commaOk := map[token.Pos]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		collectAssignCommaOk(n, commaOk)
		collectValueSpecCommaOk(n, commaOk)
		return true
	})
	return commaOk
}

// collectAssignCommaOk marks assertions in `v, ok := x.(T)`.
func collectAssignCommaOk(n ast.Node, commaOk map[token.Pos]bool) {
	assign, ok := n.(*ast.AssignStmt)
	if !ok || len(assign.Lhs) != 2 || len(assign.Rhs) != 1 {
		return
	}
	for _, rhs := range assign.Rhs {
		markAssertExprs(rhs, commaOk)
	}
}

// collectValueSpecCommaOk marks assertions in `var v, ok = x.(T)`.
func collectValueSpecCommaOk(n ast.Node, commaOk map[token.Pos]bool) {
	vs, ok := n.(*ast.ValueSpec)
	if !ok || len(vs.Names) != 2 || len(vs.Values) != 1 {
		return
	}
	for _, v := range vs.Values {
		markAssertExprs(v, commaOk)
	}
}

// markAssertExprs marks all type assertions under e.
func markAssertExprs(e ast.Expr, commaOk map[token.Pos]bool) {
	ast.Inspect(e, func(m ast.Node) bool {
		if tae, ok := m.(*ast.TypeAssertExpr); ok {
			commaOk[tae.Pos()] = true
		}
		return true
	})
}

// checkTypeAssert checks one node for an unchecked assertion.
func checkTypeAssert(n ast.Node, fset *token.FileSet, info *types.Info, commaOk map[token.Pos]bool) *patterns.ForceTypeAssertHit {
	tae, ok := n.(*ast.TypeAssertExpr)
	if !ok || commaOk[tae.Pos()] || tae.Type == nil {
		return nil
	}
	if isEmptyInterface(info, tae.Type) {
		return nil
	}
	return &patterns.ForceTypeAssertHit{
		Line:     fset.Position(tae.Pos()).Line,
		ExprText: "(type assertion)",
	}
}

// isEmptyInterface reports whether e is `any` or empty interface.
func isEmptyInterface(info *types.Info, e ast.Expr) bool {
	if info == nil {
		return false
	}
	t := info.TypeOf(e)
	if t == nil {
		return false
	}
	iface, ok := t.Underlying().(*types.Interface)
	return ok && iface.Empty()
}

// findSuppressedKinds extracts suppressed pattern kinds from the function's
// doc comment. Supports both the standard lint-ignore comment form and the
// legacy allowlist form.
func findSuppressedKinds(fn *ast.FuncDecl) []string {
	if fn.Doc == nil {
		return nil
	}
	var kinds []string
	for _, c := range fn.Doc.List {
		kinds = append(kinds, parseSuppressComment(c.Text)...)
	}
	return kinds
}

// parseSuppressComment extracts kind IDs from one comment line.
func parseSuppressComment(text string) []string {
	if strings.HasPrefix(text, "//lint:ignore") {
		return parseLintIgnoreKinds(text)
	}
	if strings.HasPrefix(text, "// gstyle-allow(") {
		return parseGstyleAllowKinds(text)
	}
	return nil
}

// parseLintIgnoreKinds extracts kinds from `//lint:ignore k1, k2 reason`.
func parseLintIgnoreKinds(text string) []string {
	parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(text, "//lint:ignore")))
	if len(parts) == 0 {
		return nil
	}
	var kinds []string
	for i, p := range parts {
		for _, kind := range strings.Split(strings.TrimSuffix(p, ","), ",") {
			if isPatternKind(kind) {
				kinds = append(kinds, kind)
			}
		}
		if !strings.HasSuffix(parts[i], ",") {
			break
		}
	}
	return kinds
}

// parseGstyleAllowKinds extracts kinds from the legacy allowlist comment form.
func parseGstyleAllowKinds(text string) []string {
	inner := strings.TrimPrefix(text, "// gstyle-allow(")
	idx := strings.Index(inner, "):")
	if idx < 0 {
		return nil
	}
	var kinds []string
	for _, r := range strings.Split(inner[:idx], ",") {
		r = strings.TrimSpace(r)
		if isPatternKind(r) {
			kinds = append(kinds, r)
		}
	}
	return kinds
}

// isPatternKind reports whether s is a known pattern kind ID.
func isPatternKind(s string) bool {
	switch s {
	case "nilerr", "forcetypeassert", "typednil", "nil_guard", "negative_index", "literal_index", "unchecked_output":
		return true
	}
	return false
}
