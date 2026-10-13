package gopatterns

import (
	"encoding/json"
	"go/types"

	"github.com/rstyled/gstyle/domain/pdg"
)

const resolvedTypeExtension = "gstyle.go.scalar-normal:resolved-type"

var scalarPrimitiveTypes = map[types.BasicKind]string{
	types.Bool: "bool", types.Int: "isize", types.Int8: "i8", types.Int16: "i16", types.Int32: "i32", types.Int64: "i64",
	types.Uint: "usize", types.Uint8: "u8", types.Uint16: "u16", types.Uint32: "u32", types.Uint64: "u64", types.Uintptr: "usize",
}

// Resolved evidence comes from go/types, never from spelling substitutions.
// Named types and aliases retain their original Type fact beside this extension.
func (b *irBuilder) resolvedType(typ types.Type) []pdg.Extension {
	if typ == nil {
		return nil
	}
	basic, ok := typ.Underlying().(*types.Basic)
	if !ok {
		return nil
	}
	normalized := scalarPrimitiveTypes[basic.Kind()]
	if normalized == "" {
		return nil
	}
	value, _ := json.Marshal(struct {
		Primitive string `json:"primitive"`
	}{normalized})
	return b.pool.SingletonExtension(resolvedTypeExtension, string(value))
}
