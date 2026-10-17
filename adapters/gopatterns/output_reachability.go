package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/outputflow"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/patterns/outputcheck"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

type outputSpan struct{ start, end uint32 }

type outputCompletionProof struct {
	flow  *outputflow.SuccessFlow
	calls map[outputSpan]runtimeguard.Call
	fset  *token.FileSet
}

func (p outputCompletionProof) accept(call *ast.CallExpr) bool {
	span := outputSpan{uint32(p.fset.PositionFor(call.Pos(), false).Offset), uint32(p.fset.PositionFor(call.End(), false).Offset)}
	fact, found := p.calls[span]
	return found && p.flow.Output(fact.Node, fact.FirstArgument.Nil)
}

func reachableErrorCheckFacts(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info, graph *pdg.Graph) ([]patterns.ErrorCheckSite, []int, error) {
	checks, err := newErrorUseFacts(graph, fset)
	if err != nil {
		return nil, nil, err
	}
	walker := errorCheckWalker{fset: fset, info: info, checks: checks}
	sites, _ := walker.scan(fn)
	if !hasDiscardedOutput(sites) {
		return sites, nil, nil
	}
	proof, err := outputCompletion(fn, fset, info, graph)
	if err != nil {
		return nil, nil, err
	}
	walker.successOutput = proof
	sites, success := walker.scan(fn)
	return sites, success, nil
}

func outputCompletion(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info, graph *pdg.Graph) (func(*ast.CallExpr) bool, error) {
	slot := outputErrorResult(fn, info)
	if slot < 0 {
		return nil, nil
	}
	proof, err := outputflow.NewSuccessFlow(graph, pdg.Position(slot+1))
	if err != nil || proof == nil {
		return nil, err
	}
	report, err := runtimeguard.Analyze(graph)
	if err != nil {
		return nil, err
	}
	calls := outputCalls(graph, report)
	completion := outputCompletionProof{flow: proof, calls: calls, fset: fset}
	return completion.accept, nil
}

func outputCalls(graph *pdg.Graph, report runtimeguard.Report) map[outputSpan]runtimeguard.Call {
	calls := make(map[outputSpan]runtimeguard.Call)
	for _, call := range report.Calls {
		node := graph.Nodes[call.Node-1]
		if node.Category != pdg.Call || node.Span == 0 {
			continue
		}
		span := graph.Spans[node.Span-1]
		calls[outputSpan{span.Start, span.End}] = call
	}
	return calls
}

func runtimeOutputCall(call *ast.CallExpr, info *types.Info) bool {
	callee := outputCalleeIdentity(call, info)
	return outputcheck.Formatted(callee.Package, callee.Name)
}

func hasDiscardedOutput(sites []patterns.ErrorCheckSite) bool {
	for _, site := range sites {
		if site.DiscardedOutput {
			return true
		}
	}
	return false
}
