package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// These package-local contracts describe the dynamic type of the first
// interface result. Nil, mixed, named, generic and deferred results stay unknown.
func concreteReturnTypes(pkg *packages.Package, functions map[types.Object]resourceFunction) map[types.Object]types.Type {
	known := map[types.Object]types.Type{}
	for object, function := range functions {
		if typ := concreteReturnType(pkg, function); typ != nil {
			known[object] = typ
		}
	}
	return known
}

func concreteReturnType(pkg *packages.Package, function resourceFunction) types.Type {
	if !interfaceReturnSignature(function.object.Type().(*types.Signature)) || pdg.ReachableProjection(function.graph) == nil {
		return nil
	}
	typ, returns := concreteReturnCandidate(function.declaration, pkg.TypesInfo)
	if typ == nil || function.query == nil {
		return nil
	}
	source := releaseGraph{graph: function.graph, query: function.query, nodes: releaseSourceNodes(function.graph), fset: pkg.Fset}
	if !concreteReturnOperands(source, returns) {
		return nil
	}
	return typ
}

func interfaceReturnSignature(signature *types.Signature) bool {
	if signature.Results().Len() == 0 || signature.TypeParams().Len()+signature.RecvTypeParams().Len() != 0 {
		return false
	}
	result := signature.Results().At(0)
	_, iface := result.Type().Underlying().(*types.Interface)
	return iface && result.Name() == ""
}

func concreteReturnCandidate(fn *ast.FuncDecl, info *types.Info) (types.Type, []*ast.ReturnStmt) {
	returns, valid := concreteReturnStatements(fn)
	if !valid || len(returns) == 0 {
		return nil, nil
	}
	typ := concreteReturnExpression(returns[0], info)
	for _, statement := range returns {
		if !matchingConcreteReturn(typ, concreteReturnExpression(statement, info)) {
			return nil, nil
		}
	}
	return typ, returns
}

func concreteReturnStatements(fn *ast.FuncDecl) ([]*ast.ReturnStmt, bool) {
	var returns []*ast.ReturnStmt
	valid := true
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.DeferStmt:
			valid = false
		case *ast.ReturnStmt:
			returns = append(returns, node)
		}
		return valid
	})
	return returns, valid
}

func matchingConcreteReturn(previous, next types.Type) bool {
	return previous != nil && next != nil && types.Identical(previous, next)
}

func concreteReturnExpression(statement *ast.ReturnStmt, info *types.Info) types.Type {
	if len(statement.Results) == 0 {
		return nil
	}
	return concreteInterfaceValueType(info.TypeOf(statement.Results[0]))
}

func concreteInterfaceValueType(typ types.Type) types.Type {
	if typ == nil {
		return nil
	}
	typ = types.Unalias(typ)
	switch typ.(type) {
	case *types.TypeParam, *types.Tuple:
		return nil
	}
	if _, iface := typ.Underlying().(*types.Interface); iface {
		return nil
	}
	if basic, ok := typ.(*types.Basic); ok && basic.Info()&types.IsUntyped != 0 {
		return nil
	}
	return typ
}

// Compiler types identify each explicit result. Canonical inputs must agree
// with that expression, and at least one modeled return must be reachable.
func concreteReturnOperands(source releaseGraph, returns []*ast.ReturnStmt) bool {
	reachable := false
	for _, statement := range returns {
		ref, value := source.ref(statement), source.ref(statement.Results[0])
		if ref == 0 || value == 0 || source.query.Operand(ref, 1) != value {
			return false
		}
		reachable = reachable || source.query.Reachable(ref)
	}
	return reachable
}
