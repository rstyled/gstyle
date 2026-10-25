package taintflow

import (
	"maps"
	"strings"
)

// A widened suffix reads all reachable stored descendants. It cannot be used
// to justify a strong update or to discard an existing concrete dependency.
func wildcardValue(store state, base value) value {
	result := reachableStore(store, base)
	result.Objects = map[object]bool{}
	return result
}
func reachableStore(store state, base value) value {
	result := emptyValue()
	queue := enqueueObjects(make([]object, 0, len(base.Objects)), result.Objects, base.Objects)
	for cursor := 0; cursor < len(queue); cursor++ {
		root := queue[cursor]
		if root.Input != 0 {
			result.Data[atom{Input: root.Input, Fields: appendFields(root.Fields, "*")}] = true
		}
		queue = wildcardChildren(store, root, result, queue)
	}
	return result
}

// Record discovery before enqueueing so shared descendants occupy one slot.
func enqueueObjects(queue []object, seen, objects map[object]bool) []object {
	for root := range objects {
		if seen[root] {
			continue
		}
		seen[root] = true
		queue = append(queue, root)
	}
	return queue
}
func wildcardChildren(store state, root object, result value, queue []object) []object {
	prefix := root.Fields
	root.Fields = ""
	for key, item := range store.Memory {
		if callbackMarker(key.Fields) {
			continue
		}
		if key.Object != root || !fieldPrefix(prefix, key.Fields) {
			continue
		}
		maps.Copy(result.Data, item.Data)
		queue = enqueueObjects(queue, result.Objects, item.Objects)
	}
	return queue
}
func fieldPrefix(prefix, fields string) bool {
	return prefix == "" || prefix == fields || strings.HasPrefix(fields, prefix+"/")
}
func wildcardWrites(store state, key cell, item value) value {
	if len(store.Memory) == 0 {
		return item
	}
	item = joinWildcard(store, key, "*", item)
	item = joinWildcard(store, key, "/*", item)
	for index := range key.Fields {
		if index > 0 && key.Fields[index] == '/' {
			item = joinWildcard(store, key, key.Fields[:index]+"/*", item)
		}
	}
	if key.Fields == "" {
		return item
	}
	return joinWildcard(store, key, key.Fields+"/*", item)
}

func joinWildcard(store state, key cell, fields string, item value) value {
	key.Fields = fields
	stored, found := store.Memory[key]
	if !found {
		return item
	}
	return joinValue(item, stored)
}

func wildcardField(fields string) bool { return fields == "*" || strings.HasSuffix(fields, "/*") }
func wildcardPrefix(fields string) string {
	if fields == "*" {
		return ""
	}
	return strings.TrimSuffix(fields, "/*")
}
