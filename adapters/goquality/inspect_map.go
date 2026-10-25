package goquality

import (
	"go/ast"
	"go/types"
)

// Inspect completes all visits before returning and does not retain its inline
// visitor. Only direct map indexing crosses this capture boundary. The normal
// whole-use and binding proofs still check escapes and every header assignment;
// referenced children do not acquire the map's ownership.
func (x *extractor) inspectMapUse(id *ast.Ident) bool {
	if _, ok := underlying(x.pkg.TypesInfo.TypeOf(id)).(*types.Map); !ok {
		return false
	}
	node, parent := (receiverProof{parents: x.fields.parents}).parent(id)
	index, ok := parent.(*ast.IndexExpr)
	if !ok || index.X != node {
		return false
	}
	return x.inspectMapLifetime(id)
}

func (x *extractor) inspectMapLifetime(node ast.Node) bool {
	for ; node != nil; node = x.fields.parents[node] {
		switch node := node.(type) {
		case *ast.GoStmt, *ast.DeferStmt:
			return false
		case *ast.FuncLit:
			if !x.inlineInspectMap(node) {
				return false
			}
		}
	}
	return true
}

func (x *extractor) inlineInspectMap(closure *ast.FuncLit) bool {
	proof := receiverProof{pkg: x.pkg, parents: x.fields.parents}
	node, parent := proof.parent(closure)
	call, ok := parent.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 || call.Args[1] != node {
		return false
	}
	return proof.receiverInvocation(call, node)
}
