package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/types/typeutil"
)

func (a lifecycleAnalysis) workerFact(body *ast.BlockStmt, acquired *ast.CallExpr, binding types.Object) *patterns.ResourceWorker {
	var evidence *patterns.ResourceWorker
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		spawn, ok := node.(*ast.GoStmt)
		if ok && evidence == nil {
			evidence = a.namedWorker(spawn.Call, acquired, binding)
		}
		return evidence == nil
	})
	return evidence
}

func (a lifecycleAnalysis) namedWorker(call, acquired *ast.CallExpr, binding types.Object) *patterns.ResourceWorker {
	function := typeutil.Callee(a.pkg.TypesInfo, call)
	worker, found := a.context.resourceFunctions[function]
	if !found {
		return nil
	}
	offset := methodReceiverOffset(a.pkg.TypesInfo, call)
	if !simpleWorkerCall(worker.object.Type().(*types.Signature), call, offset) {
		return nil
	}
	for index := range call.Args {
		position := pdg.Position(index + offset + 1)
		proven := a.graph.query.InputAcquisition(a.graph.ref(call), position, a.graph.ref(acquired))
		if proven || sameSourceBinding(a.pkg.TypesInfo, call.Args[index], binding) {
			evidence := a.workerEvidence(worker, call, position, offset)
			evidence.HandoffProven = proven
			return evidence
		}
	}
	return nil
}

func simpleWorkerCall(signature *types.Signature, call *ast.CallExpr, offset int) bool {
	if signature.Variadic() || call.Ellipsis.IsValid() {
		return false
	}
	count := signature.Params().Len()
	if signature.Recv() != nil && offset == 0 {
		count++
	}
	return len(call.Args) == count
}

func (a lifecycleAnalysis) workerEvidence(worker resourceFunction, call *ast.CallExpr, position pdg.Position, offset int) *patterns.ResourceWorker {
	query := worker.observation.WithProtocol(a.facts.protocol)
	evidence := &patterns.ResourceWorker{Function: worker.graph.Tables.Text(worker.graph.Function.ID), CleanupProvenOnExit: query.InputCleanup(position, a.context.exits), Shutdown: "unresolved"}
	signature := worker.object.Type().(*types.Signature)
	inputOffset := 0
	if signature.Recv() != nil {
		inputOffset = 1
	}
	resourceIndex := int(position) - inputOffset - 1
	if resourceIndex >= 0 && resourceIndex < signature.Params().Len() {
		evidence.CleanupDeferObserved = a.cleanupDeferObserved(worker.declaration.Body, signature.Params().At(resourceIndex))
	}
	evidence.Shutdown = a.workerShutdown(worker, call, offset, inputOffset)
	return evidence
}

func (a lifecycleAnalysis) workerShutdown(worker resourceFunction, call *ast.CallExpr, offset, inputOffset int) string {
	signature := worker.object.Type().(*types.Signature)
	for index := 0; index < signature.Params().Len(); index++ {
		if !contextType(signature.Params().At(index).Type()) {
			continue
		}
		if receivesCancellation(worker.declaration.Body, a.pkg.TypesInfo, signature.Params().At(index)) {
			return a.contextShutdown(call.Args[index+inputOffset-offset])
		}
	}
	return "unresolved"
}

func methodReceiverOffset(info *types.Info, call *ast.CallExpr) int {
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return 0
	}
	selection := info.Selections[selector]
	if selection != nil && selection.Kind() == types.MethodVal {
		return 1
	}
	return 0
}

func contextType(typ types.Type) bool {
	return types.TypeString(typ, func(pkg *types.Package) string { return pkg.Path() }) == "context.Context"
}

func (a lifecycleAnalysis) contextShutdown(argument ast.Expr) string {
	name := a.graph.query.AcquisitionName(a.graph.ref(argument))
	if name == "context.Background" || name == "context.TODO" {
		return "non_cancellable_context"
	}
	return "cancellation_receive; caller_cancellation_unverified"
}

// This records a compiler-resolved receive/return shape, not a termination
// proof. Partial sequential flow never makes this observation a cleanup proof.
func receivesCancellation(body *ast.BlockStmt, info *types.Info, input types.Object) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		clause, ok := node.(*ast.CommClause)
		if ok && cancellationReturn(clause, info, input) {
			found = true
		}
		return !found
	})
	return found
}

func cancellationReturn(clause *ast.CommClause, info *types.Info, input types.Object) bool {
	statement, ok := clause.Comm.(*ast.ExprStmt)
	if !ok || len(clause.Body) != 1 {
		return false
	}
	receive, ok := ast.Unparen(statement.X).(*ast.UnaryExpr)
	if !ok || receive.Op != token.ARROW {
		return false
	}
	_, ok = clause.Body[0].(*ast.ReturnStmt)
	return ok && typedReceiverCall(info, receive.X, "context.Context.Done", input)
}

// Partial-flow worker observations use compiler bindings only. They never
// discharge ownership or establish that a defer or cancellation path executes.
func sameSourceBinding(info *types.Info, expression ast.Expr, binding types.Object) bool {
	identifier, ok := ast.Unparen(expression).(*ast.Ident)
	return ok && binding != nil && info.ObjectOf(identifier) == binding
}

func typedReceiverCall(info *types.Info, expression ast.Expr, name string, binding types.Object) bool {
	call, ok := ast.Unparen(expression).(*ast.CallExpr)
	if !ok {
		return false
	}
	function, ok := typeutil.Callee(info, call).(*types.Func)
	if !ok || patterns.FuncID(function) != name {
		return false
	}
	selector, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	return ok && sameSourceBinding(info, selector.X, binding)
}

func (a lifecycleAnalysis) cleanupDeferObserved(body *ast.BlockStmt, resource types.Object) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		deferred, ok := node.(*ast.DeferStmt)
		if ok {
			found = a.observedRelease(deferred.Call, resource)
		}
		return !found
	})
	return found
}

func (a lifecycleAnalysis) observedRelease(call *ast.CallExpr, resource types.Object) bool {
	for _, name := range a.facts.protocol.Release {
		if typedReceiverCall(a.pkg.TypesInfo, call, name, resource) {
			return true
		}
	}
	return false
}
