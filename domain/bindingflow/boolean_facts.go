package bindingflow

import (
	"encoding/json"
	"fmt"
	"github.com/rstyled/gstyle/domain/pdg"
	"strings"
)

// Boolean identity uses the compiler's resolved primitive contract. Original
// type spellings do not establish the type of named or aliased bindings.
func (q *Query) booleanSymbols() (map[pdg.Ref]bool, error) {
	result := map[pdg.Ref]bool{}
	for i, symbol := range q.graph.Symbols {
		primitive, err := q.symbolPrimitive(symbol.Extensions)
		if err != nil {
			return nil, fmt.Errorf("symbol %d primitive fact: %w", i+1, err)
		}
		if primitive == "bool" {
			result[pdg.Ref(i+1)] = true
		}
	}
	return result, nil
}

func (q *Query) symbolPrimitive(extensions []pdg.Extension) (string, error) {
	primitive := ""
	for _, extension := range extensions {
		if q.graph.Tables.Text(extension.Name) != "gstyle.go.scalar-normal:resolved-type" {
			continue
		}
		if primitive != "" {
			return "", fmt.Errorf("duplicate resolved primitive")
		}
		value, err := decodePrimitive(q.graph.Tables.Text(extension.Value))
		if err != nil {
			return "", err
		}
		primitive = value
	}
	return primitive, nil
}

func decodePrimitive(value string) (string, error) {
	if !json.Valid([]byte(value)) {
		return "", fmt.Errorf("invalid primitive JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	start, _ := decoder.Token()
	if start != json.Delim('{') || !decoder.More() {
		return "", fmt.Errorf("require one primitive field")
	}
	return decodePrimitiveField(decoder)
}

func decodePrimitiveField(decoder *json.Decoder) (string, error) {
	key, _ := decoder.Token()
	word, _ := decoder.Token()
	primitive, ok := word.(string)
	if key != "primitive" || !ok || primitive == "" || decoder.More() {
		return "", fmt.Errorf("require one nonempty primitive string")
	}
	return primitive, nil
}
