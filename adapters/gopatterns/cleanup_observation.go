package gopatterns

import (
	"go/ast"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (g *releaseGraph) observedCleanupDefers(body *ast.BlockStmt, acquisition *ast.CallExpr) []patterns.DeferredCleanup {
	calls := g.query.ObservedDeferredReleases(g.ref(acquisition))
	if len(calls) == 0 {
		return nil
	}
	matched := make(map[pdg.Ref]bool, len(calls))
	for _, call := range calls {
		matched[call] = true
	}
	var evidence []patterns.DeferredCleanup
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		statement, ok := node.(*ast.DeferStmt)
		if ok && matched[g.ref(statement.Call)] {
			evidence = append(evidence, patterns.DeferredCleanup{Line: g.fset.PositionFor(statement.Pos(), false).Line, Call: expressionText(statement.Call)})
		}
		return true
	})
	return evidence
}
