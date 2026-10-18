package gopatterns

import (
	"fmt"
	"go/ast"
	"go/types"
	"maps"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

// A declaration facade reuses the sole typed graph lowering. Its name object
// lives in a private Info view; compiler syntax and the loader's Info stay intact.
func (p *packageAnalysis) closurePackage(literal *ast.FuncLit, identity string) (*packages.Package, *ast.FuncDecl, error) {
	pkg := p.pkg
	signature, ok := pkg.TypesInfo.TypeOf(literal).(*types.Signature)
	if !ok {
		return nil, nil, fmt.Errorf("closure %s has no compiler signature", identity)
	}
	name := &ast.Ident{Name: strings.TrimPrefix(identity, pkg.PkgPath+"."), NamePos: literal.Pos()}
	declaration := &ast.FuncDecl{Name: name, Type: literal.Type, Body: literal.Body}
	if p.closurePackageView == nil {
		info := *pkg.TypesInfo
		info.Defs = maps.Clone(info.Defs)
		owned := *pkg
		owned.TypesInfo = &info
		p.closurePackageView = &owned
	}
	p.closurePackageView.TypesInfo.Defs[name] = types.NewFunc(literal.Pos(), pkg.Types, name.Name, signature)
	return p.closurePackageView, declaration, nil
}

func (p *packageAnalysis) closureGraphs(parent functionGraph, filename string) ([]functionGraph, error) {
	var result []functionGraph
	for _, literal := range immediateClosures(parent.declaration.Body) {
		identity := parent.graph.Tables.Text(parent.graph.Function.ID) + ".closure@" + strconv.Itoa(p.pkg.Fset.PositionFor(literal.Pos(), false).Offset)
		pkg, declaration, err := p.closurePackage(literal, identity)
		if err != nil {
			return nil, err
		}
		source, err := p.typedFunctionGraph(pkg, declaration, parent.path, filename)
		if err != nil {
			return nil, err
		}
		source.closure = true
		nested, err := p.closureGraphs(source, filename)
		if err != nil {
			return nil, err
		}
		result = append(result, source)
		result = append(result, nested...)
	}
	return result, nil
}

func immediateClosures(body *ast.BlockStmt) []*ast.FuncLit {
	var result []*ast.FuncLit
	ast.Inspect(body, func(node ast.Node) bool {
		literal, ok := node.(*ast.FuncLit)
		if !ok {
			return true
		}
		result = append(result, literal)
		return false
	})
	return result
}
