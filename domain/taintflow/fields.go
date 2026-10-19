package taintflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"strconv"
	"strings"
)

func fieldKey(fields []uint32) string {
	parts := make([]string, len(fields))
	for index, field := range fields {
		parts[index] = strconv.FormatUint(uint64(field), 10)
	}
	return strings.Join(parts, "/")
}
func (e *evaluation) substitute(item value, call pdg.Ref) value {
	result := emptyValue()
	result.Functions, result.Opaque, result.Nil = item.Functions, item.Opaque, item.Nil
	for dep := range item.Data {
		translated := e.substituteAtom(dep, call)
		if dep.SafeFor != "" {
			translated = encodeValue(translated, dep.SafeFor)
		}
		maps.Copy(result.Data, translated.Data)
	}
	for root := range item.Objects {
		e.substituteObject(result.Objects, root, call)
	}
	return result
}
func (e *evaluation) substituteAtom(dep atom, call pdg.Ref) value {
	if dep.Input == 0 {
		return value{Data: dependencies{dep: true}, Objects: map[object]bool{}}
	}
	actual := e.actual(call, pdg.Position(dep.Input))
	if dep.Fields == "" {
		return actual
	}
	return pathValue(e.state, actual, dep.Fields)
}
func (e *evaluation) substituteObject(target map[object]bool, root object, call pdg.Ref) {
	if root.Input == 0 {
		root = e.rebaseObject(root, call)
		target[root] = true
		return
	}
	actual := e.actual(call, pdg.Position(root.Input))
	for item := range pathValue(e.state, actual, root.Fields).Objects {
		target[item] = true
	}
}
func (e *evaluation) callWrites(call pdg.Ref, summary summary) {
	projected := map[cell]value{}
	for root := range summary.Repeated {
		targets := map[object]bool{}
		e.substituteObject(targets, root, call)
		for target := range targets {
			e.state.Repeated[target] = true
		}
	}
	for key, item := range summary.Writes {
		e.projectWrite(call, key, item, projected)
	}
	for key, item := range projected {
		e.state.Memory[key] = item
	}
}
func (e *evaluation) projectWrite(call pdg.Ref, key cell, item value, projected map[cell]value) {
	targets := e.projectTargets(call, key)
	translated := e.substitute(item, call)
	for _, target := range targets {
		if key.Object.Input == 0 {
			if _, found := projected[target]; found {
				e.state.Repeated[target.Object] = true
			}
		}
		next := translated
		if e.weakTarget(key, target, targets) {
			next = joinValue(readCell(e.state, target), next)
		}
		projected[target] = joinValue(projected[target], next)
	}
}
func (e *evaluation) projectTargets(call pdg.Ref, key cell) []cell {
	if key.Object.Input == 0 {
		return []cell{{Object: e.rebaseObject(key.Object, call), Fields: key.Fields}}
	}
	actual := e.actual(call, pdg.Position(key.Object.Input))
	fields := appendFields(key.Object.Fields, key.Fields)
	if callbackMarker(fields) {
		prefix := strings.TrimSuffix(strings.TrimSuffix(fields, callbackEffect), "/")
		return e.callbackTargets(actual, prefix)
	}
	if wildcardField(fields) {
		return e.wildcardTargets(actual, fields)
	}
	return pathTargets(e.state, actual, fields)
}
func (e *evaluation) weakTarget(original, target cell, targets []cell) bool {
	if original.Object.Input == 0 {
		if _, found := e.state.Memory[target]; found {
			e.state.Repeated[target.Object] = true
		}
	}
	return len(targets) != 1 || e.state.Repeated[target.Object] || strings.Contains(target.Fields, "*")
}
func (e *evaluation) rebaseObject(root object, call pdg.Ref) object {
	if root.Input != 0 {
		return root
	}
	site, _ := json.Marshal(struct {
		Function string
		Node     pdg.Ref
	}{e.query.name, call})
	var repeated bool
	root.History, repeated = root.History.Extend(string(site))
	if repeated {
		e.state.Repeated[root] = true
	}
	return root
}
func cellKey(root object, fields string) cell {
	fields = appendFields(root.Fields, fields)
	root.Fields = ""
	return cell{Object: root, Fields: fields}
}

func (e *evaluation) wildcardTargets(base value, fields string) []cell {
	base = pathValue(e.state, base, wildcardPrefix(fields))
	reachable := reachableStore(e.state, base)
	var result []cell
	for root := range reachable.Objects {
		result = append(result, cellKey(root, "*"))
	}
	return result
}
