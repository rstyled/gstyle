package callflow

import (
	"encoding/json"
	"github.com/rstyled/gstyle/domain/bindingflow"
)

// These strings are owned internal encodings, never external graph input.
func encodeTuple(values []exitValue) string {
	if len(values) == 0 {
		return ""
	}
	data, _ := json.Marshal(values)
	return string(data)
}
func decodeTuple(encoded string) []exitValue {
	if encoded == "" {
		return nil
	}
	var values []exitValue
	if err := json.Unmarshal([]byte(encoded), &values); err != nil {
		panic(err)
	}
	return values
}
func knownTuple(values []exitValue) bool {
	for _, value := range values {
		if value != unknownValue {
			return true
		}
	}
	return false
}
func joinValue(left, right exitValue) exitValue {
	if left == right {
		return left
	}
	a, b := callableSet(left), callableSet(right)
	if a.Known() && b.Known() {
		return callableValue(bindingflow.NewBoundCallableSet(append(a.Targets(), b.Targets()...)...))
	}
	return unknownValue
}
