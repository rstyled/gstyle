package gopatterns

import (
	"fmt"
	"go/types"
	"path/filepath"

	"github.com/rstyled/gstyle/domain/pdg"
)

// scope IDs use physical file offsets, independent of FileSet allocation order.
func (b *irBuilder) scope(obj types.Object) pdg.Text {
	scope := obj.Parent()
	if scope == types.Universe {
		return b.pool.Text("scope:universe")
	}
	if scope == nil {
		return b.pool.Text("scope:unbound:" + scopePackage(obj))
	}
	if !scope.Pos().IsValid() {
		return b.pool.Text("scope:package:" + scopePackage(obj))
	}
	start := b.fset.PositionFor(scope.Pos(), false)
	end := b.fset.PositionFor(scope.End(), false)
	return b.pool.Text(fmt.Sprintf("scope:%s:lexical:%d:%d", b.scopePath(start.Filename), start.Offset, end.Offset))
}

func scopePackage(obj types.Object) string {
	if obj.Pkg() == nil {
		return "universe"
	}
	return obj.Pkg().Path()
}

func (b *irBuilder) scopePath(filename string) string {
	relative, err := filepath.Rel(filepath.Dir(b.sourceFilename), filename)
	if err != nil {
		return b.pool.Tables.Text(b.graph.Sources[0].Path)
	}
	return filepath.ToSlash(filepath.Join(filepath.Dir(b.pool.Tables.Text(b.graph.Sources[0].Path)), relative))
}
