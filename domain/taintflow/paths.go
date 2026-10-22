package taintflow

import (
	"maps"
	"strings"
)

func pathValue(store state, base value, fields string) value {
	if fields == "" {
		return base
	}
	for _, field := range strings.Split(fields, "/") {
		base = pathComponent(store, base, field)
	}
	return base
}
func pathComponent(store state, base value, field string) value {
	if field == callbackEffect {
		return callbackValue(store, base)
	}
	if field == "*" {
		return wildcardValue(store, base)
	}
	return readPathObjects(store, base, field)
}

func readPathObjects(store state, base value, field string) value {
	result := emptyValue()
	for root := range base.Objects {
		item := readCell(store, cellKey(root, field))
		if len(base.Objects) == 1 {
			return item
		}
		result.Opaque = result.Opaque || item.Opaque
		result.Nil = result.Nil || item.Nil
		maps.Copy(result.Functions, item.Functions)
		maps.Copy(result.Data, item.Data)
		maps.Copy(result.Objects, item.Objects)
	}
	return result
}
func pathTargets(store state, base value, fields string) []cell {
	parts := strings.Split(fields, "/")
	if len(parts) > 1 {
		base = pathValue(store, base, strings.Join(parts[:len(parts)-1], "/"))
	}
	result := make([]cell, 0, len(base.Objects))
	for root := range base.Objects {
		result = append(result, cellKey(root, parts[len(parts)-1]))
	}
	return result
}
