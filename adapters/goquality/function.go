package goquality

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/rstyled/gstyle/adapters/gopatterns"
	"github.com/rstyled/gstyle/domain/quality"
	"golang.org/x/tools/go/packages"
)

type extractor struct {
	receiverFields map[types.Object]bool
	closedReceiver bool
	factories      map[*types.Func]factoryResult
	pkg            *packages.Package
	stable         map[types.Object]bool
	fact           quality.Function
	bindings       map[types.Object]referenceBinding
	fields         fieldOwnership
	parameters     map[types.Object]bool
	// paramIndex maps outer-function parameter objects to their ParamList
	// index. The receiver and closure parameters are not indexed: they are
	// not forwardable as bare Param(i) arguments.
	paramIndex map[types.Object]int
	receiver   types.Object
}

func (x *extractor) addParameters(fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for _, name := range field.Names {
			x.parameters[x.pkg.TypesInfo.Defs[name]] = true
		}
	}
}

// indexParameters builds the ParamList in source order and maps each named
// parameter object to its index. Unnamed parameters keep an empty entry so
// indices stay aligned with the Params count. The receiver is excluded,
// matching the fn_params budget.
func (x *extractor) indexParameters(fields *ast.FieldList) {
	x.fact.ParamList, x.paramIndex = buildParams(fields, x.pkg.TypesInfo)
}

func (x *extractor) noteReceiver(fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for _, name := range field.Names {
			if obj := x.pkg.TypesInfo.Defs[name]; obj != nil {
				x.receiver = obj
			}
		}
	}
}

// countParam records one syntactic reference to a parameter, if the node
// is a parameter use.
func (x *extractor) countParam(node ast.Node) {
	if id, ok := node.(*ast.Ident); ok {
		x.countParamUse(id)
	}
}

// countParamUse records one syntactic reference to a parameter.
func (x *extractor) countParamUse(id *ast.Ident) {
	if index, ok := x.paramIndex[x.pkg.TypesInfo.Uses[id]]; ok {
		x.fact.ParamList[index].Uses++
	}
}

func (x *extractor) addLiteralParameters(node ast.Node) {
	if literal, ok := node.(*ast.FuncLit); ok {
		x.addParameters(literal.Type.Params)
	}
}

func (x *extractor) line(pos token.Pos) int { return physicalPosition(x.pkg.Fset, pos).Line }

func (x *extractor) visit(node ast.Node) bool {
	if countedStatement(node) {
		x.fact.Statements++
	}
	x.countParam(node)
	switch node := node.(type) {
	case *ast.AssignStmt:
		x.assignment(node)
	case *ast.RangeStmt:
		x.rangeAssignment(node)
	case *ast.IncDecStmt:
		x.mutate(node.X, node.Pos(), false)
	case *ast.CallExpr:
		x.call(node)
	default:
		x.visitEffect(node)
	}
	return true
}

func (x *extractor) visitEffect(node ast.Node) {
	switch node := node.(type) {
	case *ast.Ident:
		x.globalRead(node)
	case *ast.GoStmt:
		x.effect(quality.UnknownEffect, "goroutine launch", node.Pos())
	case *ast.SendStmt:
		x.effect(quality.UnknownEffect, "channel send", node.Pos())
	case *ast.UnaryExpr:
		if node.Op == token.ARROW {
			x.effect(quality.UnknownEffect, "channel receive", node.Pos())
		}
	}
}

func (x *extractor) assignment(statement *ast.AssignStmt) {
	for index, place := range statement.Lhs {
		if id, ok := place.(*ast.Ident); ok && x.pkg.TypesInfo.Defs[id] != nil {
			continue
		}
		before := len(x.fact.Mutations)
		x.mutate(place, statement.Pos(), false)
		if site := x.appendAssignmentSite(statement, index); site != 0 && len(x.fact.Mutations) > before {
			x.fact.Mutations[before].Site = site
		}
	}
}

