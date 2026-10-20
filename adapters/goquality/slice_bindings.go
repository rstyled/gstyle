package goquality

import (
	"go/ast"
	"go/types"

	"github.com/rstyled/gstyle/domain/quality"
)

type sliceWrite struct {
	place *ast.Ident
	value ast.Expr
}

func (x *extractor) bindSliceValue(object types.Object, place *ast.Ident, value ast.Expr) {
	if object == nil {
		return
	}
	if _, ok := underlying(object.Type()).(*types.Slice); ok {
		x.sliceWrites[object] = append(x.sliceWrites[object], sliceWrite{place, value})
	}
}

// Resolve the header at this use, not at all uses of the variable. The last
// write must precede the use in the same unlabeled block. Nested writes reject
// the proof; each loop iteration must establish its own reaching assignment.
func (x *extractor) localSliceValue(id *ast.Ident) ast.Expr {
	object := x.pkg.TypesInfo.ObjectOf(id)
	if len(x.sliceWrites[object]) == 0 || !x.stableCapacityObject(object) {
		return nil
	}
	current, block := x.unlabeledFieldBlock(id)
	if block == nil {
		return nil
	}
	return x.reachingSliceValue(object, current, block)
}

func (x *extractor) reachingSliceValue(object types.Object, current ast.Stmt, block *ast.BlockStmt) ast.Expr {
	var previous sliceWrite
	for _, write := range x.sliceWrites[object] {
		if write.place.Pos() < current.Pos() {
			previous = write
		}
	}
	if previous.place == nil {
		return nil
	}
	statement, assignedBlock := x.fieldStatement(previous.place, nil)
	if assignedBlock != block || statement.End() > current.Pos() {
		return nil
	}
	return previous.value
}

func (x *extractor) identifierOrigin(id *ast.Ident, seen map[types.Object]bool) quality.Provenance {
	if x.pkg.TypesInfo.Types[id].IsNil() {
		return quality.Local
	}
	if x.freshSliceStorage(id) {
		return quality.Local
	}
	return x.objectOrigin(x.pkg.TypesInfo.ObjectOf(id), seen)
}

// Follow forced append allocation and later copies or appends. Other storage
// origins keep the existing whole-function proof rather than being reevaluated.
func (x *extractor) freshSliceStorage(value ast.Expr) bool {
	switch value := ast.Unparen(value).(type) {
	case *ast.Ident:
		return x.freshSliceStorage(x.localSliceValue(value))
	case *ast.CallExpr:
		builtin, ok := x.calledObject(unwrappedCallee(value.Fun)).(*types.Builtin)
		if !ok || builtin.Name() != "append" {
			return false
		}
		return x.freshAppend(value) || x.freshSliceStorage(value.Args[0])
	default:
		return false
	}
}
