package gopatterns

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

type nilErrValue struct {
	scalar          constant.Value
	boolean         types.Object
	negated         bool
	nilValue, empty bool
}

func (e nilErrEvidence) value(expr ast.Expr) nilErrValue {
	expr = ast.Unparen(expr)
	typed := e.info.Types[expr]
	if typed.IsNil() {
		return nilErrValue{nilValue: true}
	}
	if typed.Value != nil {
		return nilErrValue{scalar: typed.Value}
	}
	if name, ok := expr.(*ast.Ident); ok {
		return e.nameValue(name)
	}
	return e.derivedValue(expr)
}

func (e nilErrEvidence) derivedValue(expr ast.Expr) nilErrValue {
	if call, ok := expr.(*ast.CallExpr); ok {
		return e.convertedValue(call)
	}
	if value, known := e.booleanExpression(expr); known {
		return nilErrValue{scalar: constant.MakeBool(value)}
	}
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.NOT {
		value := e.value(unary.X)
		value.negated = !value.negated
		return value
	}
	return nilErrValue{empty: emptyNilErrResult(expr, e.info)}
}

func (e nilErrEvidence) convertedValue(call *ast.CallExpr) nilErrValue {
	if nilErrNilConversion(call, e.info) {
		return nilErrValue{nilValue: true}
	}
	if nilErrIdentityConversion(call, e.info) {
		return e.value(call.Args[0])
	}
	return nilErrValue{}
}

func (v nilErrValue) zero() bool {
	if v.nilValue || v.empty {
		return true
	}
	if v.scalar == nil {
		return false
	}
	switch v.scalar.Kind() {
	case constant.Bool:
		return !constant.BoolVal(v.scalar)
	case constant.String:
		return constant.StringVal(v.scalar) == ""
	default:
		return constant.Sign(v.scalar) == 0
	}
}

func (e nilErrEvidence) lostReturn(statement *ast.ReturnStmt) bool {
	if len(statement.Results) == 0 {
		return e.emptyNamedResults()
	}
	if len(e.positions) == 0 {
		return false
	}
	return e.nilErrorResults(statement.Results) && e.emptyResults(statement.Results)
}

func (e nilErrEvidence) emptyResults(results []ast.Expr) bool {
	for i, result := range results {
		if !nilErrAssignableZero(e.resultValue(result, i), e.results[i].Type()) {
			return false
		}
	}
	return true
}

func (e nilErrEvidence) emptyNamedResults() bool {
	if len(e.results) == 0 || len(e.positions) == 0 {
		return false
	}
	if !e.nilNamedErrors() {
		return false
	}
	for _, result := range e.results {
		if !e.values[result].zero() {
			return false
		}
	}
	return true
}

func (e nilErrEvidence) nilNamedErrors() bool {
	for _, position := range e.positions {
		if !e.values[e.results[position]].nilValue {
			return false
		}
	}
	return true
}

func (e nilErrEvidence) boolean(expr ast.Expr) (bool, bool) {
	expr = ast.Unparen(expr)
	if truth, known := e.knownErrorCondition(expr); known {
		return truth, true
	}

	value := e.value(expr).scalar
	if value != nil && value.Kind() == constant.Bool {
		return constant.BoolVal(value), true
	}
	return e.booleanExpression(expr)
}

func (e nilErrEvidence) booleanExpression(expr ast.Expr) (bool, bool) {
	switch expr := expr.(type) {
	case *ast.UnaryExpr:
		if expr.Op != token.NOT {
			return false, false
		}
		truth, known := e.boolean(expr.X)
		return !truth, known
	case *ast.BinaryExpr:
		return e.booleanBinary(expr)
	default:
		return false, false
	}
}

func (e nilErrEvidence) booleanBinary(expr *ast.BinaryExpr) (bool, bool) {
	if expr.Op == token.LAND || expr.Op == token.LOR {
		return e.booleanLogical(expr)
	}
	left, right := e.value(expr.X).scalar, e.value(expr.Y).scalar
	if left == nil || right == nil {
		return false, false
	}
	if !nilErrComparison(expr.Op) {
		return false, false
	}
	return constant.Compare(left, expr.Op, right), true
}

