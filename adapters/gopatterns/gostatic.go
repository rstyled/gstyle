package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
)

// gostaticanalysis extractors: nilerr, forcetypeassert, typednil.

var errorType = types.Universe.Lookup("error").Type().Underlying().(*types.Interface)

// nilErrCheck holds the parsed `if err != nil` / `if err == nil` condition.
type nilErrCheck struct {
	errVal   ast.Expr
	isNotNil bool
}

// findNilErrHits finds `if err != nil { return nil }` and
// the equivalent `if err == nil { ... } else { return nil }` pattern.
func findNilErrHits(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info) []patterns.NilErrHit {
	var out []patterns.NilErrHit
	if fn.Body == nil {
		return out
	}
	errPositions := errorResultPositions(fn, info)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		if hit := checkNilErrIf(ifStmt, fset, info, errPositions); hit != nil {
			out = append(out, *hit)
		}
		return true
	})
	return out
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

// checkNilErrIf checks one if statement for the nilerr pattern.
func checkNilErrIf(ifStmt *ast.IfStmt, fset *token.FileSet, info *types.Info, errPositions []int) *patterns.NilErrHit {
	check := parseNilErrCond(ifStmt.Cond, info)
	if check == nil {
		return nil
	}
	branch := nilErrBranch(ifStmt, check.isNotNil)
	if branch == nil {
		return nil
	}
	if !branchReturnsNilErrors(branch, errPositions, info) || usesErrorValue(branch, check.errVal, info) {
		return nil
	}
	return &patterns.NilErrHit{
		Line:     fset.Position(ifStmt.Pos()).Line,
		CondText: condText(fset, ifStmt.Cond),
		Kind:     "returns nil when err != nil",
	}
}

// parseNilErrCond parses `err != nil` or `err == nil`, returning the error
// value and whether it's the != form.
func parseNilErrCond(cond ast.Expr, info *types.Info) *nilErrCheck {
	cond = specUnwrapParens(cond)
	if unary, ok := cond.(*ast.UnaryExpr); ok && unary.Op == token.NOT {
		check := parseNilErrCond(unary.X, info)
		if check != nil {
			check.isNotNil = !check.isNotNil
		}
		return check
	}
	return binaryNilErrCheck(cond, info)
}

func binaryNilErrCheck(cond ast.Expr, info *types.Info) *nilErrCheck {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return nil
	}
	if bin.Op == token.NEQ {
		if errVal := nilErrOperand(bin, info); errVal != nil {
			return &nilErrCheck{errVal: errVal, isNotNil: true}
		}
	} else if bin.Op == token.EQL {
		if errVal := nilErrOperand(bin, info); errVal != nil {
			return &nilErrCheck{errVal: errVal, isNotNil: false}
		}
	}
	return nil
}

// nilErrOperand returns the error-valued side of `err != nil` / `err == nil`.
func nilErrOperand(bin *ast.BinaryExpr, info *types.Info) ast.Expr {
	if typedNilExpr(bin.Y, info) && isErrorExpr(info, bin.X) {
		return bin.X
	}
	if typedNilExpr(bin.X, info) && isErrorExpr(info, bin.Y) {
		return bin.Y
	}
	return nil
}

// nilErrBranch returns the branch to check: Body for !=, Else for ==.
func nilErrBranch(ifStmt *ast.IfStmt, isNotNil bool) ast.Stmt {
	if isNotNil {
		return ifStmt.Body
	}
	return ifStmt.Else
}

// isNil reports whether e is the nil identifier.
func isNilIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "nil"
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

// usesErrorValue reports whether the error value is used in the branch
// (e.g. log(err)). If used, it's not a nilerr bug.
func usesErrorValue(branch ast.Stmt, errVal ast.Expr, info *types.Info) bool {
	errID, _ := specUnwrapParens(errVal).(*ast.Ident)
	if errID == nil {
		return false
	}
	errObject := info.ObjectOf(errID)
	used := false
	ast.Inspect(branch, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok && callUsesObject(call, errObject, info) {
			used = true
		}

		return true
	})
	return used
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

func typedNilExpr(expr ast.Expr, info *types.Info) bool {
	expr = specUnwrapParens(expr)
	if info == nil {
		return isNilIdent(expr)
	}
	return info.Types[expr].IsNil()
}

func branchReturnsNilErrors(branch ast.Stmt, positions []int, info *types.Info) bool {
	found := false
	ast.Inspect(branch, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		ret, ok := n.(*ast.ReturnStmt)
		if ok && nilErrorResults(ret, positions, info) {
			found = true
		}
		return true
	})
	return found
}

func nilErrorResults(ret *ast.ReturnStmt, positions []int, info *types.Info) bool {
	if len(positions) == 0 {
		return false
	}
	for _, pos := range positions {
		if pos >= len(ret.Results) || !typedNilExpr(ret.Results[pos], info) {
			return false
		}
	}
	return true
}

func callUsesObject(call *ast.CallExpr, object types.Object, info *types.Info) bool {
	for _, arg := range call.Args {
		id, _ := specUnwrapParens(arg).(*ast.Ident)
		if id != nil && object != nil && info.ObjectOf(id) == object {
			return true
		}
	}
	return false
}
