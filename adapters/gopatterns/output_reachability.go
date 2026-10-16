package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/patterns/outputcheck"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

type outputSpan struct{ start, end uint32 }

func reachableErrorCheckFacts(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info, graph *pdg.Graph) ([]patterns.ErrorCheckSite, []int, error) {
	sites, success := findErrorCheckFacts(fn, fset, info)
	if len(success) == 0 {
		return sites, nil, nil
	}
	nodes, err := runtimeguard.ReachableNodes(graph)
	if err != nil {
		return nil, nil, err
	}
	reachable := make(map[outputSpan]bool)
	for _, ref := range nodes {
		node := graph.Nodes[ref-1]
		if node.Category != pdg.Call || node.Span == 0 {
			continue
		}
		span := graph.Spans[node.Span-1]
		reachable[outputSpan{span.Start, span.End}] = true
	}
	sites, success = scanErrorCheckFacts(fn, fset, info, reachable)
	return sites, success, nil
}

func runtimeOutputCall(call *ast.CallExpr, info *types.Info) bool {
	callee := outputCalleeIdentity(call, info)
	return outputcheck.Formatted(callee.Package, callee.Name)
}
