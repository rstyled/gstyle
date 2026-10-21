package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
	"golang.org/x/tools/go/packages"
)

type lifecycleAnalysis struct {
	pkg      *packages.Package
	graph    *releaseGraph
	facts    protocolFacts
	context  extractCtx
	function string
}

func resourceLifecycleFacts(pkg *packages.Package, fn *ast.FuncDecl, ctx extractCtx, graph *pdg.Graph) []patterns.ResourceLifecycle {
	if len(ctx.protocols) == 0 {
		return nil
	}
	// Package summaries validate and prepare every named function before extraction.
	prepared := ctx.resourceFunctions[pkg.TypesInfo.Defs[fn.Name]].observation
	nodes := releaseSourceNodes(graph)
	var evidence []patterns.ResourceLifecycle
	for _, facts := range ctx.protocols {
		query := releaseGraph{graph: graph, query: prepared.WithProtocol(facts.protocol), nodes: nodes, fset: pkg.Fset}
		analysis := lifecycleAnalysis{pkg: pkg, graph: &query, facts: facts, context: ctx, function: graph.Tables.Text(graph.Function.ID)}
		evidence = append(evidence, analysis.functionFacts(fn)...)
	}
	return evidence
}

func (a lifecycleAnalysis) functionFacts(fn *ast.FuncDecl) []patterns.ResourceLifecycle {
	var evidence []patterns.ResourceLifecycle
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		evidence = append(evidence, a.returnCarrierFacts(node)...)
		if fact := a.nodeFact(node, fn.Body); fact != nil {
			if fact.State == patterns.ResourceReturned && !a.returnsResource(fn) {
				fact.State = patterns.ResourceReturnedOwner
				fact.Reason = "resource is carried by the returned object; the object's cleanup lifecycle remains unverified"
			}
			if fact.State == patterns.ResourceUnresolved {
				a.attachWorker(fn.Body, node, fact)
			}
			evidence = append(evidence, *fact)
		}
		return true
	})
	return evidence
}

// Inline composite acquisitions have no local binding to follow. Record the
// typed source shape as an ownership warning; never discharge it from syntax.
func (a lifecycleAnalysis) returnCarrierFacts(node ast.Node) []patterns.ResourceLifecycle {
	statement, ok := node.(*ast.ReturnStmt)
	if !ok || len(statement.Results) == 0 {
		return nil
	}
	var evidence []patterns.ResourceLifecycle
	for _, call := range inlineCarrierCalls(statement.Results[0]) {
		if !a.graph.query.Observed(a.graph.ref(call)) || !a.acquires(call) {
			continue
		}
		fact := a.newFact(call)
		fact.State = patterns.ResourceReturnedOwner
		fact.Reason = "resource acquisition is observed in a returned composite object; the object's cleanup lifecycle remains unverified"
		evidence = append(evidence, fact)
	}
	return evidence
}

func inlineCarrierCalls(expression ast.Expr) []*ast.CallExpr {
	switch expression := ast.Unparen(expression).(type) {
	case *ast.UnaryExpr:
		if expression.Op == token.AND {
			return inlineCarrierCalls(expression.X)
		}
	case *ast.CompositeLit:
		var calls []*ast.CallExpr
		for _, element := range expression.Elts {
			calls = append(calls, inlineElementCalls(element)...)
		}
		return calls
	}
	return nil
}

func inlineElementCalls(expression ast.Expr) []*ast.CallExpr {
	if field, ok := expression.(*ast.KeyValueExpr); ok {
		expression = field.Value
	}
	if call, ok := ast.Unparen(expression).(*ast.CallExpr); ok {
		return []*ast.CallExpr{call}
	}
	return inlineCarrierCalls(expression)
}