func nilErrComparison(operator token.Token) bool {
	return operator == token.EQL || operator == token.NEQ || operator == token.LSS || operator == token.LEQ || operator == token.GTR || operator == token.GEQ
}

func (e nilErrEvidence) booleanLogical(expr *ast.BinaryExpr) (bool, bool) {
	left, lk := e.boolean(expr.X)
	if result, known := nilErrShortCircuit(expr.Op, left, lk); known {
		return result, true
	}
	right, rk := e.boolean(expr.Y)
	if result, known := nilErrShortCircuit(expr.Op, right, rk); known {
		return result, true
	}
	return nilErrLogicalResult(expr.Op, left, right), lk && rk
}

func nilErrShortCircuit(operator token.Token, value, known bool) (bool, bool) {
	if operator == token.LAND {
		return false, known && !value
	}
	return true, known && value
}

func nilErrLogicalResult(operator token.Token, left, right bool) bool {
	if operator == token.LAND {
		return left && right
	}
	return left || right
}

func nilErrBooleanName(expr ast.Expr, info *types.Info) bool {
	expr = ast.Unparen(expr)
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.NOT {
		return nilErrBooleanName(unary.X, info)
	}
	name, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	basic, ok := info.TypeOf(name).Underlying().(*types.Basic)
	return ok && basic.Kind() == types.Bool
}

func (e *nilErrEvidence) assume(expr ast.Expr, truth bool) {
	if check := parseNilErrCond(expr, e.info); check != nil {
		check.isNotNil = check.isNotNil == truth
		e.assumeError(*check)
		return
	}
	expr = ast.Unparen(expr)
	if unary, ok := expr.(*ast.UnaryExpr); ok {
		e.assume(unary.X, !truth)
		return
	}
	value := e.value(expr)
	if value.boolean != nil {
		e.truths[value.boolean] = truth != value.negated
	}
}

func (e nilErrEvidence) knownErrorCondition(expr ast.Expr) (bool, bool) {
	check := parseNilErrCond(expr, e.info)
	if check == nil {
		return false, false
	}
	for i := len(e.guards) - 1; i >= 0; i-- {
		fact := e.guards[i]
		if sameNilErrValue(check.errVal, fact.errVal, e.info) {
			return check.isNotNil == fact.isNotNil, true
		}
	}
	if e.value(check.errVal).nilValue {
		return !check.isNotNil, true
	}
	return false, false
}

func (e *nilErrEvidence) record(node ast.Node) {
	if node == nil {
		return
	}
	ast.Inspect(node, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		e.recordNode(node)
		return true
	})
}

func (e *nilErrEvidence) recordNode(node ast.Node) {
	switch node := node.(type) {
	case *ast.AssignStmt:
		e.recordAssignment(node)
	case *ast.ValueSpec:
		e.recordDeclaration(node)
	}
}

func (e *nilErrEvidence) recordAssignment(statement *ast.AssignStmt) {
	values := make([]nilErrValue, len(statement.Lhs))
	if len(statement.Lhs) == len(statement.Rhs) {
		for i, value := range statement.Rhs {
			values[i] = e.assignedValue(value, e.info.TypeOf(statement.Lhs[i]))
		}
	}
	for i, target := range statement.Lhs {
		e.store(target, values[i])
	}
}

func (e *nilErrEvidence) store(target ast.Expr, value nilErrValue) {
	name, ok := ast.Unparen(target).(*ast.Ident)
	if !ok {
		return
	}
	object := e.info.ObjectOf(name)
	if object == nil {
		return
	}
	e.invalidateGuards(target)
	e.values[object] = value
}

func (e *nilErrEvidence) recordDeclaration(declaration *ast.ValueSpec) {
	for i, name := range declaration.Names {
		value := nilErrZeroType(e.info.TypeOf(name))
		if len(declaration.Values) > 0 {
			value = e.declarationValue(declaration, i)
		}
		e.store(name, value)
	}
}

func (e nilErrEvidence) declarationValue(declaration *ast.ValueSpec, index int) nilErrValue {
	if len(declaration.Names) != len(declaration.Values) {
		return nilErrValue{}
	}
	return e.assignedValue(declaration.Values[index], e.info.TypeOf(declaration.Names[index]))
}