func (x *extractor) appendAssignmentSite(statement *ast.AssignStmt, index int) int {
	if len(statement.Rhs) != len(statement.Lhs) {
		return 0
	}
	call, ok := ast.Unparen(statement.Rhs[index]).(*ast.CallExpr)
	if !ok {
		return 0
	}
	builtin, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Builtin)
	if !ok || builtin.Name() != "append" {
		return 0
	}
	return physicalPosition(x.pkg.Fset, call.Pos()).Offset + 1
}

func (x *extractor) rangeAssignment(statement *ast.RangeStmt) {
	if statement.Tok != token.ASSIGN {
		return
	}
	for _, place := range []ast.Expr{statement.Key, statement.Value} {
		if place != nil {
			x.mutate(place, statement.Pos(), false)
		}
	}
}

func (x *extractor) effect(kind quality.Kind, detail string, pos token.Pos) {
	x.fact.Effects = append(x.fact.Effects, quality.Effect{Kind: kind, Detail: detail, Line: x.line(pos)})
}

func (x *extractor) globalRead(id *ast.Ident) {
	object, ok := x.pkg.TypesInfo.Uses[id].(*types.Var)
	if !ok || !packageVariable(object) || x.stable[object] {
		return
	}
	// A write's global effect already captures observability. Deduplicate the
	// identifier visit at the same line (including compound assignment reads).
	line := x.line(id.Pos())
	detail := resolvedVariable(object)
	if hasGlobalEffect(x.fact.Effects, line, detail) {
		return
	}
	x.fact.Effects = append(x.fact.Effects, quality.Effect{Kind: quality.Global, Detail: detail, Line: line})
}

func (x *extractor) objectID(object types.Object) string {
	pos := physicalPosition(x.pkg.Fset, object.Pos())
	return strconv.Itoa(pos.Line) + ":" + strconv.Itoa(pos.Column)
}

func (extraction packageExtraction) extractFunction(fn *ast.FuncDecl, path string, source []byte) (quality.Function, error) {
	pkg := extraction.pkg
	start, end := physicalPosition(pkg.Fset, fn.Pos()), physicalPosition(pkg.Fset, fn.End())
	name := fn.Name.Name
	if object, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok {
		name = resolvedName(object)
	}
	x := extractor{receiverFields: extraction.receiverFields, pkg: pkg, factories: extraction.factories, stable: extraction.stable, bindings: map[types.Object]referenceBinding{}, parameters: map[types.Object]bool{}, paramIndex: map[types.Object]int{}, fact: quality.Function{
		Location: quality.Location{Path: path, Line: start.Line, Column: start.Column, Name: name},
		Params:   parameterCount(fn.Type.Params), HasSelf: fn.Recv != nil,
		Source: string(source[start.Offset:end.Offset]), PrecedingLine: precedingSuppression(fn.Doc),
		Mutations: []quality.Mutation{}, Calls: []quality.Call{}, Effects: []quality.Effect{},
	}}
	lines, err := codeLines(x.fact.Source)
	if err != nil {
		return quality.Function{}, fmt.Errorf("count code lines for %s: %w", name, err)
	}
	x.fact.CodeLines = lines
	x.indexParameters(fn.Type.Params)
	x.noteReceiver(fn.Recv)
	if x.receiver != nil {
		x.closedReceiver = extraction.receivers[receiverType(x.receiver.Type())]
	}
	x.addParameters(fn.Type.Params)
	x.addParameters(fn.Recv)
	x.collectBindings(fn.Body)
	ast.Inspect(fn.Body, x.visit)
	x.fact.DDD = x.dddViolations(fn, path)
	return x.fact, nil
}

