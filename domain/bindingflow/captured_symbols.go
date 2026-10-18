package bindingflow

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
)

func (q *Query) capturedSymbols() (map[pdg.Ref]bool, error) {
	result := map[pdg.Ref]bool{}
	for i, symbol := range q.graph.Symbols {
		captured, err := q.captureMarker(symbol.Extensions)
		if err != nil {
			return nil, err
		}
		if captured || symbol.Role == pdg.CaptureRole {
			result[pdg.Ref(i+1)] = true
		}
	}
	return result, nil
}

func (q *Query) captureMarker(extensions []pdg.Extension) (bool, error) {
	return q.booleanSymbolMarker(extensions, "gstyle.memory:captured")
}

func (q *Query) booleanSymbolMarker(extensions []pdg.Extension, name string) (bool, error) {
	data := ""
	found := false
	for _, extension := range extensions {
		if q.graph.Tables.Text(extension.Name) != name {
			continue
		}
		if found {
			return false, fmt.Errorf("duplicate symbol marker: %s", name)
		}
		data = q.graph.Tables.Text(extension.Value)
		found = true
	}
	if !found {
		return false, nil
	}
	return decodeBooleanSymbolMarker(data)
}

func decodeBooleanSymbolMarker(data string) (bool, error) {
	var value *bool
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return false, err
	}
	if value == nil {
		return false, fmt.Errorf("symbol marker is not Boolean")
	}
	return *value, nil
}