func nilErrZeroType(typ types.Type) nilErrValue {
	if _, generic := types.Unalias(typ).(*types.TypeParam); generic {
		return nilErrValue{}
	}
	switch typ := typ.Underlying().(type) {
	case *types.Basic:
		return nilErrZeroBasic(typ)
	case *types.Array, *types.Struct:
		return nilErrValue{empty: true}
	default:
		return nilErrValue{nilValue: true}
	}
}

func nilErrZeroBasic(typ *types.Basic) nilErrValue {
	if typ.Info()&types.IsBoolean != 0 {
		return nilErrValue{scalar: constant.MakeBool(false)}
	}
	if typ.Info()&types.IsString != 0 {
		return nilErrValue{scalar: constant.MakeString("")}
	}
	return nilErrValue{scalar: constant.MakeInt64(0)}
}

func (e nilErrEvidence) nilErrorResults(results []ast.Expr) bool {
	for _, position := range e.positions {
		if position >= len(results) || !e.resultValue(results[position], position).nilValue {
			return false
		}
	}
	return true
}

func nilErrConstantCondition(expr ast.Expr, info *types.Info) (bool, bool) {
	evidence := nilErrEvidence{info: info, values: map[types.Object]nilErrValue{}, truths: map[types.Object]bool{}}
	if evidence.unknownHandling(expr) {
		return false, false
	}
	return evidence.boolean(expr)
}

func (e *nilErrEvidence) invalidateGuards(target ast.Expr) {
	var retained []nilErrCheck
	for _, fact := range e.guards {
		if !replacesNilErrValue(target, fact.errVal, e.info) {
			retained = append(retained, fact)
		}
	}
	e.guards = retained
}

func nilErrAssignableZero(value nilErrValue, typ types.Type) bool {
	if _, generic := types.Unalias(typ).(*types.TypeParam); generic {
		return false
	}
	if _, iface := typ.Underlying().(*types.Interface); iface {
		return value.nilValue
	}
	return value.zero()
}

func (e nilErrEvidence) nameValue(name *ast.Ident) nilErrValue {
	object := e.info.ObjectOf(name)
	value, known := e.values[object]
	if !known && nilErrBooleanName(name, e.info) {
		value.boolean = object
	}
	if truth, known := e.truths[value.boolean]; known {
		return nilErrValue{scalar: constant.MakeBool(truth != value.negated)}
	}
	return value
}

func nilErrIdentityConversion(call *ast.CallExpr, info *types.Info) bool {
	if len(call.Args) != 1 || !info.Types[call.Fun].IsType() {
		return false
	}
	return types.Identical(info.TypeOf(call), info.TypeOf(call.Args[0]))
}

func (e nilErrEvidence) resultValue(expr ast.Expr, index int) nilErrValue {
	return e.assignedValue(expr, e.results[index].Type())
}

// A nil pointer boxed into an interface is non-nil. Only a nil interface or
// untyped nil transfers whole-interface nilness across an assignment/return.
func (e nilErrEvidence) assignedValue(expr ast.Expr, target types.Type) nilErrValue {
	value := e.value(expr)
	if target == nil {
		return nilErrValue{}
	}
	if e.info.Types[ast.Unparen(expr)].IsNil() {
		return value
	}
	if _, iface := target.Underlying().(*types.Interface); !iface {
		return value
	}
	source := e.info.TypeOf(expr)
	if _, generic := types.Unalias(source).(*types.TypeParam); generic {
		return nilErrValue{}
	}
	if _, iface := source.Underlying().(*types.Interface); !iface {
		return nilErrValue{}
	}
	return value
}

func (e *nilErrEvidence) assumeError(fact nilErrCheck) {
	e.guards = append(append([]nilErrCheck{}, e.guards...), fact)
	location := nilErrLocationOf(fact.errVal, e.info)
	if location.root != nil && len(location.fields) == 0 {
		e.values[location.root] = nilErrValue{nilValue: !fact.isNotNil}
	}
}

func nilErrNilConversion(call *ast.CallExpr, info *types.Info) bool {
	return len(call.Args) == 1 && info.Types[call.Fun].IsType() && info.Types[ast.Unparen(call.Args[0])].IsNil()
}
