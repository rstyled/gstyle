package taintflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
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
	for dep := range item.Data {
		joinDependencies(result.Data, e.substituteAtom(dep, call).Data)
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
		root.Node = call
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
	for key, item := range summary.Writes {
		e.projectWrite(call, key, item, projected)
	}
	for key, item := range projected {
		e.state.Memory[key] = item
	}
}
func (e *evaluation) projectWrite(call pdg.Ref, key cell, item value, projected map[cell]value) {
	actual := e.actual(call, pdg.Position(key.Object.Input))
	translated := e.substitute(item, call)
	targets := pathTargets(e.state, actual, appendFields(key.Object.Fields, key.Fields))
	for _, target := range targets {
		next := translated
		if len(targets) != 1 || strings.Contains(target.Fields, "*") {
			next = joinValue(readCell(e.state, target), next)
		}
		projected[target] = joinValue(projected[target], next)
	}
}
func cellKey(root object, fields string) cell {
	fields = appendFields(root.Fields, fields)
	root.Fields = ""
	return cell{Object: root, Fields: fields}
}
