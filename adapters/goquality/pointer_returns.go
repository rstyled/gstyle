package goquality

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
)

// Pointer certificates describe outer result identity, never helper purity or
// ownership of children. All returns must agree; unresolved cycles stay unknown.
func (extraction packageExtraction) pointerFactory(fn *ast.FuncDecl) factoryResult {
	function := extraction.pkg.TypesInfo.Defs[fn.Name].(*types.Func)
	signature := function.Type().(*types.Signature)
	if !pointerResultSignature(fn, signature) {
		return factoryResult{}
	}
	x := extractor{pkg: extraction.pkg, factories: extraction.factories, cappedReturns: extraction.cappedReturns, bindings: map[types.Object]referenceBinding{}, parameters: map[types.Object]bool{}}
	x.addParameters(fn.Type.Params)
	x.addParameters(fn.Recv)
	x.collectBindings(fn.Body)
	return x.pointerReturns(fn, signature)
}

func pointerResultSignature(fn *ast.FuncDecl, signature *types.Signature) bool {
	if fn.Body == nil || signature.Variadic() || signature.Results().Len() != 1 {
		return false
	}
	_, pointer := underlying(signature.Results().At(0).Type()).(*types.Pointer)
	return pointer && signature.Results().At(0).Name() == ""
}

func (x *extractor) pointerReturns(fn *ast.FuncDecl, signature *types.Signature) factoryResult {
	var result factoryResult
	valid := true
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			candidate := x.pointerReturn(node, fn, signature)
			valid = valid && candidate.pointer && samePointerResult(result, candidate)
			result = candidate
		}
		return valid
	})
	if !valid {
		return factoryResult{}
	}
	return result
}

func samePointerResult(previous, next factoryResult) bool {
	if !previous.pointer {
		return true
	}
	if previous.parameter == nil || next.parameter == nil {
		return previous.parameter == nil && next.parameter == nil
	}
	return *previous.parameter == *next.parameter
}

func (x *extractor) pointerReturn(statement *ast.ReturnStmt, fn *ast.FuncDecl, signature *types.Signature) factoryResult {
	if len(statement.Results) != 1 {
		return factoryResult{}
	}
	return x.pointerExpression(statement.Results[0], fn, signature)
}

func (x *extractor) pointerExpression(value ast.Expr, fn *ast.FuncDecl, signature *types.Signature) factoryResult {
	switch value := ast.Unparen(value).(type) {
	case *ast.Ident:
		return x.pointerFormal(value, fn, signature)
	case *ast.UnaryExpr:
		if value.Op == token.AND && x.freshPointerAddress(value.X, fn, signature) {
			return factoryResult{value: fn.Name, pointer: true}
		}
	case *ast.CallExpr:
		return x.pointerCallResult(value, fn, signature)
	}
	return factoryResult{}
}

func (x *extractor) pointerFormal(id *ast.Ident, fn *ast.FuncDecl, signature *types.Signature) factoryResult {
	object := x.pkg.TypesInfo.ObjectOf(id)
	if _, rebound := x.bindings[object]; rebound {
		return factoryResult{}
	}
	for index := range signature.Params().Len() {
		if signature.Params().At(index) == object && x.pointerObjectClosed(object, map[types.Object]bool{}) {
			return factoryResult{value: fn.Name, parameter: &index, pointer: true}
		}
	}
	return factoryResult{}
}

func (x *extractor) freshPointerAddress(value ast.Expr, fn *ast.FuncDecl, signature *types.Signature) bool {
	if _, literal := ast.Unparen(value).(*ast.CompositeLit); literal {
		return true
	}
	if len(fn.Body.List) != 1 {
		return false
	}
	id, ok := ast.Unparen(value).(*ast.Ident)
	if !ok {
		return false
	}
	object := x.pkg.TypesInfo.ObjectOf(id)
	for parameter := range signature.Params().Variables() {
		if parameter == object {
			return true // Address of a by-value formal is a fresh outer slot.
		}
	}
	return false
}

func (x *extractor) pointerCallResult(call *ast.CallExpr, fn *ast.FuncDecl, signature *types.Signature) factoryResult {
	object := x.calledObject(unwrappedCallee(call.Fun))
	if builtin, ok := object.(*types.Builtin); ok {
		if builtin.Name() == "new" {
			return factoryResult{value: fn.Name, pointer: true}
		}
		return factoryResult{}
	}
	function, ok := object.(*types.Func)
	if !ok {
		return factoryResult{}
	}
	result := x.factories[function.Origin()]
	if !result.pointer {
		return factoryResult{}
	}
	if result.parameter == nil {
		return factoryResult{value: fn.Name, pointer: true}
	}
	return x.pointerExpression(x.factoryArgument(call, result), fn, signature)
}

func (x *extractor) pointerReturnArgument(call *ast.CallExpr) ast.Expr {
	function, ok := x.calledObject(unwrappedCallee(call.Fun)).(*types.Func)
	if !ok {
		return nil
	}
	result := x.factories[function.Origin()]
	if !result.pointer || result.parameter == nil {
		return nil
	}
	return x.factoryArgument(call, result)
}

func (x *extractor) pointerFactoryOrigin(call *ast.CallExpr, result factoryResult, seen map[types.Object]bool) quality.Provenance {
	origin := quality.Local
	if result.parameter != nil {
		argument := x.factoryArgument(call, result)
		origin = x.origin(argument, seen)
		if origin == quality.Local && !x.pointerInputClosed(argument) {
			return quality.Unknown
		}
	}
	if origin == quality.Local && !x.pointerValueClosed(call, map[types.Object]bool{}) {
		return quality.Unknown
	}
	return origin
}
