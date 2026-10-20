package govalidation

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

type bindingKey struct{ name, typ, scope string }

func Bindings(g *pdg.Graph, fn *ast.FuncDecl, fset *token.FileSet, info *types.Info) error {
	if err := pdg.Validate(g); err != nil {
		return err
	}
	objects := sourceBindings(fn, info)
	refs, err := bindSymbols(g, objects, fset)
	if err != nil {
		return err
	}
	if err := verifyRoles(g, fn, info, refs); err != nil {
		return err
	}
	return (&sourceAccesses{fset, info, refs, map[site][]pdg.Ref{}}).check(g, fn)
}

func verifyRoles(g *pdg.Graph, fn *ast.FuncDecl, info *types.Info, refs map[*types.Var]pdg.Ref) error {
	parameters := parameterBindings(fn, info)
	for variable, ref := range refs {
		expected := pdg.LocalRole
		if parameters[variable] {
			expected = pdg.ParameterRole
		}
		if g.Symbols[ref-1].Role != expected {
			return fmt.Errorf("symbol role disagrees with source binding")
		}
	}
	return nil
}

func parameterBindings(fn *ast.FuncDecl, info *types.Info) map[*types.Var]bool {
	parameters := map[*types.Var]bool{}
	for _, fields := range []*ast.FieldList{fn.Recv, fn.Type.Params} {
		if fields == nil {
			continue
		}
		for _, field := range fields.List {
			for _, name := range field.Names {
				variable, _ := info.Defs[name].(*types.Var)
				parameters[variable] = true
			}
		}
	}
	return parameters
}

func sourceBindings(fn *ast.FuncDecl, info *types.Info) map[*types.Var]bool {
	out := map[*types.Var]bool{}
	ast.Inspect(fn, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if variable, ok := info.ObjectOf(id).(*types.Var); ok {
			out[variable] = true
		}
		return true
	})
	return out
}

func variableKey(variable *types.Var, path string, fset *token.FileSet) bindingKey {
	scope := variable.Parent()
	start := fset.PositionFor(scope.Pos(), false).Offset
	end := fset.PositionFor(scope.End(), false).Offset
	return bindingKey{variable.Name(), types.TypeString(variable.Type(), func(pkg *types.Package) string { return pkg.Path() }), fmt.Sprintf("scope:%s:lexical:%d:%d", path, start, end)}
}

func bindSymbols(g *pdg.Graph, objects map[*types.Var]bool, fset *token.FileSet) (map[*types.Var]pdg.Ref, error) {
	index, err := bindingIndex(objects, g.Tables.Text(g.Sources[0].Path), fset)
	if err != nil {
		return nil, err
	}
	refs := map[*types.Var]pdg.Ref{}
	for i, symbol := range g.Symbols {
		variable, err := bindSourceSymbol(g, symbol, index, refs)
		if err != nil {
			return nil, err
		}
		refs[variable] = pdg.Ref(i + 1)
	}
	if len(refs) != len(index) {
		return nil, fmt.Errorf("source binding is missing from the symbol inventory")
	}
	return refs, nil
}

func bindSourceSymbol(g *pdg.Graph, symbol pdg.Symbol, index map[bindingKey]*types.Var, refs map[*types.Var]pdg.Ref) (*types.Var, error) {
	key := bindingKey{g.Tables.Text(symbol.Name.Value), g.Tables.Text(symbol.Type.Value), g.Tables.Text(symbol.Scope)}
	variable := index[key]
	if variable == nil {
		return nil, fmt.Errorf("symbol has no matching typed source binding")
	}
	if refs[variable] != 0 {
		return nil, fmt.Errorf("source binding is split across symbol IDs")
	}
	if err := verifiedSymbolType(g, symbol, variable); err != nil {
		return nil, err
	}
	return variable, nil
}

func verifiedSymbolType(g *pdg.Graph, symbol pdg.Symbol, variable *types.Var) error {
	if symbol.Name.Status != pdg.Known || symbol.Type.Status != pdg.Known {
		return fmt.Errorf("symbol name/type facts are not known")
	}
	return resolvedEvidence(g, symbol.Extensions, Primitive(variable.Type()))
}

func bindingIndex(objects map[*types.Var]bool, path string, fset *token.FileSet) (map[bindingKey]*types.Var, error) {
	index := map[bindingKey]*types.Var{}
	for variable := range objects {
		if variable.Name() == "_" {
			continue
		}
		if variable.Parent() == nil {
			return nil, fmt.Errorf("source binding has unknown scope")
		}
		index[variableKey(variable, path, fset)] = variable
	}
	return index, nil
}

func (a *sourceAccesses) check(g *pdg.Graph, fn *ast.FuncDecl) error {
	bySpan := map[pdg.Span][]*types.Var{}
	ast.Inspect(fn, func(node ast.Node) bool {
		id, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		variable, ok := a.info.ObjectOf(id).(*types.Var)
		if ok {
			bySpan[sourceSpan(a.fset, id)] = []*types.Var{variable}
		}
		return true
	})
	bareReturnBindings(fn, a.fset, a.info, bySpan)
	for _, node := range g.Nodes {
		if node.Category != pdg.Read {
			continue
		}
		if err := checkRead(g, node, bySpan, a.refs); err != nil {
			return err
		}
	}
	return a.checkWrites(g, fn)
}

func sourceSpan(fset *token.FileSet, node ast.Node) pdg.Span {
	return pdg.Span{Source: 1, Start: uint32(fset.PositionFor(node.Pos(), false).Offset), End: uint32(fset.PositionFor(node.End(), false).Offset)}
}

func checkRead(g *pdg.Graph, node pdg.Node, bySpan map[pdg.Span][]*types.Var, refs map[*types.Var]pdg.Ref) error {
	if node.Span == 0 {
		return fmt.Errorf("read has no source span")
	}
	variables := bySpan[g.Spans[node.Span-1]]
	if len(variables) == 0 {
		return fmt.Errorf("read has no typed source binding")
	}
	reads := g.NodeAttributes(node).Reads
	if len(reads) != 1 {
		return fmt.Errorf("read binding cardinality disagrees with source")
	}
	variable, err := readVariable(variables, reads[0].Position)
	if err != nil {
		return err
	}
	if reads[0].Symbol != refs[variable] {
		return fmt.Errorf("read binding disagrees with typed source")
	}
	return nil
}

func readVariable(variables []*types.Var, position pdg.Position) (*types.Var, error) {
	index := 0
	if len(variables) > 1 {
		index = int(position) - 1
	}
	if index < 0 || index >= len(variables) {
		return nil, fmt.Errorf("bare return read position is invalid")
	}
	return variables[index], nil
}

func bareReturnBindings(fn *ast.FuncDecl, fset *token.FileSet, info *types.Info, bySpan map[pdg.Span][]*types.Var) {
	if fn.Type.Results == nil {
		return
	}
	var results []*types.Var
	for _, field := range fn.Type.Results.List {
		for _, name := range field.Names {
			variable, _ := info.Defs[name].(*types.Var)
			results = append(results, variable)
		}
	}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		ret, ok := node.(*ast.ReturnStmt)
		if ok && len(ret.Results) == 0 {
			bySpan[sourceSpan(fset, ret)] = results
		}
		return true
	})
}
