package gopatterns

import (
	"go/ast"
	"go/token"

	"github.com/rstyled/gstyle/domain/pdg"
)

type commentOwnerCollector struct {
	fset   *token.FileSet
	source []byte
	owners []pdg.SourceCommentOwner
}

func (c *commentOwnerCollector) add(doc *ast.CommentGroup, owner string) {
	if doc == nil {
		return
	}
	for _, comment := range doc.List {
		start := c.fset.PositionFor(comment.Pos(), false).Offset
		span := pdg.ByteRange{Start: uint32(start), End: uint32(rawCommentEnd(c.source, start))}
		c.owners = append(c.owners, pdg.SourceCommentOwner{Range: span, Owner: owner})
	}
}

func (c *commentOwnerCollector) specs(decl *ast.GenDecl) {
	for _, spec := range decl.Specs {
		switch node := spec.(type) {
		case *ast.TypeSpec:
			c.add(node.Doc, "type:"+node.Name.Name)
		case *ast.ValueSpec:
			if len(node.Names) > 0 {
				c.add(node.Doc, decl.Tok.String()+":"+node.Names[0].Name)
			}
		}
	}
}

func sourceCommentOwners(fset *token.FileSet, file *ast.File, source []byte) []pdg.SourceCommentOwner {
	collector := commentOwnerCollector{fset: fset, source: source}
	for _, decl := range file.Decls {
		switch node := decl.(type) {
		case *ast.FuncDecl:
			collector.add(node.Doc, "func:"+node.Name.Name)
		case *ast.GenDecl:
			collector.add(node.Doc, commentDeclarationOwner(node))
			collector.specs(node)
		}
	}
	return collector.owners
}

func commentDeclarationOwner(decl *ast.GenDecl) string {
	if len(decl.Specs) == 0 {
		return ""
	}
	switch spec := decl.Specs[0].(type) {
	case *ast.TypeSpec:
		return "type:" + spec.Name.Name
	case *ast.ValueSpec:
		if len(spec.Names) > 0 {
			return decl.Tok.String() + ":" + spec.Names[0].Name
		}
	}
	return decl.Tok.String()
}

func checkCommentOwners(original pdg.SourceSnapshot, current []pdg.SourceCommentOwner, content []byte) bool {
	type key struct{ text, owner string }
	counts := make(map[key]int)
	for _, owner := range current {
		counts[key{string(content[owner.Range.Start:owner.Range.End]), owner.Owner}]++
	}
	for _, owner := range original.Owners {
		entry := key{original.Content[owner.Range.Start:owner.Range.End], owner.Owner}
		if counts[entry] == 0 {
			return false
		}
		counts[entry]--
	}
	return true
}
