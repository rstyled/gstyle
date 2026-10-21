package goquality

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// stableGlobals proves reads of private literal tables, unit values, and modeled values do not
// depend on mutable state. All uses in the active package must preserve value reads.
func stableGlobals(pkg *packages.Package) map[types.Object]bool {
	stable := map[types.Object]bool{}
	if opaqueGlobals(pkg) {
		return stable
	}
	unitGlobals(pkg.TypesInfo, stable)
	parents := map[ast.Node]ast.Node{}
	for _, file := range pkg.Syntax {
		indexParents(file, parents)
		stableDeclarations(pkg, file, stable)
	}
	for id, object := range pkg.TypesInfo.Uses {
		if stable[object] && !stableGlobalRead(id, parents, pkg.TypesInfo) {
			delete(stable, object)
		}
	}
	return stable
}

func stableDeclarations(pkg *packages.Package, file *ast.File, stable map[types.Object]bool) {
	ast.Inspect(file, func(node ast.Node) bool {
		if spec, ok := node.(*ast.ValueSpec); ok {
			stableCandidates(pkg, spec, stable)
		}
		return true
	})
}

func opaqueGlobals(pkg *packages.Package) bool {
	if len(pkg.OtherFiles) > 0 {
		return true
	}
	if _, unsafe := pkg.Imports["unsafe"]; unsafe {
		return true
	}
	for _, file := range pkg.Syntax {
		if linkedGlobals(file) {
			return true
		}
	}
	return false
}

func linkedGlobals(file *ast.File) bool {
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:linkname") {
				return true
			}
		}
	}
	return false
}

func indexParents(node ast.Node, parents map[ast.Node]ast.Node) {
	var stack []ast.Node
	ast.Inspect(node, func(child ast.Node) bool {
		if child == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[child] = stack[len(stack)-1]
		}
		stack = append(stack, child)
		return true
	})
}

func stableCandidates(pkg *packages.Package, spec *ast.ValueSpec, stable map[types.Object]bool) {
	if len(spec.Names) != len(spec.Values) {
		return
	}
	for index, name := range spec.Names {
		if stableVariable(pkg.TypesInfo.Defs[name], spec.Values[index], pkg) {
			stable[pkg.TypesInfo.Defs[name]] = true
		}
	}
}

func stableVariable(object types.Object, value ast.Expr, pkg *packages.Package) bool {
	variable, ok := object.(*types.Var)
	if !ok || variable.Exported() || !packageVariable(variable) {
		return false
	}
	return stableInitializer(variable.Type(), value, pkg.TypesInfo) ||
		(styleValueType(variable.Type()) && stableStyle(pkg, value))
}

func stableInitializer(typ types.Type, value ast.Expr, info *types.Info) bool {
	if stableErrorInitializer(value, info) || stableMetadataInitializer(typ, value, info) {
		return true
	}
	literal, ok := value.(*ast.CompositeLit)
	return ok && (stableTableType(typ) || scalarValue(typ)) && constantLiteral(literal, info)
}

func stableTableType(typ types.Type) bool {
	switch typ := typ.Underlying().(type) {
	case *types.Map:
		return scalarValue(typ.Key()) && scalarValue(typ.Elem())
	case *types.Slice:
		return scalarValue(typ.Elem())
	case *types.Array:
		return scalarValue(typ.Elem())
	}
	return false
}

func scalarValue(typ types.Type) bool {
	switch typ := typ.Underlying().(type) {
	case *types.Basic:
		return typ.Kind() != types.UnsafePointer
	case *types.Array:
		return scalarValue(typ.Elem())
	case *types.Struct:
		for field := range typ.Fields() {
			if !scalarValue(field.Type()) {
				return false
			}
		}
		return true
	}
	// A copied function value cannot replace the table's entry. Calls retain
	// their own effect evidence, independently of reading a fixed dispatch table.
	_, function := typ.Underlying().(*types.Signature)
	return function
}

