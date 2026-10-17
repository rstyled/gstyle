package gopatterns

import (
	"go/ast"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/cfg"
)

// A range writes its bindings only after the iterator produces an element.
// The false outcome preserves the previous values, including zero iterations.
func (a *scalarCFGBuilder) rangeSuccessors(source *cfg.Block, successors []*cfg.Block) []*cfg.Block {
	if source.Kind != cfg.KindRangeLoop || len(successors) != 2 {
		return successors
	}
	binding := a.builder.rangeBinding(source.Stmt.(*ast.RangeStmt))
	if binding == 0 {
		return successors
	}
	decision := a.current
	a.current = a.block()
	statement := source.Stmt.(*ast.RangeStmt)
	if statement.Key != nil {
		a.walk(statement.Key)
	}
	if statement.Value != nil {
		a.walk(statement.Value)
	}
	a.operation(binding)
	a.current.Succs = []*cfg.Block{successors[0]}
	successors[0] = a.current
	a.current = decision
	return successors
}

func (b *irBuilder) rangeBinding(statement *ast.RangeStmt) pdg.Ref {
	ref := b.nodes[statement]
	node := b.graph.Nodes[ref-1]
	attrs := &b.attributes[node.Attributes-1]
	if len(attrs.Writes) == 0 {
		return 0
	}
	bindingAttrs := pdg.Attributes{Construct: b.pool.Text("range-binding"), WriteForm: b.pool.Text(statement.Tok.String()), Symbols: attrs.Symbols, Writes: attrs.Writes, Defines: attrs.Defines, Extensions: attrs.Extensions}
	attrs.Symbols, attrs.Writes, attrs.Defines = nil, nil, nil
	binding := b.scalarEntity(node.Span, pdg.Assignment, bindingAttrs)
	b.graph.Nodes[binding-1].Line = node.Line
	for _, definition := range bindingAttrs.Defines {
		b.graph.Definitions[definition-1].Node = binding
	}
	b.rangeAddresses(ref, binding)
	b.rangeValues(b.rangeSource(statement), binding, bindingAttrs.Writes)
	return binding
}

func (b *irBuilder) rangeAddresses(source, target pdg.Ref) {
	for i, edge := range b.graph.Edges {
		if edge.Target == source && b.pool.Tables.Text(edge.Subkind) == "write-address" {
			b.graph.Edges[i].Target = target
		}
	}
}

func (b *irBuilder) rangeValues(source, target pdg.Ref, writes []pdg.Access) {
	if source == 0 {
		return
	}
	for _, write := range writes {
		b.graph.Edges = append(b.graph.Edges, pdg.Edge{ID: b.pool.Text("range-e" + strconv.Itoa(len(b.graph.Edges)+1)), Source: source, Target: target, Kind: pdg.Data, Position: write.Position, Subkind: b.pool.Text("iteration-value"), Evidence: b.pool.Evidence(pdg.Approximated, "Iteration values depend on the ranged collection or bound; element and key precision is separate."), Provenance: b.sourceOrigin})
	}
}

// x/tools places range assignment targets in the preheader. Go evaluates them
// on each successful iteration, so operation lowering relocates those sites.
func controlRangeTargets(body *ast.BlockStmt) map[ast.Node]bool {
	targets := map[ast.Node]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		if _, closure := node.(*ast.FuncLit); closure {
			return false
		}
		if statement, ok := node.(*ast.RangeStmt); ok {
			targets[statement.Key] = true
			targets[statement.Value] = true
		}
		return true
	})
	return targets
}
