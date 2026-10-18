package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

// These facts describe one operation, not the evaluation of its operands or
// the completion of a path. Calls are summarized from canonical targets.
type operationEffects struct {
	Panic    bool `json:"panic,omitempty"`
	NilInput bool `json:"nil_input,omitempty"`
	Block    bool `json:"block,omitempty"`
	Unknown  bool `json:"unknown,omitempty"`
}

func (b *irBuilder) operationEffectFacts(body *ast.BlockStmt) {
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil {
			return true
		}
		b.operationEffectNode(node)
		_, closure := node.(*ast.FuncLit)
		return !closure
	})
}

func (b *irBuilder) operationEffectNode(source ast.Node) {
	ref := b.nodes[source]
	if ref == 0 || b.graph.Nodes[ref-1].Category == pdg.Call {
		return
	}
	effects := b.sourceOperationEffects(source)
	data, _ := json.Marshal(effects)
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:operation-effects", string(data))...)
	b.bindingInputFacts(source, attrs)
}

func (b *irBuilder) sourceOperationEffects(source ast.Node) operationEffects {
	if expression, ok := source.(ast.Expr); ok {
		return b.expressionOperationEffects(expression)
	}
	switch source := source.(type) {
	case *ast.AssignStmt:
		return operationEffects{Panic: storeCanPanic(source.Lhs) || compoundCanPanic(source.Tok)}
	case *ast.SendStmt:
		return operationEffects{Panic: true, Block: true}
	case *ast.RangeStmt:
		return rangeOperationEffects(b.info.TypeOf(source.X))
	default:
		return statementOperationEffects(source)
	}
}

func (b *irBuilder) expressionOperationEffects(expression ast.Expr) operationEffects {
	if b.unsafeOperation(expression) {
		return operationEffects{Unknown: true}
	}
	if unary, ok := expression.(*ast.UnaryExpr); ok && unary.Op == token.ARROW {
		return operationEffects{Block: true}
	}
	if b.info.Types[expression].IsType() {
		return operationEffects{}
	}
	if normalExpressionOperation(expression, b.info) {
		return operationEffects{}
	}
	return b.typedExpressionEffects(expression)
}

