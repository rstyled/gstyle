package gopatterns

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/pdg"
	"go/ast"
	"go/types"
	"strconv"
)

// Field paths use compiler-resolved selection indices, never field spelling.
// They identify local access paths; they do not assert memory alias freedom.
func (b *irBuilder) runtimeFieldPaths(expr *ast.SelectorExpr, attrs *pdg.Attributes) {
	selection := b.info.Selections[expr]
	if selection == nil || selection.Kind() != types.FieldVal {
		return
	}
	path := b.runtimePath(expr.X)
	if path == "" {
		return
	}
	path, dereferences := runtimeSelectedPaths(selection, path)
	b.runtimePathFacts(expr, attrs, path, dereferences)
}

func (b *irBuilder) runtimePathFacts(expr *ast.SelectorExpr, attrs *pdg.Attributes, path string, dereferences []string) {
	if pointerType(b.info.TypeOf(expr)) {
		b.runtimeExtension(attrs, "field-path", strconv.Quote(path))
	}
	if len(dereferences) != 0 {
		encoded, _ := json.Marshal(dereferences)
		b.runtimeExtension(attrs, "dereference-paths", string(encoded))
	}
	if pointerType(b.info.TypeOf(expr.X)) {
		b.runtimeExtension(attrs, "receiver-pointer", "true")
	}
}

func (b *irBuilder) runtimePath(expr ast.Expr) string {
	switch expr := unparen(expr).(type) {
	case *ast.Ident:
		if ref := b.symbols[b.info.ObjectOf(expr)]; ref != 0 {
			return strconv.Itoa(int(ref))
		}
	case *ast.SelectorExpr:
		return b.runtimeSelectionPath(expr)
	}
	return ""
}

func (b *irBuilder) runtimeSelectionPath(expr *ast.SelectorExpr) string {
	selection := b.info.Selections[expr]
	if selection == nil || selection.Kind() != types.FieldVal {
		return ""
	}
	path := b.runtimePath(expr.X)
	if path == "" {
		return ""
	}
	for _, index := range selection.Index() {
		path += "/" + strconv.Itoa(index)
	}
	return path
}

func runtimeSelectedPaths(selection *types.Selection, path string) (string, []string) {
	typ := selection.Recv()
	var dereferences []string
	for _, index := range selection.Index() {
		path += "/" + strconv.Itoa(index)
		typ = runtimeStructType(typ).Field(index).Type()
		if pointerType(typ) {
			dereferences = append(dereferences, path)
		}
	}
	if pointerType(typ) {
		dereferences = dereferences[:len(dereferences)-1]
	}
	return path, dereferences
}
