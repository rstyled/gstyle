package gopatterns

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"slices"
	"strconv"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Closure construction retains outer cells, not the execution of its body.
// Compiler object identity distinguishes captures from same-named inner locals.
func (b *irBuilder) closureSourceFacts(fn *ast.FuncDecl) {
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		literal, ok := node.(*ast.FuncLit)
		if !ok {
			return true
		}
		b.closureValue(literal)
		return false
	})
}

func (b *irBuilder) closureValue(literal *ast.FuncLit) {
	captures, globals := b.closureReferences(literal)
	attrs := pdg.Attributes{Construct: b.pool.Text("closure"), Operator: b.pool.Known("closure"), Symbols: captures, Description: b.pool.Text("Closure value captures variable cells; body execution belongs to a separate canonical callable graph."), Extensions: b.closureExtensions(literal, globals)}
	b.addNode(literal, pdg.Other, attrs)
	b.closureValues = true
}

func (b *irBuilder) closureExtensions(literal *ast.FuncLit, globals []pdg.Ref) []pdg.Extension {
	bodyID := b.pool.Tables.Text(b.graph.Function.ID) + ".closure@" + strconv.Itoa(b.fset.PositionFor(literal.Pos(), false).Offset)
	extensions := append(b.memoryType(b.info.TypeOf(literal)), []pdg.Extension{
		{Name: b.pool.Text("gstyle.runtime:non-nil"), Value: b.pool.Text("true")},
		{Name: b.pool.Text("gstyle.go:value-fact"), Value: b.pool.Text(`{"non_nil":true}`)},
		{Name: b.pool.Text("gstyle.runtime:normal-completion"), Value: b.pool.Text("true")},
		{Name: b.pool.Text("gstyle.closure:body-id"), Value: b.pool.Text(strconv.Quote(bodyID))},
		{Name: b.pool.Text("gstyle.go:value-operands"), Value: b.pool.Text(`[]`)},
	}...)
	return append(extensions, b.closureGlobalFacts(globals)...)
}

func (b *irBuilder) closureGlobalFacts(globals []pdg.Ref) []pdg.Extension {
	if len(globals) == 0 {
		return nil
	}
	ids := make([]string, len(globals))
	for i, ref := range globals {
		ids[i] = b.pool.Tables.Text(b.graph.Symbols[ref-1].ID)
	}
	data, _ := json.Marshal(ids)
	return b.pool.SingletonExtension("gstyle.closure:globals", string(data))
}

func (b *irBuilder) closureReferences(literal *ast.FuncLit) ([]pdg.Ref, []pdg.Ref) {
	captures, globals := map[pdg.Ref]bool{}, map[pdg.Ref]bool{}
	ast.Inspect(literal.Body, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		b.closureReference(identifier, captures, globals)
		return true
	})
	return sortedClosureSymbols(captures), sortedClosureSymbols(globals)
}

func (b *irBuilder) closureReference(identifier *ast.Ident, captures, globals map[pdg.Ref]bool) {
	variable, ok := b.info.Uses[identifier].(*types.Var)
	if !ok || variable.IsField() {
		return
	}
	if packageVariable(variable) {
		globals[b.symbol(identifier, pdg.GlobalRole)] = true
		return
	}
	ref := b.symbols[variable]
	if ref == 0 || captures[ref] {
		return
	}
	captures[ref] = true
	b.markCapturedSymbol(ref)
}

func (b *irBuilder) markCapturedSymbol(ref pdg.Ref) {
	marker := b.pool.SingletonExtension("gstyle.memory:captured", "true")[0]
	symbol := &b.graph.Symbols[ref-1]
	if !slices.Contains(symbol.Extensions, marker) {
		symbol.Extensions = append(symbol.Extensions, marker)
	}
}

func sortedClosureSymbols(symbols map[pdg.Ref]bool) []pdg.Ref {
	refs := make([]pdg.Ref, 0, len(symbols))
	for ref := range symbols {
		refs = append(refs, ref)
	}
	slices.Sort(refs)
	return refs
}

func packageVariable(variable *types.Var) bool {
	return variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope()
}
