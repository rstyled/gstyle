package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

// These package-local contracts describe the dynamic type of the first
// interface result. Nullable contracts require a caller-side non-nil proof.
// Mixed, named, generic and interface-valued results stay unknown.
type concreteReturn struct {
	typ      types.Type
	nullable bool
}

func concreteReturnTypes(pkg *packages.Package, functions map[types.Object]resourceFunction) map[types.Object]concreteReturn {
	known := map[types.Object]concreteReturn{}
	for object, function := range functions {
		if typ := concreteReturnType(pkg, function); typ.typ != nil {
			known[object] = typ
		}
	}
	return known
}

func concreteReturnType(pkg *packages.Package, function resourceFunction) concreteReturn {
	if !interfaceReturnSignature(function.object.Type().(*types.Signature)) || pdg.BodyFlow(function.graph) == nil {
		return concreteReturn{}
	}
	typ, returns := concreteReturnCandidate(function.declaration, pkg.TypesInfo)
	if typ.typ == nil || function.query == nil {
		return concreteReturn{}
	}
	source := releaseGraph{graph: function.graph, query: function.query, nodes: releaseSourceNodes(function.graph), fset: pkg.Fset}
	if !concreteReturnOperands(source, returns) {
		return concreteReturn{}
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

func concreteReturnCandidate(fn *ast.FuncDecl, info *types.Info) (concreteReturn, []*ast.ReturnStmt) {
	returns, deferred := concreteReturnStatements(fn)
	candidate := concreteReturn{typ: firstConcreteReturn(returns, info), nullable: deferred}
	for _, statement := range returns {
		if nilReturnExpression(statement, info) {
			candidate.nullable = true
			continue
		}
		if !matchingConcreteReturn(candidate.typ, concreteReturnExpression(statement, info)) {
			return concreteReturn{}, nil
		}
	}
	return candidate, returns
}

func firstConcreteReturn(returns []*ast.ReturnStmt, info *types.Info) types.Type {
	for _, statement := range returns {
		if typ := concreteReturnExpression(statement, info); typ != nil {
			return typ
		}
	}
	return nil
}

func nilReturnExpression(statement *ast.ReturnStmt, info *types.Info) bool {
	return len(statement.Results) > 0 && info.Types[statement.Results[0]].IsNil()
}

// Unnamed results are fixed before defers run. Recovery can instead return the
// zero interface result, so every deferred body makes the contract nullable.
func concreteReturnStatements(fn *ast.FuncDecl) ([]*ast.ReturnStmt, bool) {
	var returns []*ast.ReturnStmt
	deferred := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.DeferStmt:
			deferred = true
		case *ast.ReturnStmt:
			returns = append(returns, node)
		}
		return true
	})
	return returns, deferred
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

// Body flow supplies explicit return inputs before deferred unwind. The
// nullable contract accounts for recovery; named result writes are excluded.
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