func (a lifecycleAnalysis) returnsResource(fn *ast.FuncDecl) bool {
	signature := a.pkg.TypesInfo.Defs[fn.Name].Type().(*types.Signature)
	if signature.Results().Len() == 0 {
		return false
	}
	typ := types.Unalias(signature.Results().At(0).Type())
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	named, ok := typ.(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	identity := named.Obj().Pkg().Path() + "." + named.Obj().Name()
	return slices.Contains(a.facts.protocol.Release, identity+"."+a.newReleaseName())
}

func (a lifecycleAnalysis) newReleaseName() string {
	release := a.facts.protocol.Release[0]
	return release[strings.LastIndex(release, ".")+1:]
}

func (a lifecycleAnalysis) attachWorker(body *ast.BlockStmt, node ast.Node, fact *patterns.ResourceLifecycle) {
	call := sourceAcquisition(a.pkg.TypesInfo, node)
	if call == nil {
		return
	}
	targets, _ := acquisitionSyntax(node)
	binding := a.pkg.TypesInfo.ObjectOf(ast.Unparen(targets[0]).(*ast.Ident))
	fact.Worker = a.workerFact(body, call, binding)
	if fact.Worker != nil {
		fact.State = patterns.ResourceInWorker
		fact.Reason = "worker handoff is observed; resource value flow, cleanup execution and caller-controlled shutdown require separate verification"
	}
}

func (a lifecycleAnalysis) nodeFact(node ast.Node, body *ast.BlockStmt) *patterns.ResourceLifecycle {
	ref := a.graph.ref(node)
	if !a.graph.query.Observed(ref) {
		return nil
	}
	if returned, ok := node.(*ast.ReturnStmt); ok {
		return a.returnFact(returned, ref)
	}
	call := sourceAcquisition(a.pkg.TypesInfo, node)
	if call == nil || !a.acquires(call) {
		return nil
	}
	fact := a.newFact(call)
	policy := a.policy(call, ref)
	fact.State = a.state(call, ref, policy)
	fact.Reason = lifecycleReason(fact.State)
	if fact.State == patterns.ResourceUnresolved {
		fact.DeferredCleanup = a.graph.observedCleanupDefers(body, call)
	}
	return &fact
}

func (a lifecycleAnalysis) acquires(call *ast.CallExpr) bool {
	return a.graph.query.Acquires(a.graph.ref(call), releaseHelperIdentities(a.facts.sources))
}

func (a lifecycleAnalysis) newFact(call *ast.CallExpr) patterns.ResourceLifecycle {
	name := a.graph.query.AcquisitionName(a.graph.ref(call))
	return patterns.ResourceLifecycle{
		ResourceAcquisition: patterns.ResourceAcquisition{FuncID: a.function, Line: a.pkg.Fset.PositionFor(call.Pos(), false).Line, Resource: a.facts.protocol.Resource, AcquireCall: name},
		SourceNode:          uint32(a.graph.ref(call)),
		State:               patterns.ResourceUnresolved, Release: a.newReleaseName(),
		Reason: "cleanup or ownership transfer is not established on all modeled paths; review the owner's lifetime",
	}
}

func (a lifecycleAnalysis) returnFact(statement *ast.ReturnStmt, ref pdg.Ref) *patterns.ResourceLifecycle {
	if len(statement.Results) == 0 {
		return nil
	}
	call, ok := ast.Unparen(statement.Results[0]).(*ast.CallExpr)
	if !ok || !a.acquires(call) || a.graph.query.Operand(ref, 1) != a.graph.ref(call) {
		return nil
	}
	fact := a.newFact(call)
	if a.graph.query.HasFlow() {
		fact.State = patterns.ResourceReturned
		fact.Reason = lifecycleReason(fact.State)
	}
	return &fact
}

func (a lifecycleAnalysis) policy(call *ast.CallExpr, owner pdg.Ref) resourceflow.Policy {
	policy := resourceflow.Policy{Resource: a.graph.writtenSymbol(owner, 1), Helpers: releaseHelperIdentities(a.facts.helpers), Exits: a.context.exits}
	if a.graph.query.FailureEmpty(a.graph.ref(call), releaseHelperIdentities(a.facts.failures)) {
		policy.AcquisitionError = a.graph.writtenSymbol(owner, 2)
	}
	return policy
}

func (a lifecycleAnalysis) state(call *ast.CallExpr, owner pdg.Ref, policy resourceflow.Policy) patterns.ResourceState {
	query, acquired := a.graph.query, a.graph.ref(call)
	if !query.HasFlow() {
		return patterns.ResourceUnresolved
	}
	if query.AfterAcquisition(acquired, owner, policy) {
		return patterns.ResourceReleased
	}
	policy.TransferReturn = true
	if query.AfterAcquisition(acquired, owner, policy) {
		return patterns.ResourceReturned
	}
	return patterns.ResourceUnresolved
}

func lifecycleReason(state patterns.ResourceState) string {
	switch state {
	case patterns.ResourceReleased:
		return "cleanup is recognized on modeled paths; external method behavior remains unproved"
	case patterns.ResourceReturned:
		return "resource is returned to the caller; caller cleanup remains a separate obligation"
	default:
		return "cleanup or ownership transfer is not established on all modeled paths; review the owner's lifetime"
	}
}
