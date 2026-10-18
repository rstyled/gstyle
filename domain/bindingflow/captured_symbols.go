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
	data := ""
	found := false
	for _, extension := range extensions {
		if q.graph.Tables.Text(extension.Name) != "gstyle.memory:captured" {
			continue
		}
		if found {
			return false, fmt.Errorf("duplicate captured-cell marker")
		}
		data = q.graph.Tables.Text(extension.Value)
		found = true
	}
	if !found {
		return false, nil
	}
	return decodeCaptureMarker(data)
}

func decodeCaptureMarker(data string) (bool, error) {
	var value *bool
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return false, err
	}
	if value == nil {
		return false, fmt.Errorf("captured-cell marker is not Boolean")
	}
	return *value, nil
}
