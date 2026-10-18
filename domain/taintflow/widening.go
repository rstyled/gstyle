package taintflow

import "strings"

// A widened suffix reads all reachable stored descendants. It cannot be used
// to justify a strong update or to discard an existing concrete dependency.
func wildcardValue(store state, base value) value {
	result := emptyValue()
	queue := make([]object, 0, len(base.Objects))
	for root := range base.Objects {
		queue = append(queue, root)
	}
	seen := map[object]bool{}
	for cursor := 0; cursor < len(queue); cursor++ {
		root := queue[cursor]
		if seen[root] {
			continue
		}
		seen[root] = true
		if root.Input != 0 {
			result.Data[atom{Input: root.Input, Fields: appendFields(root.Fields, "*")}] = true
		}
		queue = wildcardChildren(store, root, result.Data, queue)
	}
	return result
}
func wildcardChildren(store state, root object, deps dependencies, queue []object) []object {
	prefix := root.Fields
	root.Fields = ""
	for key, item := range store.Memory {
		if key.Object != root || !fieldPrefix(prefix, key.Fields) {
			continue
		}
		joinDependencies(deps, item.Data)
		for child := range item.Objects {
			queue = append(queue, child)
		}
	}
	return queue
}
func fieldPrefix(prefix, fields string) bool {
	return prefix == "" || prefix == fields || strings.HasPrefix(fields, prefix+"/")
}
func wildcardWrites(store state, key cell, item value) value {
	for candidate, stored := range store.Memory {
		if candidate.Object != key.Object || !strings.HasSuffix(candidate.Fields, "/*") {
			continue
		}
		if fieldPrefix(strings.TrimSuffix(candidate.Fields, "/*"), key.Fields) {
			item = joinValue(item, stored)
		}
	}
	return item
}