// dddViolations runs the DDD detectors (Evans) for the quality gate:
// aggregate boundaries, repository bypasses, and mutable identities.
func (x *extractor) dddViolations(fn *ast.FuncDecl, path string) []quality.DDDViolation {
	var out []quality.DDDViolation
	// Aggregate boundaries concern writes to shared or uncertain storage.
	// Building an owned local value does not modify another aggregate.
	pkg := x.pkg
	include := func(place ast.Expr) bool {
		object, _ := x.placeRoot(place)
		if object != nil && packageVariable(object) {
			return true
		}
		return x.placeProvenance(place) != quality.Local
	}
	for _, m := range gopatterns.FindAggregateModsForWrites(fn, pkg.TypesInfo, include) {
		out = append(out, quality.DDDViolation{
			RuleID: "aggregate",
			Line:   pkg.Fset.Position(fn.Pos()).Line,
			Detail: m.TypeName,
		})
	}
	// Repository: direct db calls outside repository files.
	for _, d := range gopatterns.FindDbCalls(fn, pkg.Fset, pkg.TypesInfo, path) {
		out = append(out, quality.DDDViolation{
			RuleID: "repository",
			Line:   d.Line,
			Detail: fmt.Sprintf("direct db call %s in business logic", d.Call),
		})
	}
	// Mutable identity: ID assignments outside constructors.
	for _, m := range gopatterns.FindMutableIdentities(fn, pkg.Fset) {
		out = append(out, quality.DDDViolation{
			RuleID: "mutable_identity",
			Line:   m.Line,
			Detail: fmt.Sprintf("assigns .%s outside a constructor", m.Field),
		})
	}
	return out
}

func parameterCount(fields *ast.FieldList) int {
	if fields == nil {
		return 0
	}
	count := 0
	for _, field := range fields.List {
		count += max(len(field.Names), 1)
	}
	return count
}

// buildParams is pure: it returns the parameter list and object→index map
// without mutating shared state.
func buildParams(fields *ast.FieldList, info *types.Info) ([]quality.ParamFacts, map[types.Object]int) {
	var list []quality.ParamFacts
	index := map[types.Object]int{}
	if fields == nil {
		return list, index
	}
	for _, field := range fields.List {
		typ := paramType(field.Type, info)
		if len(field.Names) == 0 {
			list = append(list, quality.ParamFacts{Type: typ})
			continue
		}
		for _, name := range field.Names {
			if obj := info.Defs[name]; obj != nil {
				index[obj] = len(list)
			}
			list = append(list, quality.ParamFacts{Name: name.Name, Type: typ})
		}
	}
	return list, index
}

func paramType(expr ast.Expr, info *types.Info) string {
	typ := info.TypeOf(expr)
	if typ == nil {
		return ""
	}
	return types.TypeString(typ, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Name()
	})
}

func countedStatement(node ast.Node) bool {
	switch node.(type) {
	case *ast.BlockStmt, *ast.EmptyStmt, *ast.CaseClause, *ast.CommClause, *ast.LabeledStmt:
		return false
	default:
		_, ok := node.(ast.Stmt)
		return ok
	}
}

func hasGlobalEffect(effects []quality.Effect, line int, detail string) bool {
	for _, effect := range effects {
		if effect.Kind == quality.Global && effect.Line == line && effect.Detail == detail {
			return true
		}
	}
	return false
}

func packageVariable(object types.Object) bool {
	return object.Pkg() != nil && object.Parent() == object.Pkg().Scope()
}

func resolvedVariable(object types.Object) string { return object.Pkg().Path() + "." + object.Name() }

func resolvedName(function *types.Func) string {
	name := function.Name()
	signature := function.Type().(*types.Signature)
	if receiver := signature.Recv(); receiver != nil {
		typ := types.Unalias(receiver.Type())
		if pointer, ok := typ.(*types.Pointer); ok {
			typ = types.Unalias(pointer.Elem())
		}
		if named, ok := typ.(*types.Named); ok {
			name = named.Obj().Name() + "." + name
		}
	}
	if function.Pkg() == nil {
		return name
	}
	return function.Pkg().Path() + "." + name
}

func precedingSuppression(comments *ast.CommentGroup) string {
	if comments == nil {
		return ""
	}
	for index := len(comments.List) - 1; index >= 0; index-- {
		text := comments.List[index].Text
		if strings.Contains(text, "gstyle-allow") || strings.HasPrefix(text, "//lint:ignore") {
			return text
		}
	}
	return ""
}
