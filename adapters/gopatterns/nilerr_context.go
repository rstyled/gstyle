package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
)

func (e *nilErrEvidence) initialize(fn *ast.FuncDecl, before token.Pos) {
	signature := e.info.Defs[fn.Name].Type().(*types.Signature)
	writes := nilErrWrites(fn, e.info)
	e.initializeResults(signature, writes)
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if node == nil || node.Pos() >= before {
			return false
		}
		e.stableDefinition(node, writes)
		return true
	})
}

func nilErrWrites(fn *ast.FuncDecl, info *types.Info) map[types.Object]int {
	counts := map[types.Object]int{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		for _, name := range nilErrWrittenNames(node) {
			counts[info.ObjectOf(name)]++
		}
		return true
	})
	return counts
}

func nilErrWrittenNames(node ast.Node) []*ast.Ident {
	switch node := node.(type) {
	case *ast.AssignStmt:
		return nilErrTargetNames(node.Lhs)
	case *ast.ValueSpec:
		return node.Names
	case *ast.IncDecStmt:
		return nilErrTargetNames([]ast.Expr{node.X})
	default:
		return nil
	}
}

func nilErrTargetNames(targets []ast.Expr) []*ast.Ident {
	var out []*ast.Ident
	for _, target := range targets {
		if name, ok := ast.Unparen(target).(*ast.Ident); ok {
			out = append(out, name)
		}
	}
	return out
}

func (e *nilErrEvidence) stableDefinition(node ast.Node, writes map[types.Object]int) {
	staged := e.clone()
	staged.recordNode(node)
	for _, name := range nilErrWrittenNames(node) {
		object := e.info.Defs[name]
		if object == nil || writes[object] != 1 {
			continue
		}
		e.values[object] = staged.values[object]
	}
}

// Address-taking, receiver aliases, captured writes and deferred execution can
// change the checked value or return slots outside this local path model.
func nilErrUnstable(fn *ast.FuncDecl, checked ast.Expr, info *types.Info) bool {
	root := nilErrLocationOf(checked, info)
	unstable := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		unstable = unstable || nilErrUnstableNode(node, root, info)
		return !unstable
	})
	return unstable
}

func nilErrUnstableNode(node ast.Node, location nilErrLocation, info *types.Info) bool {
	switch node := node.(type) {
	case *ast.DeferStmt, *ast.GoStmt:
		return true
	case *ast.UnaryExpr:
		return node.Op == token.AND && nilErrRootMention(node.X, location.root, info)
	case *ast.AssignStmt:
		return nilErrReceiverAlias(node.Rhs, location, info)
	case *ast.ValueSpec:
		return nilErrReceiverAlias(node.Values, location, info)
	default:
		return false
	}
}

func nilErrReceiverAlias(values []ast.Expr, location nilErrLocation, info *types.Info) bool {
	if len(location.fields) == 0 {
		return false
	}
	for _, value := range values {
		source := nilErrLocationOf(value, info)
		if source.root == location.root && len(source.fields) < len(location.fields) {
			return true
		}
	}
	return false
}

func nilErrRootMention(node ast.Node, root types.Object, info *types.Info) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		if name, ok := node.(*ast.Ident); ok {
			found = found || info.ObjectOf(name) == root
		}
		return !found
	})
	return found
}

func (e *nilErrEvidence) initializeResults(signature *types.Signature, writes map[types.Object]int) {
	for i := 0; i < signature.Results().Len(); i++ {
		result := signature.Results().At(i)
		if result.Name() == "" {
			e.results = append(e.results, result)
			continue
		}
		e.results = append(e.results, result)
		if writes[result] == 0 {
			e.values[result] = nilErrZeroType(result.Type())
		}
	}
}
