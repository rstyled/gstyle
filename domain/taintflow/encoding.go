package taintflow

import "github.com/rstyled/gstyle/domain/pdg"

func encodeValue(item value, kind string) value {
	result := emptyValue()
	for dep := range item.Data {
		dep.SafeFor = kind
		result.Data[dep] = true
	}
	return result
}
func joinSinkDependencies(to, from dependencies, kind string) {
	for dep := range from {
		if dep.SafeFor != kind {
			to[dep] = true
		}
	}
}
func (e *evaluation) externalCall(ref pdg.Ref) value {
	positions, _ := e.query.paths.RequiredOperands(ref)
	encodings := e.callEncodings(ref)
	result := emptyValue()
	for _, position := range positions {
		item := e.externalArgument(ref, position, encodings)
		result = joinValue(result, item)
	}
	return result
}
func (e *evaluation) callEncodings(ref pdg.Ref) map[pdg.Position]string {
	result := map[pdg.Position]string{}
	if e.query.models.Encodings == nil {
		return result
	}
	for _, encoding := range e.query.models.Encodings(e.query.graph, ref) {
		result[encoding.Position] = encoding.SafeFor
	}
	return result
}
func (e *evaluation) externalArgument(ref pdg.Ref, position pdg.Position, encodings map[pdg.Position]string) value {
	item := e.input(ref, position)
	if kind := encodings[position]; kind != "" {
		return encodeValue(item, kind)
	}
	if e.query.models.PreservesEncoding != nil && e.query.models.PreservesEncoding(e.query.graph, ref) {
		return item
	}
	result := encodeValue(item, "")
	result.Opaque = true
	result.Objects = item.Objects
	return result
}

func (e *evaluation) expressionEncoding(ref pdg.Ref, item value) value {
	if converted, known := e.callableConversion(ref, item); known {
		return converted
	}
	if e.query.models.PreservesValueEncoding != nil && e.query.models.PreservesValueEncoding(e.query.graph, ref) {
		return item
	}
	result := encodeValue(item, "")
	result.Opaque = true
	result.Objects = item.Objects
	return result
}
