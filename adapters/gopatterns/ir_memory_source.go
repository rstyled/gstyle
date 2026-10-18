package gopatterns

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

// These are resolved source facts, not alias or dependency decisions. The
// points-to solver consumes only the canonical PDG and these typed annotations.
func (b *irBuilder) memorySourceFacts(fn *ast.FuncDecl) {
	for object, ref := range b.symbols {
		b.graph.Symbols[ref-1].Extensions = append(b.graph.Symbols[ref-1].Extensions, b.memoryType(object.Type())...)
	}
	ast.Inspect(fn.Body, b.memorySourceNode)

}

func (b *irBuilder) memoryType(typ types.Type) []pdg.Extension {
	kind := memoryTypeKind(typ)
	if kind == "" {
		return nil
	}
	result := b.pool.SingletonExtension("gstyle.memory:kind", strconv.Quote(kind))
	if kind != "immutable" {
		result = append(result, b.pool.SingletonExtension("gstyle.memory:reference", "true")...)
	}
	return result
}

func (b *irBuilder) memoryExpression(node ast.Node) {
	expr, ok := node.(ast.Expr)
	if !ok || b.info.Types[expr].IsType() {
		return
	}
	ref := b.nodes[node]
	if ref == 0 {
		return
	}
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	attrs.Extensions = append(attrs.Extensions, b.memoryType(b.info.TypeOf(expr))...)
	b.dependencySourceFacts(expr, attrs)
	b.basicSourceFact(expr, attrs)
	if b.writeTargets[expr] {
		b.memoryWriteOperator(expr, attrs)
	}
}

func (b *irBuilder) memoryWriteOperator(expr ast.Expr, attrs *pdg.Attributes) {
	switch expr.(type) {
	case *ast.StarExpr:
		attrs.Operator = b.pool.Known("dereference")
	case *ast.IndexExpr:
		attrs.Operator = b.pool.Known("index")
	case *ast.SelectorExpr:
		attrs.Operator = b.pool.Known("field")
	}
}

func (b *irBuilder) memoryWriteSource(owner ast.Node, targets []ast.Expr) {
	ref := b.nodes[owner]
	if ref == 0 {
		return
	}
	attrs := &b.attributes[b.graph.Nodes[ref-1].Attributes-1]
	if len(attrs.Writes) == 0 {
		return // A type-switch guard is syntax, not a binding write.
	}
	write := 0
	for position, target := range targets {
		if memoryBlankTarget(target) {
			continue
		}
		attrs.Writes[write].Position = pdg.Position(position + 1)
		write++
		b.memoryAddressEdge(owner, target, pdg.Position(position+1))
	}
}

func (b *irBuilder) memoryAddressEdge(owner ast.Node, target ast.Expr, position pdg.Position) {
	if _, direct := unparen(target).(*ast.Ident); direct {
		return
	}
	source := b.nodes[target]
	if source == 0 {
		return
	}
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("address-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: source, Target: b.nodes[owner], Kind: pdg.Data, Subkind: b.pool.Text("write-address"), Position: position, Evidence: b.sourceEvidence, Provenance: b.sourceOrigin})
}

func (b *irBuilder) memorySourceNode(node ast.Node) bool {
	if _, closure := node.(*ast.FuncLit); closure {
		return false
	}
	b.memoryExpression(node)
	switch node := node.(type) {
	case *ast.AssignStmt:
		b.memoryWriteSource(node, node.Lhs)
	case *ast.RangeStmt:
		b.memoryWriteSource(node, []ast.Expr{node.Key, node.Value})
	case *ast.IncDecStmt:
		b.memoryWriteSource(node, []ast.Expr{node.X})
	case *ast.ValueSpec:
		b.memoryWriteSource(node, memoryDeclaredTargets(node.Names))
	}
	return true
}

func memoryTypeKind(typ types.Type) string {
	if typ == nil {
		return ""
	}
	if _, generic := types.Unalias(typ).(*types.TypeParam); generic {
		return "unknown"
	}
	return memoryReferenceKind(typ.Underlying())
}

func memoryReferenceKind(typ types.Type) string {
	switch typ.(type) {
	case *types.Pointer:
		return "pointer"
	case *types.Slice:
		return "slice"
	case *types.Map:
		return "map"
	case *types.Chan:
		return "channel"
	default:
		return memoryValueKind(typ)
	}
}

func memoryValueKind(typ types.Type) string {
	switch typ := typ.(type) {
	case *types.Interface:
		return "interface"
	case *types.Signature:
		return "function"
	case *types.Tuple:
		return "tuple"
	case *types.Struct, *types.Array:
		return "aggregate"
	case *types.Basic:
		return memoryBasicKind(typ)
	default:
		return ""
	}
}

func memoryBasicKind(typ *types.Basic) string {
	switch typ.Kind() {
	case types.UnsafePointer:
		return "pointer"
	case types.String, types.UntypedString:
		return "immutable"
	default:
		return ""
	}
}

func memoryBlankTarget(target ast.Expr) bool {
	if target == nil {
		return true
	}
	id, ok := unparen(target).(*ast.Ident)
	return ok && id.Name == "_"
}

func memoryDeclaredTargets(names []*ast.Ident) []ast.Expr {
	targets := make([]ast.Expr, len(names))
	for index, name := range names {
		targets[index] = name
	}
	return targets
}
