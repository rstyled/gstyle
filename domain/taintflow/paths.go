package taintflow

import "strings"

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
	if field == "*" {
		return wildcardValue(store, base)
	}
	result := emptyValue()
	for root := range base.Objects {
		result = joinValue(result, readCell(store, cellKey(root, field)))
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
