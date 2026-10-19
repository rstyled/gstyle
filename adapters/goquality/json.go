package goquality

import (
	"go/types"
	"slices"
)

// Reject marshal methods and open interfaces; track recursive records once.
func plainJSONType(typ types.Type, seen map[types.Type]bool) bool {
	if seen[typ] {
		return true
	}
	seen[typ] = true
	if named, ok := types.Unalias(typ).(*types.Named); ok {
		return plainJSONNamed(named, seen)
	}
	return plainJSONShape(underlying(typ), seen)
}

func plainJSONShape(typ types.Type, seen map[types.Type]bool) bool {
	switch typ := typ.(type) {
	case *types.Basic:
		return true
	case *types.Map:
		return plainJSONType(typ.Key(), seen) && plainJSONType(typ.Elem(), seen)
	case *types.Struct:
		return plainJSONFields(typ, seen)
	case interface{ Elem() types.Type }:
		return plainJSONType(typ.Elem(), seen)
	default:
		return false
	}
}

func plainJSONFields(typ *types.Struct, seen map[types.Type]bool) bool {
	for i := 0; i < typ.NumFields(); i++ {
		if !plainJSONType(typ.Field(i).Type(), seen) {
			return false
		}
	}
	return true
}

func plainJSONNamed(typ *types.Named, seen map[types.Type]bool) bool {
	for i := 0; i < typ.NumMethods(); i++ {
		if slices.Contains([]string{"MarshalJSON", "MarshalText"}, typ.Method(i).Name()) {
			return false
		}
	}
	return plainJSONType(typ.Underlying(), seen)
}