func constantLiteral(value ast.Expr, info *types.Info) bool {
	if info.Types[value].Value != nil || fixedFunctionValue(value, info) {
		return true
	}
	literal, ok := value.(*ast.CompositeLit)
	if !ok {
		return false
	}
	for _, element := range literal.Elts {
		if !constantElement(element, info) {
			return false
		}
	}
	return true
}

func fixedFunctionValue(value ast.Expr, info *types.Info) bool {
	switch value := ast.Unparen(value).(type) {
	case *ast.FuncLit:
		return true // Captured state belongs to invocation effects, not table reads.
	case *ast.Ident:
		_, ok := info.ObjectOf(value).(*types.Func)
		return ok
	case *ast.SelectorExpr:
		_, ok := info.ObjectOf(value.Sel).(*types.Func)
		return ok && info.Selections[value] == nil
	}
	return false
}

func constantElement(value ast.Expr, info *types.Info) bool {
	pair, ok := value.(*ast.KeyValueExpr)
	if !ok {
		return constantLiteral(value, info)
	}
	// Struct keys are field names, not evaluated expressions.
	if id, ok := pair.Key.(*ast.Ident); ok {
		if field, ok := info.Uses[id].(*types.Var); ok && field.IsField() {
			return constantLiteral(pair.Value, info)
		}
	}
	return constantLiteral(pair.Key, info) && constantLiteral(pair.Value, info)
}

func stableTableRead(id *ast.Ident, parents map[ast.Node]ast.Node, info *types.Info) bool {
	node, parent := (receiverProof{parents: parents}).parent(id)
	switch parent := parent.(type) {
	case *ast.RangeStmt:
		return parent.X == node
	case *ast.IndexExpr:
		return parent.X == node && copiedTableValue(parent, parents, info)
	case *ast.CallExpr:
		return tableLength(parent, node, info) || tableContains(parent, node, info)
	}
	return false
}

func tableLength(call *ast.CallExpr, node ast.Node, info *types.Info) bool {
	name, ok := call.Fun.(*ast.Ident)
	if !ok || len(call.Args) != 1 || call.Args[0] != node {
		return false
	}
	builtin, ok := info.Uses[name].(*types.Builtin)
	return ok && (builtin.Name() == "len" || builtin.Name() == "cap")
}

// Contains neither retains the table nor invokes its elements.
func tableContains(call *ast.CallExpr, node ast.Node, info *types.Info) bool {
	if len(call.Args) != 2 || call.Args[0] != node {
		return false
	}
	selector, ok := unwrappedCallee(call.Fun).(*ast.SelectorExpr)
	if !ok || info.Selections[selector] != nil {
		return false
	}
	function, ok := info.ObjectOf(selector.Sel).(*types.Func)
	return ok && function.FullName() == "slices.Contains"
}

func copiedTableValue(node ast.Node, parents map[ast.Node]ast.Node, info *types.Info) bool {
	for parent := parents[node]; parent != nil; parent = parents[node] {
		// An index/key consumes the copied value; the indexed target is separate.
		if index, ok := parent.(*ast.IndexExpr); ok && index.Index == node {
			return true
		}
		if !tableValueContext(node, parent, info) {
			return false
		}
		node = parent
	}
	return true
}

func tableValueContext(node, parent ast.Node, info *types.Info) bool {
	if tableWritePlace(node, parent) {
		return false
	}
	switch parent := parent.(type) {
	case *ast.UnaryExpr:
		return parent.Op != token.AND
	case *ast.SliceExpr:
		return false
	case *ast.SelectorExpr:
		selection := info.Selections[parent]
		return selection == nil || selection.Kind() == types.FieldVal
	}
	return true
}

func tableWritePlace(node, parent ast.Node) bool {
	switch parent := parent.(type) {
	case *ast.AssignStmt:
		return assignmentPlace(node, parent)
	case *ast.RangeStmt:
		return parent.Tok == token.ASSIGN && (parent.Key == node || parent.Value == node)
	case *ast.IncDecStmt:
		return true
	}
	return false
}

func assignmentPlace(node ast.Node, statement *ast.AssignStmt) bool {
	for _, place := range statement.Lhs {
		if place == node {
			return true
		}
	}
	return false
}
