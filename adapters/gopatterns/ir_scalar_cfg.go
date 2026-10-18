package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/token"
	"golang.org/x/tools/go/cfg"
	"strconv"
)

type scalarCFGBuilder struct {
	builder    *irBuilder
	graph      *cfg.CFG
	current    *cfg.Block
	controls   map[ast.Node]pdg.Ref
	maximal    bool
	outcomes   map[int][]pdg.Position
	conditions map[ast.Node]bool
	selections map[ast.Expr]controlSelection
	selectors  map[*ast.CaseClause]int
}

func (a *scalarCFGBuilder) block() *cfg.Block {
	block := &cfg.Block{Index: int32(len(a.graph.Blocks))}
	a.graph.Blocks = append(a.graph.Blocks, block)
	return block
}

func (a *scalarCFGBuilder) source(source ast.Node) {
	a.walk(source)
	if control := a.controls[source]; control != 0 {
		a.operation(control)
	}
}

func (a *scalarCFGBuilder) walk(source ast.Node) {
	var stack []ast.Node
	ast.Inspect(source, func(node ast.Node) bool {
		if a.skipClosure(node) {
			return false
		}
		if node == nil {
			a.finish(stack[len(stack)-1])
			stack = stack[:len(stack)-1]
			return true
		}
		if logical, ok := logicalExpression(node); ok {
			a.logical(logical)
			return false
		}
		stack = append(stack, node)
		for _, read := range a.builder.implicitReads[node] {
			a.operation(read)
		}
		return true
	})
}

func (a *scalarCFGBuilder) finish(source ast.Node) {
	if node := a.builder.nodes[source]; node != 0 {
		a.operation(node)
	}
	if call, ok := source.(*ast.CallExpr); ok && a.maximal && !a.builder.mayReturn(call) {
		a.current = a.block()
	}
}

func (a *scalarCFGBuilder) operation(ref pdg.Ref) {
	// This temporary AST key carries only a PDG lookup. Source identity and spans
	// remain on the semantic operation; no fake source syntax enters the PDG.
	marker := &ast.EmptyStmt{}
	a.builder.nodes[marker] = ref
	a.current.Nodes = append(a.current.Nodes, marker)
}

func (a *scalarCFGBuilder) logical(binary *ast.BinaryExpr) {
	a.walk(binary.X)
	predicate := a.logicalPredicate(binary)
	a.operation(predicate)
	branch := a.current
	rhs, join := a.block(), a.block()
	a.current = branch
	a.transfer(binary.X, logicalSuccessors(binary.Op, rhs, join))
	a.current = rhs
	a.walk(binary.Y)
	a.current.Succs = append(a.current.Succs, join)
	a.current = join
	a.logicalValue(binary)
}

func (a *scalarCFGBuilder) logicalPredicate(binary *ast.BinaryExpr) pdg.Ref {
	b := a.builder
	ref := b.logicalPredicates[binary]
	if ref == 0 {
		ref = b.scalarEntity(b.scalarSourceSpan(binary.X), pdg.Control, pdg.Attributes{Construct: b.pool.Text("short-circuit"), Operator: b.pool.Known(binary.Op.String()), Outcomes: []pdg.Text{b.pool.Text("true"), b.pool.Text("false")}})
		b.logicalPredicates[binary] = ref
		a.predicateValue(binary.X, ref)
	}
	if !b.scalarOperations {
		b.approximateControlNode(ref)
	}
	if a.maximal {
		b.graph.Nodes[ref-1].MatchLabel = b.pool.MatchLabel(pdg.MatchLabel{Kind: b.pool.Text("branch"), Detail: b.pool.Text("short-circuit")})
	}
	return ref
}

func (a *scalarCFGBuilder) logicalValue(binary *ast.BinaryExpr) {
	b := a.builder
	node := &b.graph.Nodes[b.nodes[binary]-1]
	node.Category = pdg.Merge
	node.Synthetic = true
	attrs := &b.attributes[node.Attributes-1]
	attrs.Description = b.pool.Text("Short-circuit boolean value join: skipped RHS uses the left value, evaluated RHS uses the right value.")
}

func (a *scalarCFGBuilder) predicateValue(expression ast.Expr, target pdg.Ref) {
	b := a.builder
	source := b.nodes[expression]
	if source == 0 {
		source = b.nodes[unparen(expression)]
	}
	description := "Value tested by a short-circuit decision; the right operand is evaluated only on its selected outcome."
	b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("logical-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: source, Target: target, Kind: pdg.Data, Position: 1, Subkind: b.pool.Text("scalar-value"), Evidence: b.pool.Evidence(pdg.Supported, description), Provenance: b.pool.Origin(pdg.Analysis, description)})
}

// Expand typed scalar expressions into operation sites. Logical operators split
// normal flow; their result is a nonexecuting value join, never an eager RHS.
func (b *irBuilder) scalarCFG(body *ast.BlockStmt, original *cfg.CFG) *cfg.CFG {
	a := scalarCFGBuilder{builder: b, graph: &cfg.CFG{}, controls: b.executionControls(body)}
	for range original.Blocks {
		a.block()
	}
	for i, block := range original.Blocks {
		a.current = a.graph.Blocks[i]
		for _, source := range block.Nodes {
			a.source(source)
		}
		for _, next := range block.Succs {
			a.current.Succs = append(a.current.Succs, a.graph.Blocks[next.Index])
		}
	}
	return a.graph
}

func logicalExpression(node ast.Node) (*ast.BinaryExpr, bool) {
	binary, ok := node.(*ast.BinaryExpr)
	return binary, ok && (binary.Op == token.LAND || binary.Op == token.LOR)
}

func logicalSuccessors(operator token.Token, rhs, join *cfg.Block) []*cfg.Block {
	if operator == token.LOR {
		return []*cfg.Block{join, rhs}
	}
	return []*cfg.Block{rhs, join}
}

func (a *scalarCFGBuilder) skipClosure(node ast.Node) bool {
	_, closure := node.(*ast.FuncLit)
	return closure && (a.builder.runtimeOperations || a.maximal)
}
