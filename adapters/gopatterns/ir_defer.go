package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"slices"
	"strconv"

	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

type deferredInput struct {
	expression ast.Expr
	slot       deferflow.Slot
}

func (b *irBuilder) deferredSourceFacts(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if statement, ok := node.(*ast.DeferStmt); ok {
			b.deferredSite(statement)
		}
		return true
	})
}

func (b *irBuilder) deferredInvocation(node ast.Node) bool {
	call, ok := node.(*ast.CallExpr)
	return ok && b.deferredCalls[call] != 0
}

func (b *irBuilder) deferredSite(statement *ast.DeferStmt) {
	registration, invocation := b.nodes[statement], b.nodes[statement.Call]
	b.deferredCalls[statement.Call] = registration
	slots := b.publishDeferredInputs(statement, b.deferredInputs(statement.Call))
	fact := deferflow.Fact{Invocation: b.pool.Tables.Text(b.graph.Nodes[invocation-1].ID), Slots: slots, Spread: statement.Call.Ellipsis.IsValid(), Tuple: b.deferredTuple(statement.Call)}
	data, _ := json.Marshal(fact) // Only strings, booleans and numeric slices.
	b.addNode(statement, pdg.Other, b.deferredAttributes(b.pool.SingletonExtension(deferflow.Extension, string(data))))
	b.graph.Edges = slices.DeleteFunc(b.graph.Edges, func(edge pdg.Edge) bool {
		return edge.Source == registration && edge.Target == invocation && b.pool.Tables.Text(edge.Subkind) == "mining-value"
	})
	b.deferredEdge(registration, invocation, "registration", 0)
}

// Plan captures without changing the graph. Publication preserves this order.
func (b *irBuilder) deferredInputs(call *ast.CallExpr) []deferredInput {
	inputs := []deferredInput{b.deferredTarget(call), b.deferredReceiver(call)}
	offset := 0
	if inputs[1].expression != nil {
		offset = 1
	}
	for i, argument := range call.Args {
		inputs = append(inputs, deferredInput{argument, deferflow.Slot{Kind: "argument", Position: pdg.Position(i + offset + 1), Mode: "value"}})
	}
	return inputs
}

func (b *irBuilder) deferredTarget(call *ast.CallExpr) deferredInput {
	attrs := b.attributes[b.graph.Nodes[b.nodes[call]-1].Attributes-1]
	if attrs.CallForm == pdg.DirectCall && attrs.Callee.Status == pdg.Known {
		return deferredInput{}
	}
	return deferredInput{call.Fun, deferflow.Slot{Kind: "target", Mode: "value"}}
}

func (b *irBuilder) deferredReceiver(call *ast.CallExpr) deferredInput {
	if !b.implicitReceiver(call.Fun) {
		return deferredInput{}
	}
	selector := unparen(call.Fun).(*ast.SelectorExpr)
	return deferredInput{selector.X, deferflow.Slot{Kind: "receiver", Position: 1, Mode: b.deferredReceiverMode(selector)}}
}

func (b *irBuilder) publishDeferredInputs(statement *ast.DeferStmt, inputs []deferredInput) []deferflow.Slot {
	slots := []deferflow.Slot{}
	for _, input := range inputs {
		if input.expression == nil {
			continue
		}
		b.deferredCapture(statement, input.expression, input.slot)
		slots = append(slots, input.slot)
	}
	return slots
}

func (b *irBuilder) deferredAttributes(extensions []pdg.Extension) pdg.Attributes {
	return pdg.Attributes{Construct: b.pool.Text("defer-register"), Description: b.pool.Text("Evaluate and save a deferred call; invocation occurs during function exit."), Extensions: extensions}
}

func (b *irBuilder) deferredCapture(statement *ast.DeferStmt, expression ast.Expr, slot deferflow.Slot) {
	ref := b.nodes[expression]
	if ref == 0 {
		ref = b.addNode(expression, pdg.Computation, pdg.Attributes{Description: b.pool.Text("Evaluated input saved by a deferred registration; its value is unknown.")})
		b.runtimeExpression(expression)
		b.memoryExpression(expression)
	}
	b.deferredEdge(ref, b.nodes[statement], slot.Kind, slot.Position)
}

func (b *irBuilder) deferredEdge(source, target pdg.Ref, kind string, position pdg.Position) {
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("defer-" + strconv.Itoa(int(source)) + "-" + strconv.Itoa(int(target)) + "-" + kind + "-" + strconv.Itoa(int(position))), Source: source, Target: target, Kind: pdg.Data, Subkind: b.pool.Text("defer-" + kind), Position: position, Evidence: b.pool.Evidence(pdg.Supported, "Deferred registration saves evaluated inputs; invocation consumes the saved registration during exit."), Provenance: b.sourceOrigin})
}

func (b *irBuilder) deferredTuple(call *ast.CallExpr) bool {
	if len(call.Args) != 1 {
		return false
	}
	_, tuple := b.info.TypeOf(call.Args[0]).(*types.Tuple)
	return tuple
}

func (b *irBuilder) deferredReceiverMode(selector *ast.SelectorExpr) string {
	selection := b.info.Selections[selector]
	if len(selection.Index()) != 1 {
		return "unknown"
	}
	function, ok := selection.Obj().(*types.Func)
	if !ok {
		return "unknown"
	}
	signature := function.Type().(*types.Signature)
	declared, actual := pointerType(signature.Recv().Type()), pointerType(b.info.TypeOf(selector.X))
	if declared == actual {
		return "value"
	}
	if declared {
		return "address"
	}
	return "dereference"
}
