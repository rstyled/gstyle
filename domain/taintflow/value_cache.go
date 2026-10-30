package taintflow

import (
	"reflect"
	"unsafe"
)

type valueKey struct {
	Data      unsafe.Pointer
	Objects   unsafe.Pointer
	Functions unsafe.Pointer
	Opaque    bool
	Nil       bool
}

type unionKey struct {
	Left, Right valueKey
}

type valueUnion struct {
	Left, Right, Result value
}

type valueUnions map[unionKey]valueUnion

func (unions valueUnions) join(left, right value) value {
	if unions == nil {
		return joinValue(left, right)
	}
	key := unionKey{Left: valueIdentity(left), Right: valueIdentity(right)}
	if cached, found := unions[key]; found {
		return cached.Result
	}
	result := joinValue(left, right)
	unions[key] = valueUnion{Left: left, Right: right, Result: result}
	return result
}

func valueIdentity(item value) valueKey {
	return valueKey{
		Data:      reflect.ValueOf(item.Data).UnsafePointer(),
		Objects:   reflect.ValueOf(item.Objects).UnsafePointer(),
		Functions: reflect.ValueOf(item.Functions).UnsafePointer(),
		Opaque:    item.Opaque,
		Nil:       item.Nil,
	}
}
