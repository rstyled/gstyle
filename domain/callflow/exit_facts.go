package callflow

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rstyled/gstyle/domain/deferflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func exitBodyWithQueries(queries Queries) *exitBody {
	if queries.Calls == nil {
		return nil
	}
	graph := queries.Calls.graph
	machine := deferflow.NewMachineWithIndex[struct{}](queries.Deferred)
	return &exitBody{graph: graph, flow: pdg.ExitBodyFlow(graph), paths: queries.Bindings, calls: queries.Calls, machine: machine, facts: queries.Facts, operations: queries.Operations}
}

func decodeOperationEffect(value string) (operationEffect, error) {
	if !json.Valid([]byte(value)) {
		return operationEffect{}, fmt.Errorf("invalid operation effects JSON")
	}
	decoder := json.NewDecoder(strings.NewReader(value))
	token, _ := decoder.Token()
	if token != json.Delim('{') {
		return operationEffect{}, fmt.Errorf("operation effects require an object")
	}
	fields := map[string]bool{}
	for decoder.More() {
		if err := readOperationEffectField(decoder, fields); err != nil {
			return operationEffect{}, err
		}
	}
	return operationEffect{Panic: fields["panic"], NilInput: fields["nil_input"], Block: fields["block"], Unknown: fields["unknown"]}, nil
}
func readOperationEffectField(decoder *json.Decoder, fields map[string]bool) error {
	token, _ := decoder.Token()
	name := token.(string) // Valid object JSON has string keys at this position.
	if _, duplicate := fields[name]; duplicate {
		return fmt.Errorf("duplicate operation effect field: %s", name)
	}
	if !knownOperationEffectField(name) {
		return fmt.Errorf("unknown operation effect field: %s", name)
	}
	token, _ = decoder.Token()
	flag, ok := token.(bool)
	if !ok {
		return fmt.Errorf("operation effect %s requires a boolean", name)
	}
	fields[name] = flag
	return nil
}
func knownOperationEffectField(name string) bool {
	switch name {
	case "panic", "nil_input", "block", "unknown":
		return true
	default:
		return false
	}
}

func directOperationAccess(read pdg.Access) bool {
	return read.Symbol != 0 && read.Location == 0 && read.UnresolvedReason == 0
}
