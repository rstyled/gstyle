package gopatterns

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/callflow"
	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"golang.org/x/tools/go/packages"
)

type fileReleaseAnalysis struct {
	exits             callflow.ExitSummaries
	sources, failures map[string]bool
	pkg               *packages.Package
	fn                *ast.FuncDecl
	graph             *releaseGraph
	helpers           map[string]bool
}

func (a fileReleaseAnalysis) facts() (released []int, sites []patterns.ResourceAcquisition) {
	ast.Inspect(a.fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		line, site := a.fileNodeFacts(node)
		if line != 0 {
			released = append(released, line)
		}
		if site != nil {
			sites = append(sites, *site)
		}
		return true
	})
	return released, sites
}

func (a fileReleaseAnalysis) fileNodeFacts(node ast.Node) (int, *patterns.ResourceAcquisition) {
	ref := a.graph.ref(node)
	if !a.graph.query.Reachable(ref) {
		return 0, nil
	}
	if _, returned := node.(*ast.ReturnStmt); returned {
		return a.returnedFileLine(node, ref), nil
	}
	return a.acquisitionFacts(node, ref)
}

func (a fileReleaseAnalysis) acquisitionFacts(node ast.Node, ref pdg.Ref) (int, *patterns.ResourceAcquisition) {
	call := sourceAcquisition(a.pkg.TypesInfo, node)
	if call == nil || !a.graph.query.Acquires(a.graph.ref(call), a.sources) {
		return 0, nil
	}
	if a.graph.query.Operand(ref, 1) != a.graph.ref(call) {
		return 0, nil
	}
	return a.acquiredFacts(node, ref, call)
}

func (a fileReleaseAnalysis) acquiredFacts(node ast.Node, ref pdg.Ref, call *ast.CallExpr) (int, *patterns.ResourceAcquisition) {
	policy := resourceflow.Policy{Resource: a.graph.writtenSymbol(ref, 1), AcquisitionError: a.acquisitionError(ref, call), TransferReturn: true, Helpers: a.helpers, Exits: a.exits}
	if a.graph.query.AfterAcquisition(a.graph.ref(call), ref, policy) {
		return a.pkg.Fset.PositionFor(node.Pos(), false).Line, nil
	}
	if a.graph.query.OSFileAcquisition(a.graph.ref(call)) {
		return 0, nil
	}
	policy.TransferCalls = true
	if a.graph.query.AfterAcquisition(a.graph.ref(call), ref, policy) {
		return 0, nil
	}
	return 0, &patterns.ResourceAcquisition{Line: a.pkg.Fset.PositionFor(call.Pos(), false).Line, Resource: "file", AcquireCall: a.graph.query.AcquisitionName(a.graph.ref(call))}
}

// Typed syntax identifies a direct file-return site. Its actual producer must
// agree with the canonical return operand; syntax cannot replace a missing edge.
func (a fileReleaseAnalysis) returnedFileLine(node ast.Node, ref pdg.Ref) int {
	statement, ok := node.(*ast.ReturnStmt)
	if !ok || len(statement.Results) != 1 {
		return 0
	}
	call, ok := ast.Unparen(statement.Results[0]).(*ast.CallExpr)
	if !ok || !a.graph.query.Acquires(a.graph.ref(call), a.sources) {
		return 0
	}
	if a.graph.query.Operand(ref, 1) != a.graph.ref(call) {
		return 0
	}
	return a.pkg.Fset.PositionFor(call.Pos(), false).Line
}

func (a fileReleaseAnalysis) acquisitionError(ref pdg.Ref, call *ast.CallExpr) pdg.Ref {
	if !a.graph.query.FailureEmpty(a.graph.ref(call), a.failures) {
		return 0
	}
	return a.graph.writtenSymbol(ref, 2)
}

func fileObligationFacts(pkg *packages.Package, fn *ast.FuncDecl, ctx extractCtx, graph *pdg.Graph) ([]int, []patterns.ResourceAcquisition, error) {
	released, sites, _, err := resourceObligationFacts(pkg, fn, ctx, graph)
	return released, sites, err
}

func resourceObligationFacts(pkg *packages.Package, fn *ast.FuncDecl, ctx extractCtx, graph *pdg.Graph) ([]int, []patterns.ResourceAcquisition, []int, error) {
	facts, err := newReleaseGraph(graph, pkg.Fset)
	if err != nil {
		return nil, nil, nil, err
	}
	analysis := fileReleaseAnalysis{exits: ctx.exits, sources: releaseHelperIdentities(ctx.fileSources), failures: releaseHelperIdentities(ctx.fileFailures), pkg: pkg, fn: fn, graph: facts, helpers: releaseHelperIdentities(ctx.helpers)}
	released, sites := analysis.facts()
	return released, sites, recognizedDeferredLines(graph, facts.query, ctx.exits), nil
}

func releaseHelperIdentities(helpers map[types.Object]bool) map[string]bool {
	identities := make(map[string]bool, len(helpers))
	for object, releases := range helpers {
		if function, ok := object.(*types.Func); ok && releases {
			identities[patterns.FuncID(function)] = true
		}
	}
	return identities
}

func recognizedDeferredLines(graph *pdg.Graph, query *resourceflow.Query, exits callflow.ExitSummaries) []int {
	var lines []int
	for i, node := range graph.Nodes {
		if node.Category != pdg.Call {
			continue
		}
		attrs := graph.NodeAttributes(node)
		method, field, receiver := patterns.AcquisitionRelease(graph.Tables.Text(attrs.Callee.Value))
		rule := resourceflow.CleanupRule{Method: method, Field: field, Receiver: receiver}
		if method != "" && query.RecognizesDeferredCleanup(pdg.Ref(i+1), rule, exits) {
			lines = append(lines, int(node.Line))
		}
	}
	return lines
}
