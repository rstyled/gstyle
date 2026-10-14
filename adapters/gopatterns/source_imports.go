package gopatterns

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/rstyled/gstyle/domain/pdg"
	"golang.org/x/tools/go/packages"
)

type sourceImportCollector struct {
	pkg      *packages.Package
	imports  []pdg.SourceImport
	bindings map[types.Object]int
}

func (c *sourceImportCollector) declaration(group *ast.GenDecl) {
	for _, spec := range group.Specs {
		item, ok := spec.(*ast.ImportSpec)
		if !ok {
			continue
		}
		binding := importObject(c.pkg.TypesInfo, item)
		if binding == nil {
			continue
		}
		c.bindings[binding] = len(c.imports)
		c.imports = append(c.imports, pdg.SourceImport{Name: strings.Clone(binding.Name()), Path: strings.Clone(binding.Imported().Path()), Spec: nodeByteRange(c.pkg.Fset, item), Declaration: nodeByteRange(c.pkg.Fset, group), Binding: importBindingRange(c.pkg.Fset, item), Grouped: group.Lparen.IsValid()})
	}
}

func (c *sourceImportCollector) use(node ast.Node) bool {
	ident, ok := node.(*ast.Ident)
	if !ok {
		return true
	}
	if index, ok := c.bindings[c.pkg.TypesInfo.Uses[ident]]; ok {
		c.imports[index].Uses = append(c.imports[index].Uses, nodeByteRange(c.pkg.Fset, ident))
	}
	return true
}

func sourceImports(pkg *packages.Package, file *ast.File) []pdg.SourceImport {
	collector := sourceImportCollector{pkg: pkg, bindings: map[types.Object]int{}}
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.IMPORT {
			continue
		}
		collector.declaration(group)
	}
	ast.Inspect(file, collector.use)
	return collector.imports
}

func importObject(info *types.Info, spec *ast.ImportSpec) *types.PkgName {
	object := info.Implicits[spec]
	if spec.Name != nil {
		object = info.Defs[spec.Name]
	}
	binding, _ := object.(*types.PkgName)
	return binding
}

func importBindingRange(fset *token.FileSet, spec *ast.ImportSpec) pdg.ByteRange {
	if spec.Name != nil {
		return nodeByteRange(fset, spec.Name)
	}
	start := uint32(fset.PositionFor(spec.Path.Pos(), false).Offset)
	return pdg.ByteRange{Start: start, End: start}
}