func (b *irBuilder) unsafeOperation(expression ast.Expr) bool {
	if unsafeOperationType(b.info.TypeOf(expression)) {
		return true
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok {
		return false
	}
	for _, argument := range call.Args {
		if unsafeOperationType(b.info.TypeOf(argument)) {
			return true
		}
	}
	return false
}

func (b *irBuilder) typedExpressionEffects(expression ast.Expr) operationEffects {
	switch expression := expression.(type) {
	case *ast.StarExpr:
		return operationEffects{NilInput: true}
	case *ast.SelectorExpr:
		return b.selectorOperationEffects(expression)
	case *ast.CompositeLit:
		return compositeOperationEffects(b.info.TypeOf(expression))
	case *ast.KeyValueExpr:
		return operationEffects{}
	default:
		return finiteExpressionEffects(expression)
	}
}

func (b *irBuilder) selectorOperationEffects(expression *ast.SelectorExpr) operationEffects {
	selection := b.info.Selections[expression]
	if selection == nil || selection.Kind() == types.MethodExpr {
		return operationEffects{}
	}
	if len(selection.Index()) > 1 {
		return operationEffects{Panic: true}
	}
	if selection.Kind() == types.FieldVal {
		return operationEffects{NilInput: pointerOperationType(selection.Recv())}
	}
	return boundMethodOperationEffects(selection)
}

// Binding statements inventory RHS values separately from target address
// evaluation. Canonical edges still identify every actual producer.
func (b *irBuilder) bindingInputFacts(source ast.Node, attrs *pdg.Attributes) {
	values, known := bindingInputValues(source)
	if !known {
		return
	}
	positions := b.valuePositions(values, 0)
	if positions == nil {
		positions = []pdg.Position{}
	}
	data, _ := json.Marshal(positions)
	attrs.Extensions = append(attrs.Extensions, b.pool.SingletonExtension("gstyle.go:value-operands", string(data))...)
}

func finiteExpressionEffects(expression ast.Expr) operationEffects {
	switch expression.(type) {
	case *ast.FuncLit, *ast.Ident, *ast.BasicLit, *ast.ParenExpr:
		return operationEffects{}
	case *ast.CallExpr, *ast.BinaryExpr, *ast.UnaryExpr, *ast.StarExpr, *ast.SelectorExpr, *ast.IndexExpr, *ast.IndexListExpr, *ast.SliceExpr, *ast.TypeAssertExpr, *ast.CompositeLit, *ast.KeyValueExpr:
		return operationEffects{Panic: true}
	default:
		return operationEffects{Unknown: true}
	}
}

func unsafeOperationType(typ types.Type) bool {
	if typ == nil {
		return false
	}
	basic, ok := typ.Underlying().(*types.Basic)
	return ok && basic.Kind() == types.UnsafePointer
}

func storeCanPanic(targets []ast.Expr) bool {
	for _, target := range targets {
		if _, direct := unparen(target).(*ast.Ident); !direct {
			return true
		}
	}
	return false
}

func compoundCanPanic(operator token.Token) bool {
	switch operator {
	case token.QUO_ASSIGN, token.REM_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN:
		return true
	default:
		return false
	}
}

func writeOperationEffects(source ast.Node) operationEffects {
	if increment, ok := source.(*ast.IncDecStmt); ok {
		return operationEffects{Panic: storeCanPanic([]ast.Expr{increment.X})}
	}
	return operationEffects{}
}

func statementOperationEffects(source ast.Node) operationEffects {
	switch source.(type) {
	case *ast.ValueSpec, *ast.IncDecStmt:
		return writeOperationEffects(source)
	case *ast.ReturnStmt, *ast.DeferStmt, *ast.BranchStmt, *ast.IfStmt, *ast.ForStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt:
		return operationEffects{}
	default:
		return operationEffects{Unknown: true}
	}
}

func rangeOperationEffects(typ types.Type) operationEffects {
	if typ == nil {
		return operationEffects{Unknown: true}
	}
	switch typ.Underlying().(type) {
	case *types.Chan:
		return operationEffects{Block: true}
	case *types.Signature, *types.Interface:
		return operationEffects{Unknown: true}
	default:
		return operationEffects{}
	}
}

func boundMethodOperationEffects(selection *types.Selection) operationEffects {
	if _, isInterface := selection.Recv().Underlying().(*types.Interface); isInterface {
		return operationEffects{NilInput: true}
	}
	signature, ok := selection.Obj().Type().(*types.Signature)
	if !ok {
		return operationEffects{Unknown: true}
	}
	return operationEffects{NilInput: pointerOperationType(selection.Recv()) && !pointerOperationType(signature.Recv().Type())}
}

func pointerOperationType(typ types.Type) bool {
	_, pointer := typ.Underlying().(*types.Pointer)
	return pointer
}

func bindingInputValues(source ast.Node) ([]ast.Expr, bool) {
	switch source := source.(type) {
	case *ast.AssignStmt:
		if source.Tok != token.ASSIGN && source.Tok != token.DEFINE {
			return nil, false
		}
		return source.Rhs, true
	case *ast.ValueSpec:
		return source.Values, true
	default:
		return nil, false
	}
}

func compositeOperationEffects(typ types.Type) operationEffects {
	if typ == nil {
		return operationEffects{Unknown: true}
	}
	switch typ := typ.Underlying().(type) {
	case *types.Struct, *types.Array, *types.Slice:
		return operationEffects{}
	case *types.Map:
		return operationEffects{Panic: !strictComparable(typ.Key())}
	default:
		return operationEffects{Unknown: true}
	}
}
